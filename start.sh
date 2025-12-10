#!/bin/bash
# Quick start script for Go-Kafkify platform

set -e

echo "🚀 Starting Go-Kafkify Platform..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: docker-compose is not installed."
    exit 1
fi

echo "✅ Docker is running"
echo ""

# Build and start all services
echo "📦 Building and starting services..."
docker-compose up -d --build

echo ""
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check service health
echo ""
echo "🔍 Checking service health..."

# Check REST service
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ REST Service is healthy"
else
    echo "⚠️  REST Service is not responding yet (may need more time)"
fi

# Check Prometheus
if curl -sf http://localhost:9091/-/healthy > /dev/null 2>&1; then
    echo "✅ Prometheus is healthy"
else
    echo "⚠️  Prometheus is not responding yet"
fi

# Check Grafana
if curl -sf http://localhost:3000/api/health > /dev/null 2>&1; then
    echo "✅ Grafana is healthy"
else
    echo "⚠️  Grafana is not responding yet"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "🎉 Go-Kafkify Platform Started!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📚 Access Points:"
echo "   REST API:     http://localhost:8080"
echo "   gRPC Service: localhost:9090"
echo "   Grafana:      http://localhost:3000 (admin/admin)"
echo "   Prometheus:   http://localhost:9091"
echo ""
echo "📊 Metrics Endpoints:"
echo "   REST Service:   http://localhost:8080/metrics"
echo "   gRPC Service:   http://localhost:9093/metrics"
echo "   Python Worker:  http://localhost:9092/metrics"
echo ""
echo "🔧 Useful Commands:"
echo "   View logs:        docker-compose logs -f [service-name]"
echo "   Stop services:    docker-compose down"
echo "   Restart service:  docker-compose restart [service-name]"
echo ""
echo "🧪 Try it out:"
echo "   curl -X POST http://localhost:8080/api/v1/resources \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"name\":\"Test\",\"description\":\"My first resource\",\"status\":\"active\"}'"
echo ""
echo "   curl http://localhost:8080/api/v1/resources"
echo ""
echo "═══════════════════════════════════════════════════════════"
