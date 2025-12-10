# Architecture Documentation

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                           │
│                                                                   │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐  │
│  │  REST API    │      │  gRPC Service│      │Python Worker │  │
│  │  (Go)        │      │  (Go)        │      │              │  │
│  │              │      │              │      │              │  │
│  │ ┌──────────┐ │      │ ┌──────────┐ │      │              │  │
│  │ │ Outbox   │ │      │ │ Outbox   │ │      │              │  │
│  │ │Processor │ │      │ │Processor │ │      │              │  │
│  │ └────┬─────┘ │      │ └────┬─────┘ │      │              │  │
│  └──────┼───────┘      └──────┼───────┘      └──────────────┘  │
│         │                     │                     ▲            │
│         │                     │                     │            │
│         ▼                     ▼                     │            │
│  ┌─────────────────────────────────────────────────┼─────────┐  │
│  │                  Kafka Broker                    │         │  │
│  │  Topics: resource.*, task.*                      │         │  │
│  └──────────────────────────────────────────────────┼─────────┘  │
│         ▲                     │                     │            │
│         │                     └─────────────────────┘            │
│         │                                                        │
│  ┌──────┴──────────────────────────────────────────────────┐   │
│  │              PostgreSQL Databases                        │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐                │   │
│  │  │ restdb  │  │ grpcdb  │  │workerdb │                │   │
│  │  └─────────┘  └─────────┘  └─────────┘                │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │           Observability Stack                             │   │
│  │  ┌──────────────┐ ┌───────────┐ ┌──────────────┐       │   │
│  │  │     OTEL     │→│Prometheus │→│   Grafana    │       │   │
│  │  │  Collector   │ │           │ │  Dashboards  │       │   │
│  │  └──────────────┘ └───────────┘ └──────────────┘       │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Component Details

### REST Service (Go)

**Responsibilities:**
- Expose RESTful API for resource management
- Implement CRUD operations
- Use Outbox Pattern for reliable event publishing
- Collect metrics and traces

**Key Features:**
- Transactional outbox for at-least-once delivery
- Background worker for event publishing
- Connection pooling for database
- OpenTelemetry instrumentation
- Graceful shutdown

**API Endpoints:**
- `POST /api/v1/resources` - Create resource
- `GET /api/v1/resources` - List resources
- `GET /api/v1/resources/{id}` - Get resource
- `PUT /api/v1/resources/{id}` - Update resource
- `DELETE /api/v1/resources/{id}` - Delete resource
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### gRPC Service (Go)

**Responsibilities:**
- Expose gRPC methods for task processing
- Consume events from Kafka
- Use Outbox Pattern for publishing task results
- Process resources asynchronously

**Key Features:**
- Kafka consumer with consumer groups
- Transactional processing
- Background outbox processor
- OpenTelemetry instrumentation
- gRPC reflection for debugging

**gRPC Methods:**
- `ProcessTask` - Create and process a task
- `GetTaskStatus` - Query task status
- `ListTasks` - List all tasks

### Python Worker Service

**Responsibilities:**
- Consume events from multiple Kafka topics
- Process events and store results
- Provide metrics endpoint

**Key Features:**
- Multi-topic Kafka consumer
- Structured JSON logging
- OpenTelemetry integration
- Prometheus metrics
- Graceful shutdown handling

**Consumed Topics:**
- `resource.created`
- `resource.updated`
- `resource.deleted`
- `task.completed`

## Data Flow

### Creating a Resource (Outbox Pattern)

```
1. Client → REST API: POST /api/v1/resources
2. REST API → DB: BEGIN TRANSACTION
3. REST API → DB: INSERT INTO resources
4. REST API → DB: INSERT INTO outbox_events
5. REST API → DB: COMMIT TRANSACTION
6. REST API → Client: 201 Created
7. Outbox Processor → DB: SELECT unprocessed events
8. Outbox Processor → Kafka: PUBLISH event
9. Outbox Processor → DB: UPDATE processed_at
10. gRPC Consumer ← Kafka: CONSUME event
11. Python Worker ← Kafka: CONSUME event
```

### Why Outbox Pattern?

The Outbox Pattern ensures reliable event publishing by:

1. **Atomicity**: Database writes and event creation happen in the same transaction
2. **No Lost Events**: Events are persisted before publishing
3. **At-Least-Once Delivery**: Failed publishes can be retried
4. **Decoupling**: Services don't directly depend on Kafka availability

### Event Processing Flow

```
REST Service             Kafka              gRPC Service        Python Worker
     │                     │                      │                    │
     ├─[resource.created]─→│                      │                    │
     │                     ├─[consume]───────────→│                    │
     │                     │                      ├─[process task]     │
     │                     │                      ├─[task.completed]──→│
     │                     │←─────────────────────┤                    │
     │                     │                      │                    │
     │                     ├─[consume]──────────────────────────────→│
     │                     │                      │         [store result]
```

## Database Schemas

### REST Service (restdb)

**resources table:**
```sql
id          VARCHAR(100) PRIMARY KEY
name        VARCHAR(255) NOT NULL
description TEXT
status      VARCHAR(50) NOT NULL
created_at  TIMESTAMP NOT NULL
updated_at  TIMESTAMP NOT NULL
```

**outbox_events table:**
```sql
id           VARCHAR(100) PRIMARY KEY
aggregate_id VARCHAR(100) NOT NULL
event_type   VARCHAR(100) NOT NULL
payload      JSONB NOT NULL
created_at   TIMESTAMP NOT NULL
processed_at TIMESTAMP NULL
```

### gRPC Service (grpcdb)

