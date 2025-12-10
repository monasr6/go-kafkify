# Getting Started with Go-Kafkify

This guide will help you get the Go-Kafkify platform up and running.

## Prerequisites

- Docker and Docker Compose
- (Optional) Kubernetes cluster with kubectl
- (Optional) K6 for load testing

## Quick Start with Docker Compose

The fastest way to get started is using Docker Compose:

```bash
# Make scripts executable
chmod +x start.sh cleanup.sh

# Start all services
./start.sh
```

This will:
1. Build all Docker images
2. Start PostgreSQL, Kafka, and all microservices
3. Run database migrations
4. Start observability stack (Prometheus, Grafana, OpenTelemetry)

### Verify Everything is Running

```bash
# Check all containers
docker-compose ps

# Check service health
curl http://localhost:8080/health

# View logs
docker-compose logs -f rest-service
```

## Using the REST API

### Create a Resource

```bash
curl -X POST http://localhost:8080/api/v1/resources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First Resource",
    "description": "This is a test resource",
    "status": "active"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My First Resource",
  "description": "This is a test resource",
  "status": "active",
  "created_at": "2025-12-10T10:30:00Z",
  "updated_at": "2025-12-10T10:30:00Z"
}
```

### List Resources

```bash
curl http://localhost:8080/api/v1/resources
```

### Get a Specific Resource

```bash
curl http://localhost:8080/api/v1/resources/{id}
```

### Update a Resource

```bash
curl -X PUT http://localhost:8080/api/v1/resources/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Resource",
    "description": "Updated description",
    "status": "inactive"
  }'
```

### Delete a Resource

```bash
curl -X DELETE http://localhost:8080/api/v1/resources/{id}
```

## Using the gRPC Service

Install grpcurl for testing:

```bash
# macOS
brew install grpcurl

# Linux
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### Process a Task

```bash
grpcurl -plaintext -d '{
  "resource_id": "550e8400-e29b-41d4-a716-446655440000",
  "action": "process",
  "metadata": {"key": "value"}
}' localhost:9090 task.v1.TaskService/ProcessTask
```

### Get Task Status

```bash
grpcurl -plaintext -d '{
  "task_id": "task-id-here"
}' localhost:9090 task.v1.TaskService/GetTaskStatus
```

### List Tasks

```bash
grpcurl -plaintext -d '{
  "page_size": 10
}' localhost:9090 task.v1.TaskService/ListTasks
```

## Observing the System

### Grafana Dashboards

1. Open http://localhost:3000
2. Login with `admin` / `admin`
3. Navigate to Dashboards → Service Overview
4. View real-time metrics:
   - Request rates
   - Latency (p95, p99)
   - Kafka message rates
   - Database connections
   - CPU & Memory usage

### Prometheus Metrics

1. Open http://localhost:9091
2. Try these queries:
   - `http_requests_total` - Total HTTP requests
   - `http_request_duration_seconds` - Request duration
   - `worker_messages_processed_total` - Kafka messages processed
   - `go_sql_open_connections` - Database connections

### Service Logs

```bash
# REST service logs
docker-compose logs -f rest-service

# gRPC service logs
docker-compose logs -f grpc-service

# Python worker logs
docker-compose logs -f python-worker

# All service logs
docker-compose logs -f
```

## Understanding the Event Flow

1. **REST Service** receives a request to create a resource
2. **Transaction begins**: Insert into `resources` table and `outbox_events` table
3. **Transaction commits**: Both operations succeed or fail together
4. **Outbox Processor**: Background worker reads unprocessed events from outbox
5. **Kafka Publishing**: Events are published to Kafka topics
6. **gRPC Service**: Consumes events from Kafka, processes them
7. **Python Worker**: Consumes events, stores processed results
8. **Observability**: All steps are traced, metrics collected, logs structured

## Database Access

Connect to PostgreSQL:

```bash
# REST service database
docker-compose exec postgres psql -U postgres -d restdb

# gRPC service database
docker-compose exec postgres psql -U postgres -d grpcdb

# Python worker database
docker-compose exec postgres psql -U postgres -d workerdb
```

Query the outbox table:

```sql
SELECT * FROM outbox_events ORDER BY created_at DESC LIMIT 10;
```

## Kafka Topics

View Kafka topics and messages:

```bash
# List topics
docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

# Consume from a topic
docker-compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic resource.created \
  --from-beginning
```

## Load Testing

Run K6 load tests:

```bash
# Install K6 (macOS)
brew install k6

# Run REST API test
k6 run load-tests/rest-api-test.js

# Run Kafka throughput test
k6 run load-tests/kafka-throughput-test.js
```

## Kubernetes Deployment

Deploy to Kubernetes:

```bash
# Make deploy script executable
chmod +x deploy-k8s.sh

# Deploy to cluster
./deploy-k8s.sh
```

Access services:

```bash
# Port-forward REST API
kubectl port-forward -n go-kafkify svc/rest-service 8080:8080

# Port-forward Grafana
kubectl port-forward -n go-kafkify svc/grafana 3000:3000

# View logs
kubectl logs -f -n go-kafkify -l app=rest-service
```

## Troubleshooting

### Services not starting

```bash
# Check container status
docker-compose ps

# View specific service logs
docker-compose logs rest-service

# Restart a service
docker-compose restart rest-service
```

### Database connection errors

```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Verify databases exist
docker-compose exec postgres psql -U postgres -c '\l'
```

### Kafka not receiving messages

```bash
# Check Kafka is running
docker-compose ps kafka

# Check outbox processor logs
docker-compose logs rest-service | grep -i outbox

# Verify Kafka connectivity
docker-compose exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092
```

### High latency

1. Check Grafana dashboards for bottlenecks
2. View database query performance
3. Check Kafka consumer lag
4. Verify resource limits in docker-compose.yml

## Cleanup

Stop and remove all services:

```bash
./cleanup.sh

# Or manually
docker-compose down -v
```

## Next Steps

- Explore the code in `services/`
- Customize Grafana dashboards
- Add more Kafka topics and consumers
- Implement additional gRPC methods
- Add authentication and authorization
- Implement rate limiting
- Add circuit breakers
- Set up CI/CD pipelines

## Learning Resources

- [Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Kubernetes Patterns](https://kubernetes.io/docs/concepts/)
- [K6 Load Testing](https://k6.io/docs/)
