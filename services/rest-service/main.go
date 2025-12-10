package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	db     *sql.DB
	logger *zap.Logger
	tracer trace.Tracer
)

type Resource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OutboxEvent struct {
	ID           string    `json:"id"`
	AggregateID  string    `json:"aggregate_id"`
	EventType    string    `json:"event_type"`
	Payload      string    `json:"payload"`
	CreatedAt    time.Time `json:"created_at"`
	ProcessedAt  *time.Time `json:"processed_at"`
}

func main() {
	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize OpenTelemetry
	shutdown, err := initTracer()
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer shutdown(context.Background())

	tracer = otel.Tracer("rest-service")

	// Initialize database
	db, err = initDB()
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Start outbox processor in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go startOutboxProcessor(ctx)

	// Setup HTTP router
	router := mux.NewRouter()
	router.Use(otelmux.Middleware("rest-service"))
	router.Use(loggingMiddleware)

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/resources", createResourceHandler).Methods("POST")
	apiRouter.HandleFunc("/resources", listResourcesHandler).Methods("GET")
	apiRouter.HandleFunc("/resources/{id}", getResourceHandler).Methods("GET")
	apiRouter.HandleFunc("/resources/{id}", updateResourceHandler).Methods("PUT")
	apiRouter.HandleFunc("/resources/{id}", deleteResourceHandler).Methods("DELETE")

	// Health and metrics
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Start server
	port := getEnv("REST_SERVICE_PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Starting REST service", zap.String("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	cancel() // Stop outbox processor

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initDB() (*sql.DB, error) {
	dbHost := getEnv("REST_DB_HOST", "localhost")
	dbPort := getEnv("REST_DB_PORT", "5432")
	dbUser := getEnv("REST_DB_USER", "postgres")
	dbPassword := getEnv("REST_DB_PASSWORD", "postgres")
	dbName := getEnv("REST_DB_NAME", "restdb")
	sslMode := getEnv("REST_DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Wait for database to be ready
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			logger.Info("Database connection established")
			return db, nil
		}
		logger.Info("Waiting for database...", zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to database after 30 attempts")
}

func createResourceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "createResource")
	defer span.End()

	var resource Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		logger.Error("Failed to decode request", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resource.ID = uuid.New().String()
	resource.Status = "active"
	resource.CreatedAt = time.Now()
	resource.UpdatedAt = time.Now()

	// Start transaction for outbox pattern
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to begin transaction", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert resource
	query := `INSERT INTO resources (id, name, description, status, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(ctx, query, resource.ID, resource.Name, resource.Description, 
		resource.Status, resource.CreatedAt, resource.UpdatedAt)
	if err != nil {
		logger.Error("Failed to insert resource", zap.Error(err))
		http.Error(w, "Failed to create resource", http.StatusInternalServerError)
		return
	}

	// Insert outbox event
	if err := insertOutboxEvent(ctx, tx, resource.ID, "resource.created", resource); err != nil {
		logger.Error("Failed to insert outbox event", zap.Error(err))
		http.Error(w, "Failed to create resource", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		http.Error(w, "Failed to create resource", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("resource.id", resource.ID))
	logger.Info("Resource created", zap.String("id", resource.ID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resource)
}

func listResourcesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "listResources")
	defer span.End()

	query := `SELECT id, name, description, status, created_at, updated_at FROM resources ORDER BY created_at DESC LIMIT 100`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logger.Error("Failed to query resources", zap.Error(err))
		http.Error(w, "Failed to list resources", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	resources := []Resource{}
	for rows.Next() {
		var r Resource
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			logger.Error("Failed to scan resource", zap.Error(err))
			continue
		}
		resources = append(resources, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

func getResourceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "getResource")
	defer span.End()

	vars := mux.Vars(r)
	id := vars["id"]

	var resource Resource
	query := `SELECT id, name, description, status, created_at, updated_at FROM resources WHERE id = $1`
	err := db.QueryRowContext(ctx, query, id).Scan(
		&resource.ID, &resource.Name, &resource.Description, &resource.Status, 
		&resource.CreatedAt, &resource.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logger.Error("Failed to query resource", zap.Error(err))
		http.Error(w, "Failed to get resource", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("resource.id", id))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resource)
}

func updateResourceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "updateResource")
	defer span.End()

	vars := mux.Vars(r)
	id := vars["id"]

	var update Resource
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	update.ID = id
	update.UpdatedAt = time.Now()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to begin transaction", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	query := `UPDATE resources SET name = $1, description = $2, status = $3, updated_at = $4 WHERE id = $5`
	result, err := tx.ExecContext(ctx, query, update.Name, update.Description, update.Status, update.UpdatedAt, id)
	if err != nil {
		logger.Error("Failed to update resource", zap.Error(err))
		http.Error(w, "Failed to update resource", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	if err := insertOutboxEvent(ctx, tx, id, "resource.updated", update); err != nil {
		logger.Error("Failed to insert outbox event", zap.Error(err))
		http.Error(w, "Failed to update resource", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		http.Error(w, "Failed to update resource", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("resource.id", id))
	logger.Info("Resource updated", zap.String("id", id))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(update)
}

func deleteResourceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, span := tracer.Start(ctx, "deleteResource")
	defer span.End()

	vars := mux.Vars(r)
	id := vars["id"]

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to begin transaction", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	query := `DELETE FROM resources WHERE id = $1`
	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		logger.Error("Failed to delete resource", zap.Error(err))
		http.Error(w, "Failed to delete resource", http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	eventPayload := map[string]string{"id": id, "status": "deleted"}
	if err := insertOutboxEvent(ctx, tx, id, "resource.deleted", eventPayload); err != nil {
		logger.Error("Failed to insert outbox event", zap.Error(err))
		http.Error(w, "Failed to delete resource", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		http.Error(w, "Failed to delete resource", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("resource.id", id))
	logger.Info("Resource deleted", zap.String("id", id))

	w.WriteHeader(http.StatusNoContent)
}

func insertOutboxEvent(ctx context.Context, tx *sql.Tx, aggregateID, eventType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	eventID := uuid.New().String()
	query := `INSERT INTO outbox_events (id, aggregate_id, event_type, payload, created_at)
			  VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.ExecContext(ctx, query, eventID, aggregateID, eventType, string(payloadJSON), time.Now())
	return err
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("Request processed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Duration("duration", time.Since(start)))
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
