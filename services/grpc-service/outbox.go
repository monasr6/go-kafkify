package main

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func startOutboxProcessor(ctx context.Context) {
	logger.Info("Starting outbox processor")

	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping outbox processor")
			return
		case <-ticker.C:
			if err := processOutboxEvents(ctx, kafkaBrokers); err != nil {
				logger.Error("Failed to process outbox events", zap.Error(err))
			}
		}
	}
}

func processOutboxEvents(ctx context.Context, kafkaBrokers string) error {
	_, span := tracer.Start(ctx, "processOutboxEvents")
	defer span.End()

	query := `SELECT id, aggregate_id, event_type, payload, created_at 
			  FROM outbox_events 
			  WHERE processed_at IS NULL 
			  ORDER BY created_at ASC 
			  LIMIT 100`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query outbox events: %w", err)
	}
	defer rows.Close()

	events := []OutboxEvent{}
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(&event.ID, &event.AggregateID, &event.EventType, &event.Payload, &event.CreatedAt); err != nil {
			logger.Error("Failed to scan outbox event", zap.Error(err))
			continue
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		return nil
	}

	logger.Info("Processing outbox events", zap.Int("count", len(events)))

	for _, event := range events {
		if err := publishToKafka(ctx, kafkaBrokers, event); err != nil {
			logger.Error("Failed to publish event to Kafka",
				zap.String("event_id", event.ID),
				zap.Error(err))
			continue
		}

		if err := markEventProcessed(ctx, event.ID); err != nil {
			logger.Error("Failed to mark event as processed",
				zap.String("event_id", event.ID),
				zap.Error(err))
		}
	}

	return nil
}

func publishToKafka(ctx context.Context, brokers string, event OutboxEvent) error {
	_, span := tracer.Start(ctx, "publishToKafka")
	defer span.End()

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{brokers},
		Topic:        event.EventType,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	})
	defer writer.Close()

	message := kafka.Message{
		Key:   []byte(event.AggregateID),
		Value: []byte(event.Payload),
		Headers: []kafka.Header{
			{Key: "event_id", Value: []byte(event.ID)},
			{Key: "event_type", Value: []byte(event.EventType)},
		},
	}

	err := writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	logger.Info("Event published to Kafka",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.EventType),
		zap.String("topic", event.EventType))

	return nil
}

func markEventProcessed(ctx context.Context, eventID string) error {
	query := `UPDATE outbox_events SET processed_at = $1 WHERE id = $2`
	_, err := db.ExecContext(ctx, query, time.Now(), eventID)
	return err
}
