# Go-Kafkify Project Structure

```
go-kafkify/
├── README.md                          # Main project documentation
├── GETTING_STARTED.md                 # Quick start guide
├── ARCHITECTURE.md                    # Detailed architecture documentation
├── LICENSE                            # MIT License
├── Makefile                           # Convenient make commands
├── .gitignore                         # Git ignore rules
├── .env.example                       # Environment variables template
├── docker-compose.yml                 # Docker Compose configuration
├── start.sh                           # Quick start script
├── deploy-k8s.sh                      # Kubernetes deployment script
├── cleanup.sh                         # Cleanup script
│
├── services/                          # Microservices
│   ├── rest-service/                  # Go REST API Service
│   │   ├── main.go                    # Main entry point
│   │   ├── outbox.go                  # Outbox pattern implementation
│   │   ├── telemetry.go               # OpenTelemetry setup
│   │   ├── go.mod                     # Go dependencies
│   │   ├── Dockerfile                 # Docker build instructions
│   │   └── README.md                  # Service documentation
│   │
│   ├── grpc-service/                  # Go gRPC Service
│   │   ├── main.go                    # Main entry point
│   │   ├── consumer.go                # Kafka consumer
│   │   ├── outbox.go                  # Outbox pattern implementation
│   │   ├── telemetry.go               # OpenTelemetry setup
│   │   ├── go.mod                     # Go dependencies
│   │   ├── Dockerfile                 # Docker build instructions
│   │   └── proto/                     # Protocol buffer definitions
│   │       └── task/
│   │           └── v1/
│   │               └── task.proto     # Task service protobuf
│   │
│   └── python-worker/                 # Python Kafka Consumer
│       ├── main.py                    # Main entry point
│       ├── requirements.txt           # Python dependencies
│       ├── Dockerfile                 # Docker build instructions
│       └── README.md                  # Service documentation
│
├── migrations/                        # Database migrations
│   ├── migrate.sh                     # Migration script
│   ├── rest-service/
│   │   ├── 001_init.up.sql            # Create tables
│   │   └── 001_init.down.sql          # Drop tables
│   ├── grpc-service/
│   │   ├── 001_init.up.sql
│   │   └── 001_init.down.sql
│   └── python-worker/
│       ├── 001_init.up.sql
│       └── 001_init.down.sql
│
├── infrastructure/                    # Infrastructure configurations
│   ├── k8s/                           # Kubernetes manifests
│   │   ├── 00-namespace.yaml          # Namespace
│   │   ├── 01-configmaps.yaml         # ConfigMaps
│   │   ├── 02-postgres.yaml           # PostgreSQL StatefulSet
│   │   ├── 03-kafka.yaml              # Kafka and Zookeeper
│   │   ├── 04-otel-collector.yaml     # OpenTelemetry Collector
│   │   ├── 05-rest-service.yaml       # REST service deployment
│   │   ├── 06-grpc-service.yaml       # gRPC service deployment
│   │   ├── 07-python-worker.yaml      # Python worker deployment
│   │   ├── 08-prometheus.yaml         # Prometheus
│   │   └── 09-grafana.yaml            # Grafana
│   │
│   └── observability/                 # Observability configurations
│       ├── otel-collector-config.yaml # OTEL Collector config
│       ├── prometheus.yml             # Prometheus config
│       └── grafana/
│           ├── datasources/
│           │   └── prometheus.yaml    # Prometheus datasource
│           └── dashboards/
│               ├── dashboard-provider.yaml
│               └── service-overview.json  # Main dashboard
│
└── load-tests/                        # K6 load testing scripts
    ├── README.md                      # Load testing documentation
    ├── rest-api-test.js               # REST API load test
    └── kafka-throughput-test.js       # Kafka throughput test
```

## Key Files Description

### Root Level

- **README.md**: Overview of the project, architecture diagram, quick links
- **GETTING_STARTED.md**: Step-by-step guide for first-time users
- **ARCHITECTURE.md**: Deep dive into architecture decisions and patterns
- **Makefile**: Convenient commands for common operations
- **docker-compose.yml**: Local development environment
- **.env.example**: Template for environment variables

