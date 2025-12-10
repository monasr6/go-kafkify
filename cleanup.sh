#!/bin/bash
# Cleanup script - stops and removes all containers and volumes

set -e

echo "🧹 Cleaning up Go-Kafkify..."
echo ""

# Stop and remove containers
echo "🛑 Stopping and removing containers..."
docker-compose down -v

echo ""
echo "🗑️  Removing Docker images..."
docker rmi -f rest-service:latest 2>/dev/null || true
docker rmi -f grpc-service:latest 2>/dev/null || true
docker rmi -f python-worker:latest 2>/dev/null || true

echo ""
echo "✨ Cleanup complete!"
