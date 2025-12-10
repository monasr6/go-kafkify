#!/bin/bash
# Deploy to Kubernetes

set -e

echo "🚀 Deploying Go-Kafkify to Kubernetes..."
echo ""

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "❌ Error: kubectl is not installed."
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Error: Cannot connect to Kubernetes cluster."
    echo "Please ensure your cluster is running and kubeconfig is set correctly."
    exit 1
fi

echo "✅ Connected to Kubernetes cluster"
echo ""

# Build Docker images
echo "🔨 Building Docker images..."
echo "Building REST service..."
docker build -t rest-service:latest ./services/rest-service

echo "Building gRPC service..."
docker build -t grpc-service:latest ./services/grpc-service

echo "Building Python worker..."
docker build -t python-worker:latest ./services/python-worker

echo "✅ Docker images built"
echo ""

# Apply Kubernetes manifests
echo "📦 Deploying to Kubernetes..."

kubectl apply -f infrastructure/k8s/00-namespace.yaml
echo "✅ Namespace created"

kubectl apply -f infrastructure/k8s/01-configmaps.yaml
echo "✅ ConfigMaps created"

kubectl apply -f infrastructure/k8s/02-postgres.yaml
echo "✅ PostgreSQL deployed"

kubectl apply -f infrastructure/k8s/03-kafka.yaml
echo "✅ Kafka deployed"

kubectl apply -f infrastructure/k8s/04-otel-collector.yaml
echo "✅ OpenTelemetry Collector deployed"

kubectl apply -f infrastructure/k8s/08-prometheus.yaml
echo "✅ Prometheus deployed"

kubectl apply -f infrastructure/k8s/09-grafana.yaml
echo "✅ Grafana deployed"

# Wait for infrastructure to be ready
echo ""
echo "⏳ Waiting for infrastructure to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n go-kafkify --timeout=120s
kubectl wait --for=condition=ready pod -l app=kafka -n go-kafkify --timeout=120s
echo "✅ Infrastructure is ready"

# Run database migrations
echo ""
echo "🗄️  Running database migrations..."
kubectl run -n go-kafkify db-migrations --image=postgres:15-alpine --rm -i --restart=Never --env="PGPASSWORD=postgres" -- sh -c "
  echo 'Creating databases...' &&
  psql -h postgres -U postgres -c 'CREATE DATABASE IF NOT EXISTS restdb;' &&
  psql -h postgres -U postgres -c 'CREATE DATABASE IF NOT EXISTS grpcdb;' &&
  psql -h postgres -U postgres -c 'CREATE DATABASE IF NOT EXISTS workerdb;' &&
  echo 'Databases created!'
"

# Deploy application services
echo ""
echo "🚀 Deploying application services..."

kubectl apply -f infrastructure/k8s/05-rest-service.yaml
echo "✅ REST service deployed"

kubectl apply -f infrastructure/k8s/06-grpc-service.yaml
echo "✅ gRPC service deployed"

kubectl apply -f infrastructure/k8s/07-python-worker.yaml
echo "✅ Python worker deployed"

echo ""
echo "⏳ Waiting for services to be ready..."
kubectl wait --for=condition=available deployment/rest-service -n go-kafkify --timeout=120s
kubectl wait --for=condition=available deployment/grpc-service -n go-kafkify --timeout=120s
kubectl wait --for=condition=available deployment/python-worker -n go-kafkify --timeout=120s

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "🎉 Go-Kafkify Successfully Deployed!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📊 Check deployment status:"
echo "   kubectl get pods -n go-kafkify"
echo "   kubectl get services -n go-kafkify"
echo ""
echo "🔗 Access services:"
echo "   REST API:  kubectl port-forward -n go-kafkify svc/rest-service 8080:8080"
echo "   Grafana:   kubectl port-forward -n go-kafkify svc/grafana 3000:3000"
echo ""
echo "📝 View logs:"
echo "   kubectl logs -f -n go-kafkify -l app=rest-service"
echo "   kubectl logs -f -n go-kafkify -l app=grpc-service"
echo "   kubectl logs -f -n go-kafkify -l app=python-worker"
echo ""
echo "═══════════════════════════════════════════════════════════"
