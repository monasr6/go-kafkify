#!/bin/bash
# Verify installation script

echo "🔍 Verifying Go-Kafkify Installation..."
echo ""

# Count files
TOTAL_FILES=$(find . -type f -not -path "./.git/*" | wc -l | tr -d ' ')
echo "✅ Total files created: $TOTAL_FILES"

# Check services
echo ""
echo "📦 Services:"
[ -f "services/rest-service/main.go" ] && echo "  ✅ REST Service (Go)" || echo "  ❌ REST Service missing"
[ -f "services/grpc-service/main.go" ] && echo "  ✅ gRPC Service (Go)" || echo "  ❌ gRPC Service missing"
[ -f "services/python-worker/main.py" ] && echo "  ✅ Python Worker" || echo "  ❌ Python Worker missing"

# Check migrations
echo ""
echo "🗄️  Migrations:"
[ -d "migrations/rest-service" ] && echo "  ✅ REST Service migrations" || echo "  ❌ REST migrations missing"
[ -d "migrations/grpc-service" ] && echo "  ✅ gRPC Service migrations" || echo "  ❌ gRPC migrations missing"
[ -d "migrations/python-worker" ] && echo "  ✅ Python Worker migrations" || echo "  ❌ Worker migrations missing"

# Check infrastructure
echo ""
echo "☸️  Infrastructure:"
K8S_FILES=$(find infrastructure/k8s -name "*.yaml" | wc -l | tr -d ' ')
echo "  ✅ Kubernetes manifests: $K8S_FILES files"
[ -f "docker-compose.yml" ] && echo "  ✅ Docker Compose configuration" || echo "  ❌ Docker Compose missing"

# Check observability
echo ""
echo "📊 Observability:"
[ -f "infrastructure/observability/otel-collector-config.yaml" ] && echo "  ✅ OpenTelemetry Collector" || echo "  ❌ OTEL missing"
[ -f "infrastructure/observability/prometheus.yml" ] && echo "  ✅ Prometheus" || echo "  ❌ Prometheus missing"
[ -d "infrastructure/observability/grafana" ] && echo "  ✅ Grafana dashboards" || echo "  ❌ Grafana missing"

# Check load tests
echo ""
echo "🧪 Load Tests:"
[ -f "load-tests/rest-api-test.js" ] && echo "  ✅ REST API test" || echo "  ❌ REST test missing"
[ -f "load-tests/kafka-throughput-test.js" ] && echo "  ✅ Kafka throughput test" || echo "  ❌ Kafka test missing"

# Check documentation
echo ""
echo "📚 Documentation:"
[ -f "README.md" ] && echo "  ✅ README.md" || echo "  ❌ README missing"
[ -f "GETTING_STARTED.md" ] && echo "  ✅ GETTING_STARTED.md" || echo "  ❌ Getting Started missing"
[ -f "ARCHITECTURE.md" ] && echo "  ✅ ARCHITECTURE.md" || echo "  ❌ Architecture missing"
[ -f "PROJECT_STRUCTURE.md" ] && echo "  ✅ PROJECT_STRUCTURE.md" || echo "  ❌ Project Structure missing"
[ -f "SUMMARY.md" ] && echo "  ✅ SUMMARY.md" || echo "  ❌ Summary missing"

# Check scripts
echo ""
echo "🔧 Scripts:"
[ -x "start.sh" ] && echo "  ✅ start.sh (executable)" || echo "  ⚠️  start.sh (not executable)"
[ -x "deploy-k8s.sh" ] && echo "  ✅ deploy-k8s.sh (executable)" || echo "  ⚠️  deploy-k8s.sh (not executable)"
[ -x "cleanup.sh" ] && echo "  ✅ cleanup.sh (executable)" || echo "  ⚠️  cleanup.sh (not executable)"
[ -f "Makefile" ] && echo "  ✅ Makefile" || echo "  ❌ Makefile missing"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "✨ Installation Verification Complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "🚀 Next steps:"
echo "   1. Review README.md for project overview"
echo "   2. Read GETTING_STARTED.md for setup instructions"
echo "   3. Run './start.sh' to start all services"
echo "   4. Access Grafana at http://localhost:3000"
echo ""
echo "💡 Quick commands:"
echo "   make help      - Show all available commands"
echo "   make start     - Start all services"
echo "   make demo      - Run a quick demo"
echo ""