### Services

Each service follows a similar structure:
- `main.go` or `main.py`: Entry point
- `Dockerfile`: Container build instructions
- `README.md`: Service-specific documentation
- Configuration files (go.mod, requirements.txt)

**REST Service (Go)**
- HTTP server with Gorilla Mux
- PostgreSQL with connection pooling
- Outbox pattern implementation
- OpenTelemetry instrumentation

**gRPC Service (Go)**
- gRPC server with Protocol Buffers
- Kafka consumer and producer
- Outbox pattern for reliable publishing
- Background workers

**Python Worker**
- Kafka consumer for multiple topics
- PostgreSQL for storing processed events
- Prometheus metrics
- Structured logging

### Migrations

SQL migration files for each service's database:
- `*.up.sql`: Create tables and indexes
- `*.down.sql`: Rollback migrations
- `migrate.sh`: Script to run all migrations

### Infrastructure

**Kubernetes Manifests:**
- Numbered for deployment order
- Includes all necessary resources (Deployments, Services, ConfigMaps, StatefulSets)
- Production-ready configurations with resource limits and health checks

**Observability:**
- OpenTelemetry Collector configuration
- Prometheus scrape configs
- Grafana datasources and dashboards

### Load Tests

K6 scripts for testing:
- REST API performance
- Kafka throughput
- Comprehensive metrics collection

## File Count Summary

```
Total Files: ~50
├── Go files: 8
├── Python files: 1
├── SQL files: 6
├── YAML files: 20
├── JavaScript files: 2
├── Markdown files: 7
├── Shell scripts: 3
└── Config files: 3
```

## Lines of Code (Approximate)

```
Go Code:        ~1,500 lines
Python Code:    ~300 lines
SQL:            ~150 lines
Kubernetes:     ~800 lines
JavaScript:     ~400 lines
Documentation:  ~2,000 lines
Total:          ~5,150 lines
```

## Technology Stack

**Backend Services:**
- Go 1.21
- Python 3.11
- Protocol Buffers

**Databases:**
- PostgreSQL 15

**Messaging:**
- Apache Kafka 7.5
- Zookeeper

**Observability:**
- OpenTelemetry
- Prometheus
- Grafana

**Container & Orchestration:**
- Docker
- Docker Compose
- Kubernetes

**Load Testing:**
- K6

## Development Workflow

1. **Local Development**: Use `docker-compose up` or `make start`
2. **Testing**: Use `make test` or run individual test suites
3. **Load Testing**: Use `make load-test`
4. **Kubernetes**: Use `make k8s-deploy`
5. **Monitoring**: Access Grafana at http://localhost:3000

## Getting Started

```bash
# Clone the repository
git clone <repo-url>
cd go-kafkify

# Copy environment template
cp .env.example .env

# Start everything
make start

# Or manually
chmod +x start.sh
./start.sh

# Run tests
make test

# Load test
make load-test

# Clean up
make clean
```

## Next Steps After Setup

1. Explore the REST API at http://localhost:8080
2. View metrics in Grafana at http://localhost:3000
3. Check Prometheus at http://localhost:9091
4. Run load tests with K6
5. Deploy to Kubernetes with `make k8s-deploy`
6. Customize for your use case

## Learning Path

1. **Start with**: GETTING_STARTED.md
2. **Understand**: ARCHITECTURE.md
3. **Explore**: Individual service README files
4. **Test**: Load testing documentation
5. **Deploy**: Kubernetes deployment guide
6. **Customize**: Modify services for your needs

## Contributing

This is a learning/reference project. Feel free to:
- Fork and modify for your needs
- Use as a template
- Learn from the patterns
- Suggest improvements

## Support

For questions or issues:
1. Check the documentation
2. Review architecture diagrams
3. Examine the code comments
4. Look at example API calls
