package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/go-kafkify/grpc-service/proto/task/v1"
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

	// Setup gRPC server
	port := getEnv("GRPC_SERVICE_PORT", "9090")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	taskv1.RegisterTaskServiceServer(grpcServer, &taskServer{})
	
	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Register reflection for debugging
	reflection.Register(grpcServer)

	// Start server
	go func() {
		logger.Info("Starting gRPC service", zap.String("port", port))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	cancel()
	grpcServer.GracefulStop()
	logger.Info("Server exited")
}

func (s *taskServer) ProcessTask(ctx context.Context, req *taskv1.ProcessTaskRequest) (*taskv1.ProcessTaskResponse, error) {
	_, span := tracer.Start(ctx, "ProcessTask")
	defer span.End()

	taskID := uuid.New().String()
	now := time.Now()

	task := Task{
		ID:         taskID,
		ResourceID: req.ResourceId,
		Action:     req.Action,
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Start transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback()

	// Insert task
	query := `INSERT INTO tasks (id, resource_id, action, status, result, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.ExecContext(ctx, query, task.ID, task.ResourceID, task.Action, task.Status, "", task.CreatedAt, task.UpdatedAt)
	if err != nil {
		logger.Error("Failed to insert task", zap.Error(err))
		return nil, err
	}

	// Insert outbox event
	if err := insertOutboxEvent(ctx, tx, taskID, "task.process", task); err != nil {
		logger.Error("Failed to insert outbox event", zap.Error(err))
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		return nil, err
	}

	logger.Info("Task created", zap.String("task_id", taskID))

	return &taskv1.ProcessTaskResponse{
		TaskId:    taskID,
		Status:    "pending",
		CreatedAt: timestamppb.New(now),
	}, nil
}

func (s *taskServer) GetTaskStatus(ctx context.Context, req *taskv1.GetTaskStatusRequest) (*taskv1.GetTaskStatusResponse, error) {
	_, span := tracer.Start(ctx, "GetTaskStatus")
	defer span.End()

	var task Task
	query := `SELECT id, resource_id, action, status, result, created_at, updated_at FROM tasks WHERE id = $1`
	err := db.QueryRowContext(ctx, query, req.TaskId).Scan(
		&task.ID, &task.ResourceID, &task.Action, &task.Status, &task.Result, &task.CreatedAt, &task.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		logger.Error("Failed to query task", zap.Error(err))
		return nil, err
	}

	return &taskv1.GetTaskStatusResponse{
		TaskId:     task.ID,
		ResourceId: task.ResourceID,
		Action:     task.Action,
		Status:     task.Status,
		Result:     task.Result,
		CreatedAt:  timestamppb.New(task.CreatedAt),
		UpdatedAt:  timestamppb.New(task.UpdatedAt),
	}, nil
}

func (s *taskServer) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	_, span := tracer.Start(ctx, "ListTasks")
	defer span.End()

	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 50
	}

	query := `SELECT id, resource_id, action, status, created_at, updated_at FROM tasks ORDER BY created_at DESC LIMIT $1`
	rows, err := db.QueryContext(ctx, query, pageSize)
	if err != nil {
		logger.Error("Failed to query tasks", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	tasks := []*taskv1.Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ResourceID, &t.Action, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			logger.Error("Failed to scan task", zap.Error(err))
			continue
		}
		tasks = append(tasks, &taskv1.Task{
			TaskId:     t.ID,
			ResourceId: t.ResourceID,
			Action:     t.Action,
			Status:     t.Status,
			CreatedAt:  timestamppb.New(t.CreatedAt),
			UpdatedAt:  timestamppb.New(t.UpdatedAt),
		})
	}

	return &taskv1.ListTasksResponse{
		Tasks: tasks,
	}, nil
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
