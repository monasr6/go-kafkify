package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func startKafkaConsumer(ctx context.Context) {
	logger.Info("Starting Kafka consumer")

	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	consumerGroup := getEnv("KAFKA_CONSUMER_GROUP_GRPC", "grpc-service-group")

	// Topics to consume
	topics := []string{
		"resource.created",
		"resource.updated",
		"resource.deleted",
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaBrokers},
		GroupID:        consumerGroup,
		GroupTopics:    topics,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer reader.Close()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Kafka consumer")
			return
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				logger.Error("Failed to fetch message", zap.Error(err))
				continue
			}

			if err := processKafkaMessage(ctx, msg); err != nil {
				logger.Error("Failed to process message", 
					zap.String("topic", msg.Topic),
					zap.String("key", string(msg.Key)),
					zap.Error(err))
			} else {
				if err := reader.CommitMessages(ctx, msg); err != nil {
					logger.Error("Failed to commit message", zap.Error(err))
				}
			}
		}
	}
}

func processKafkaMessage(ctx context.Context, msg kafka.Message) error {
	_, span := tracer.Start(ctx, "processKafkaMessage")
	defer span.End()

	logger.Info("Processing Kafka message",
		zap.String("topic", msg.Topic),
		zap.String("key", string(msg.Key)),
		zap.Int64("offset", msg.Offset),
		zap.Int("partition", msg.Partition))

	// Parse the message
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Extract resource ID
	resourceID, ok := payload["id"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid resource ID in payload")
	}

	// Determine action based on topic
	var action string
	switch msg.Topic {
	case "resource.created":
		action = "process_new_resource"
	case "resource.updated":
		action = "reprocess_resource"
	case "resource.deleted":
		action = "cleanup_resource"
	default:
		action = "unknown"
	}

	// Create a task for processing
	taskID := fmt.Sprintf("auto-%s", resourceID)
	now := time.Now()

	// Start transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert or update task
	query := `INSERT INTO tasks (id, resource_id, action, status, result, created_at, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  ON CONFLICT (id) DO UPDATE SET
			  	action = EXCLUDED.action,
			  	status = 'processing',
			  	updated_at = EXCLUDED.updated_at`
	
	result := fmt.Sprintf("Processing %s event for resource %s", msg.Topic, resourceID)
	_, err = tx.ExecContext(ctx, query, taskID, resourceID, action, "processing", result, now, now)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Update task status to completed
	updateQuery := `UPDATE tasks SET status = $1, result = $2, updated_at = $3 WHERE id = $4`
	completedResult := fmt.Sprintf("Completed %s for resource %s", action, resourceID)
	_, err = tx.ExecContext(ctx, updateQuery, "completed", completedResult, time.Now(), taskID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Insert outbox event for task completion
	taskData := map[string]string{
		"task_id":     taskID,
		"resource_id": resourceID,
		"action":      action,
		"status":      "completed",
	}
	if err := insertOutboxEvent(ctx, tx, taskID, "task.completed", taskData); err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info("Task processed successfully", zap.String("task_id", taskID))
	return nil
}
