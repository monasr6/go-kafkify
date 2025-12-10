.PHONY: help start stop restart logs clean build test k8s-deploy k8s-delete migrate lint

# Default target
help: ## Show this help message
	@echo "Go-Kafkify - Production Microservices Platform"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Local Development
start: ## Start all services with Docker Compose
	@chmod +x start.sh
	@./start.sh

stop: ## Stop all services
	@docker-compose down

restart: ## Restart all services
	@docker-compose restart

logs: ## Tail logs for all services
	@docker-compose logs -f

logs-rest: ## Tail logs for REST service
	@docker-compose logs -f rest-service

logs-grpc: ## Tail logs for gRPC service
	@docker-compose logs -f grpc-service

logs-worker: ## Tail logs for Python worker
	@docker-compose logs -f python-worker

clean: ## Clean up containers, volumes, and images
	@chmod +x cleanup.sh
	@./cleanup.sh

# Building
build: ## Build all Docker images
	@echo "Building REST service..."
	@docker build -t rest-service:latest ./services/rest-service
	@echo "Building gRPC service..."
	@docker build -t grpc-service:latest ./services/grpc-service
	@echo "Building Python worker..."
	@docker build -t python-worker:latest ./services/python-worker

build-rest: ## Build REST service image
	@docker build -t rest-service:latest ./services/rest-service

build-grpc: ## Build gRPC service image
	@docker build -t grpc-service:latest ./services/grpc-service

build-worker: ## Build Python worker image
	@docker build -t python-worker:latest ./services/python-worker

# Database
migrate: ## Run database migrations
	@chmod +x migrations/migrate.sh
	@./migrations/migrate.sh

db-console: ## Connect to PostgreSQL console
	@docker-compose exec postgres psql -U postgres

db-rest: ## Connect to REST service database
	@docker-compose exec postgres psql -U postgres -d restdb

db-grpc: ## Connect to gRPC service database
	@docker-compose exec postgres psql -U postgres -d grpcdb

db-worker: ## Connect to Python worker database
	@docker-compose exec postgres psql -U postgres -d workerdb

# Testing
test: ## Run all tests
	@echo "Running REST service tests..."
	@cd services/rest-service && go test ./... || true
	@echo "Running gRPC service tests..."
	@cd services/grpc-service && go test ./... || true
	@echo "Running Python worker tests..."
	@cd services/python-worker && python -m pytest || true

test-rest: ## Run REST service tests
	@cd services/rest-service && go test -v ./...

test-grpc: ## Run gRPC service tests
	@cd services/grpc-service && go test -v ./...

test-worker: ## Run Python worker tests
	@cd services/python-worker && python -m pytest -v

load-test: ## Run K6 load tests
	@k6 run load-tests/rest-api-test.js

load-test-kafka: ## Run Kafka throughput test
	@k6 run load-tests/kafka-throughput-test.js

# Kubernetes
k8s-deploy: ## Deploy to Kubernetes
	@chmod +x deploy-k8s.sh
	@./deploy-k8s.sh

k8s-delete: ## Delete from Kubernetes
	@kubectl delete namespace go-kafkify

k8s-status: ## Check Kubernetes deployment status
	@kubectl get all -n go-kafkify

k8s-logs-rest: ## View REST service logs in Kubernetes
	@kubectl logs -f -n go-kafkify -l app=rest-service

k8s-logs-grpc: ## View gRPC service logs in Kubernetes
	@kubectl logs -f -n go-kafkify -l app=grpc-service

k8s-logs-worker: ## View Python worker logs in Kubernetes
	@kubectl logs -f -n go-kafkify -l app=python-worker

k8s-port-forward: ## Port forward services from Kubernetes
	@echo "Starting port forwards..."
	@kubectl port-forward -n go-kafkify svc/rest-service 8080:8080 &
	@kubectl port-forward -n go-kafkify svc/grafana 3000:3000 &
	@kubectl port-forward -n go-kafkify svc/prometheus 9091:9090 &
	@echo "Port forwards started. Press Ctrl+C to stop."

# Monitoring
grafana: ## Open Grafana in browser
	@echo "Opening Grafana at http://localhost:3000"
	@echo "Default credentials: admin/admin"
	@open http://localhost:3000 || xdg-open http://localhost:3000 || echo "Please open http://localhost:3000"

prometheus: ## Open Prometheus in browser
	@echo "Opening Prometheus at http://localhost:9091"
	@open http://localhost:9091 || xdg-open http://localhost:9091 || echo "Please open http://localhost:9091"

metrics-rest: ## Show REST service metrics
	@curl -s http://localhost:8080/metrics | grep -E "^(http_|go_)"

metrics-grpc: ## Show gRPC service metrics
	@curl -s http://localhost:9093/metrics | grep -E "^(grpc_|go_)"

metrics-worker: ## Show Python worker metrics
	@curl -s http://localhost:9092/metrics | grep -E "^worker_"

# API Testing
api-health: ## Check REST service health
	@curl -s http://localhost:8080/health | jq .

api-create: ## Create a test resource
	@curl -X POST http://localhost:8080/api/v1/resources \
		-H "Content-Type: application/json" \
		-d '{"name":"Test Resource","description":"Created via Makefile","status":"active"}' | jq .

api-list: ## List all resources
	@curl -s http://localhost:8080/api/v1/resources | jq .

# Kafka
kafka-topics: ## List Kafka topics
	@docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

kafka-console-consumer: ## Start Kafka console consumer
	@echo "Consuming from resource.created topic..."
	@docker-compose exec kafka kafka-console-consumer \
		--bootstrap-server localhost:9092 \
		--topic resource.created \
		--from-beginning

kafka-consumer-groups: ## List Kafka consumer groups
	@docker-compose exec kafka kafka-consumer-groups --bootstrap-server localhost:9092 --list

kafka-consumer-lag: ## Check consumer lag
	@docker-compose exec kafka kafka-consumer-groups \
		--bootstrap-server localhost:9092 \
		--describe \
		--all-groups

# Code Quality
lint-go: ## Lint Go code
	@echo "Linting REST service..."
	@cd services/rest-service && golangci-lint run || echo "golangci-lint not installed"
	@echo "Linting gRPC service..."
	@cd services/grpc-service && golangci-lint run || echo "golangci-lint not installed"

lint-python: ## Lint Python code
	@echo "Linting Python worker..."
	@cd services/python-worker && pylint main.py || echo "pylint not installed"

fmt-go: ## Format Go code
	@cd services/rest-service && go fmt ./...
	@cd services/grpc-service && go fmt ./...

# Documentation
docs: ## Open documentation
	@echo "Opening documentation..."
	@cat GETTING_STARTED.md

# Quick Examples
demo: start api-create api-list ## Run a quick demo (start services and create/list resources)
	@echo ""
	@echo "Demo completed! Check Grafana at http://localhost:3000"
