# REST Service

Production-ready REST API service with Outbox Pattern for reliable event publishing.

## Features

- RESTful API for resource management (CRUD operations)
- PostgreSQL database with connection pooling
- Outbox Pattern for transactional event publishing
- Kafka producer for async messaging
- OpenTelemetry instrumentation (traces, metrics, logs)
- Prometheus metrics endpoint
- Health check endpoint
- Graceful shutdown

## API Endpoints

### Resources

- `POST /api/v1/resources` - Create a new resource
- `GET /api/v1/resources` - List all resources
- `GET /api/v1/resources/{id}` - Get a specific resource
- `PUT /api/v1/resources/{id}` - Update a resource
- `DELETE /api/v1/resources/{id}` - Delete a resource

### Monitoring

- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## Environment Variables

```bash
REST_SERVICE_PORT=8080
REST_DB_HOST=localhost
REST_DB_PORT=5432
REST_DB_USER=postgres
REST_DB_PASSWORD=postgres
REST_DB_NAME=restdb
REST_DB_SSLMODE=disable
KAFKA_BROKERS=localhost:9092
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
LOG_LEVEL=info
```

## Running Locally

```bash
# Install dependencies
go mod download

# Run migrations
psql -U postgres -d restdb -f ../../migrations/rest-service/001_init.up.sql

# Run service
go run .
```

## Docker

```bash
# Build image
docker build -t rest-service:latest .

# Run container
docker run -p 8080:8080 --env-file .env rest-service:latest
```

## Testing

```bash
# Create resource
curl -X POST http://localhost:8080/api/v1/resources \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Resource","description":"A test resource","status":"active"}'

# List resources
curl http://localhost:8080/api/v1/resources

# Get resource
curl http://localhost:8080/api/v1/resources/{id}

# Update resource
curl -X PUT http://localhost:8080/api/v1/resources/{id} \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Resource","description":"Updated description","status":"inactive"}'

# Delete resource
curl -X DELETE http://localhost:8080/api/v1/resources/{id}

# Health check
curl http://localhost:8080/health
```
