# Python Worker Service

Production-ready Python Kafka consumer service that processes events and stores results in PostgreSQL.

## Features

- Kafka consumer for multiple topics
- PostgreSQL database integration
- OpenTelemetry instrumentation (traces, metrics, logs)
- Prometheus metrics endpoint
- Structured JSON logging
- Graceful shutdown
- Error handling and retries

## Kafka Topics Consumed

- `task.completed` - Task completion events from gRPC service
- `resource.created` - Resource creation events from REST service
- `resource.updated` - Resource update events from REST service
- `resource.deleted` - Resource deletion events from REST service

## Environment Variables

```bash
WORKER_DB_HOST=localhost
WORKER_DB_PORT=5432
WORKER_DB_USER=postgres
WORKER_DB_PASSWORD=postgres
WORKER_DB_NAME=workerdb
KAFKA_BROKERS=localhost:9092
KAFKA_CONSUMER_GROUP_WORKER=python-worker-group
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
METRICS_PORT=9092
```

## Running Locally

```bash
# Install dependencies
pip install -r requirements.txt

# Run migrations
psql -U postgres -d workerdb -f ../../migrations/python-worker/001_init.up.sql

# Run service
python main.py
```

## Docker

```bash
# Build image
docker build -t python-worker:latest .

# Run container
docker run --env-file .env python-worker:latest
```

## Database Schema

The service stores processed events in the `processed_events` table:

```sql
CREATE TABLE processed_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100),
    task_id VARCHAR(100),
    action VARCHAR(50),
    payload JSONB NOT NULL,
    processed_at TIMESTAMP NOT NULL
);
```

## Metrics

Available at `http://localhost:9092/metrics`:

- `worker_messages_processed_total` - Total messages processed by topic and status
- `worker_processing_duration_seconds` - Message processing duration histogram
- `worker_db_operations_total` - Total database operations by operation and status

## Logging

All logs are structured JSON format with fields:
- `timestamp` - ISO 8601 timestamp
- `level` - Log level (INFO, WARNING, ERROR)
- `message` - Log message
- Additional context fields

## Error Handling

- Database connection retries (30 attempts with 2s delay)
- Message processing errors are logged but don't stop the consumer
- Failed messages are marked in metrics
- Graceful shutdown on SIGINT/SIGTERM
