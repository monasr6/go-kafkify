# Go-Kafkify: Microservices Platform

A complete microservices architecture demonstrating event-driven patterns, observability, and cloud-native deployment on Kubernetes.

## 🏗️ Architecture Overview

This system implements a complete event-driven microservices architecture with:

- **Kubernetes Cluster** hosting all components
- **Two Go Services**:
  1. **REST Service** - Handles requests via RESTful APIs
  2. **gRPC Service** - Handles requests via gRPC
- **Python Worker Service** - Processes Kafka events asynchronously
- **Kafka Message Queue** - Services communicate asynchronously via Kafka
- **Outbox Pattern** - Transactional event publishing using outbox tables
- **OpenTelemetry** - Distributed tracing, metrics, and logs
- **Prometheus & Grafana** - Metrics collection and visualization
- **K6 Load Testing** - Performance and stress testing

## 📁 Project Structure

```
go-kafkify/
├── services/
│   ├── rest-service/          # Go REST API service
│   ├── grpc-service/          # Go gRPC service
│   └── python-worker/         # Python Kafka consumer
├── infrastructure/
│   ├── k8s/                   # Kubernetes manifests
│   ├── helm/                  # Helm charts
│   ├── observability/         # Prometheus, Grafana configs
│   └── kafka/                 # Kafka configurations
├── migrations/                # Database migrations
├── load-tests/                # K6 load testing scripts
├── docker-compose.yml         # Local development
└── README.md
```

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Kubernetes cluster (minikube, kind, or cloud provider)
- kubectl
- helm (optional)
- k6 (for load testing)

### Local Development

```bash
# Start all services locally
docker-compose up -d

# Check service health
curl http://localhost:8080/health
grpcurl -plaintext localhost:9090 health.v1.HealthService/Check

# View logs
docker-compose logs -f rest-service
```

### Kubernetes Deployment

```bash
# Apply Kubernetes manifests
kubectl apply -f infrastructure/k8s/namespace.yaml
kubectl apply -f infrastructure/k8s/

# Or use Helm
helm install go-kafkify infrastructure/helm/go-kafkify

# Check deployment status
kubectl get pods -n go-kafkify
```

### Load Testing

```bash
# Run K6 load tests
k6 run load-tests/rest-api-test.js
k6 run load-tests/kafka-throughput-test.js
```

## 🧩 Services

### 1. REST Service (Go)

- **Port**: 8080
- **Language**: Go
- **Database**: PostgreSQL
- **Pattern**: Outbox Pattern for event publishing
- **Endpoints**:
  - `POST /api/v1/resources` - Create resource
  - `GET /api/v1/resources` - List resources
  - `GET /api/v1/resources/{id}` - Get resource
  - `PUT /api/v1/resources/{id}` - Update resource
  - `DELETE /api/v1/resources/{id}` - Delete resource
  - `GET /health` - Health check
  - `GET /metrics` - Prometheus metrics

### 2. gRPC Service (Go)

- **Port**: 9090
- **Language**: Go
- **Database**: PostgreSQL
- **Pattern**: Kafka consumer + Outbox producer
- **Methods**:
  - `ProcessTask` - Process tasks
  - `GetTaskStatus` - Query task status
  - Health checks

### 3. Python Worker Service

- **Language**: Python
- **Database**: PostgreSQL
- **Pattern**: Kafka consumer
- **Function**: Processes events from Kafka topics and stores results

## 📊 Observability

### OpenTelemetry

All services export:
- **Traces** to OTEL Collector
- **Metrics** to Prometheus
- **Logs** structured with correlation IDs

### Grafana Dashboards

Access Grafana at `http://localhost:3000` (default credentials: admin/admin)

Pre-configured dashboards:
- Service Latency & Throughput
- Kafka Consumer Lag
- Database Connection Pool
- Resource Utilization (CPU/Memory)
- Request Rate & Error Rate

### Prometheus

Access Prometheus at `http://localhost:9091`

Metrics collected:
- HTTP request duration
- gRPC call duration
- Kafka consumer lag
- Database query duration
- Outbox processing latency

## 🗄️ Database Schema

### REST Service Database

- `resources` - Main resource table
- `outbox_events` - Outbox pattern for event publishing

### gRPC Service Database

- `tasks` - Task processing table
- `outbox_events` - Outbox pattern for event publishing

### Python Worker Database

- `processed_events` - Processed event results

## 📨 Kafka Topics

- `resource.created` - Resource creation events
- `resource.updated` - Resource update events
- `resource.deleted` - Resource deletion events
- `task.process` - Task processing events
- `task.completed` - Task completion events

## 🔧 Configuration

Environment variables are managed through:
- `.env` files for local development
- ConfigMaps and Secrets for Kubernetes

Key configuration:
- Database connections
- Kafka brokers
- OTEL collector endpoint
- Service ports

## 🧪 Testing

```bash
# Unit tests
cd services/rest-service && go test ./...
cd services/grpc-service && go test ./...
cd services/python-worker && pytest

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit

# Load tests
k6 run --vus 50 --duration 60s load-tests/rest-api-test.js
```

## 📈 Performance Benchmarks

Expected performance (on standard 3-node K8s cluster):

- REST API: ~10,000 req/s with p95 latency < 50ms
- Kafka throughput: ~100,000 msg/s
- gRPC: ~15,000 req/s with p95 latency < 30ms
- Database: Connection pooling with 20-50 connections per service

## 🛠️ Development

### Adding a New Service

1. Create service directory under `services/`
2. Add Dockerfile
3. Update docker-compose.yml
4. Create Kubernetes manifests
5. Add observability instrumentation
6. Update this README

### Database Migrations

```bash
# Create new migration
migrate create -ext sql -dir migrations/rest-service -seq add_new_table

# Run migrations
migrate -path migrations/rest-service -database "postgresql://user:pass@localhost:5432/restdb?sslmode=disable" up
```

## 📝 License

MIT License - See LICENSE file for details

## 🤝 Contributing

This is a learning project demonstrating production-ready microservices patterns.
Feel free to use it as a reference or template for your own projects.

## 📚 Resources

- [Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
- [OpenTelemetry](https://opentelemetry.io/)
- [Kafka Documentation](https://kafka.apache.org/documentation/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
