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

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	db     *sql.DB
	logger *zap.Logger
	tracer trace.Tracer
)

type Task struct {
	ID         string    `json:"id"`
	ResourceID string    `json:"resource_id"`
	Action     string    `json:"action"`
	Status     string    `json:"status"`
	Result     string    `json:"result"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type OutboxEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	EventType   string    `json:"event_type"`
	Payload     string    `json:"payload"`
	CreatedAt   time.Time `json:"created_at"`
}

type taskServer struct {
	taskv1.UnimplementedTaskServiceServer
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

	tracer = otel.Tracer("grpc-service")

	// Initialize database
	db, err = initDB()
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Start background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go startKafkaConsumer(ctx)
	go startOutboxProcessor(ctx)

	// Start metrics server
	go startMetricsServer()
	
	// Start HTTP health check server
	go startHealthServer()

	logger.Info("gRPC service started (metrics and background workers only)")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	cancel()
	logger.Info("Server exited")
}

func startHealthServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	
	healthPort := "8081"
	logger.Info("Starting health server", zap.String("port", healthPort))
	if err := http.ListenAndServe(":"+healthPort, nil); err != nil {
		logger.Error("Health server failed", zap.Error(err))
	}
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

func initDB() (*sql.DB, error) {
	dbHost := getEnv("GRPC_DB_HOST", "localhost")
	dbPort := getEnv("GRPC_DB_PORT", "5432")
	dbUser := getEnv("GRPC_DB_USER", "postgres")
	dbPassword := getEnv("GRPC_DB_PASSWORD", "postgres")
	dbName := getEnv("GRPC_DB_NAME", "grpcdb")
	sslMode := getEnv("GRPC_DB_SSLMODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

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

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	metricsPort := getEnv("METRICS_PORT", "9091")
	logger.Info("Starting metrics server", zap.String("port", metricsPort))
	if err := http.ListenAndServe(":"+metricsPort, nil); err != nil {
		logger.Error("Metrics server failed", zap.Error(err))
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