**tasks table:**
```sql
id          VARCHAR(100) PRIMARY KEY
resource_id VARCHAR(100) NOT NULL
action      VARCHAR(100) NOT NULL
status      VARCHAR(50) NOT NULL
result      TEXT
created_at  TIMESTAMP NOT NULL
updated_at  TIMESTAMP NOT NULL
```

**outbox_events table:** (same as REST service)

### Python Worker (workerdb)

**processed_events table:**
```sql
id           UUID PRIMARY KEY
event_type   VARCHAR(100) NOT NULL
resource_id  VARCHAR(100)
task_id      VARCHAR(100)
action       VARCHAR(50)
payload      JSONB NOT NULL
processed_at TIMESTAMP NOT NULL
```

## Observability

### OpenTelemetry Integration

All services export:

1. **Traces**: Distributed traces across services
   - HTTP request traces
   - gRPC call traces
   - Database query traces
   - Kafka publish/consume traces

2. **Metrics**: Application and system metrics
   - Request rates
   - Response times
   - Error rates
   - Custom business metrics

3. **Logs**: Structured logging
   - JSON format
   - Correlation IDs
   - Service context

### Prometheus Metrics

**REST Service:**
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request duration histogram
- `go_sql_open_connections` - Database connections

**gRPC Service:**
- `grpc_server_handled_total` - Total gRPC requests
- `grpc_server_handling_seconds` - gRPC call duration
- `kafka_consumer_messages_consumed` - Messages consumed

**Python Worker:**
- `worker_messages_processed_total` - Messages processed
- `worker_processing_duration_seconds` - Processing duration
- `worker_db_operations_total` - Database operations

### Grafana Dashboards

Pre-configured dashboards show:
- Service latency percentiles (p50, p95, p99)
- Request rate and throughput
- Kafka consumer lag
- Database connection pool metrics
- CPU and memory utilization
- Error rates

## Kafka Topics

### resource.created
Published by: REST Service
Consumed by: gRPC Service, Python Worker
Schema:
```json
{
  "id": "uuid",
  "name": "string",
  "description": "string",
  "status": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### resource.updated
Published by: REST Service
Consumed by: gRPC Service, Python Worker
Schema: Same as resource.created

### resource.deleted
Published by: REST Service
Consumed by: gRPC Service, Python Worker
Schema:
```json
{
  "id": "uuid",
  "status": "deleted"
}
```

### task.process
Published by: gRPC Service
Consumed by: Python Worker
Schema:
```json
{
  "task_id": "string",
  "resource_id": "string",
  "action": "string",
  "status": "string"
}
```

### task.completed
Published by: gRPC Service
Consumed by: Python Worker
Schema:
```json
{
  "task_id": "string",
  "resource_id": "string",
  "action": "string",
  "status": "completed",
  "result": "string"
}
```

## Scalability Considerations

### Horizontal Scaling

All services are stateless and can scale horizontally:

```yaml
# Kubernetes
replicas: 3  # Scale to 3 instances

# Docker Compose
docker-compose up --scale rest-service=3
```

### Database Scaling

- Connection pooling (20-50 connections per service)
- Read replicas for read-heavy workloads
- Database indexes on frequently queried columns

### Kafka Scaling

- Multiple partitions per topic for parallelism
- Consumer groups for load distribution
- Partition key = resource ID for ordering

### Caching Strategy

Not implemented but recommended:
- Redis for frequently accessed resources
- Cache-aside pattern
- TTL-based invalidation

## Security Considerations

**Current State** (Development):
- No authentication/authorization
- Plain text secrets in ConfigMaps
- No TLS/SSL encryption

**Production Recommendations:**
1. Add OAuth2/JWT authentication
2. Use Kubernetes Secrets for sensitive data
3. Enable TLS for all services
4. Implement rate limiting
5. Add API gateway
6. Network policies in Kubernetes
7. Database encryption at rest
8. Kafka authentication (SASL/SSL)

## Deployment Patterns

### Blue-Green Deployment
```bash
# Deploy new version as "green"
kubectl apply -f deployment-green.yaml

# Test green deployment
kubectl port-forward svc/rest-service-green 8080:8080

# Switch traffic
kubectl patch service rest-service -p '{"spec":{"selector":{"version":"green"}}}'
```

### Canary Deployment
```yaml
# 90% to stable, 10% to canary
apiVersion: v1
kind: Service
metadata:
  name: rest-service
spec:
  selector:
    app: rest-service
    # No version selector - routes to all pods
```

## Monitoring and Alerting

Recommended alerts:

1. **High Error Rate**: Error rate > 5% for 5 minutes
2. **High Latency**: P95 latency > 500ms
3. **Kafka Consumer Lag**: Lag > 10000 messages
4. **Database Connections**: Open connections > 80% of max
5. **Pod Restarts**: More than 3 restarts in 10 minutes
6. **Low Throughput**: Throughput drops by 50%

## Disaster Recovery

### Backup Strategy

- PostgreSQL: Daily full backups, continuous WAL archiving
- Kafka: Replicate to secondary cluster (MirrorMaker)
- Configuration: Store in Git

### Recovery Procedures

1. **Database Loss**: Restore from backup, replay Kafka events
2. **Kafka Loss**: Rebuild from database (if needed)
3. **Service Loss**: Redeploy from container registry

## Future Enhancements

1. **Circuit Breakers**: Implement Hystrix or similar
2. **Service Mesh**: Add Istio for advanced traffic management
3. **GraphQL API**: Add GraphQL gateway
4. **Event Sourcing**: Full event-sourced architecture
5. **CQRS**: Separate read and write models
6. **Schema Registry**: Add Confluent Schema Registry
7. **Saga Pattern**: Distributed transactions
8. **Multi-Region**: Deploy across multiple regions
