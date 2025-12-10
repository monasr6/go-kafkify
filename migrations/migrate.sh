#!/bin/bash
# Script to run all database migrations

set -e

echo "Running database migrations..."

# REST Service migrations
echo "Migrating REST service database..."
PGPASSWORD=${POSTGRES_PASSWORD:-postgres} psql -h ${POSTGRES_HOST:-localhost} -p ${POSTGRES_PORT:-5432} -U ${POSTGRES_USER:-postgres} -d restdb -f migrations/rest-service/001_init.up.sql
echo "✓ REST service database migrated"

# gRPC Service migrations
echo "Migrating gRPC service database..."
PGPASSWORD=${POSTGRES_PASSWORD:-postgres} psql -h ${POSTGRES_HOST:-localhost} -p ${POSTGRES_PORT:-5432} -U ${POSTGRES_USER:-postgres} -d grpcdb -f migrations/grpc-service/001_init.up.sql
echo "✓ gRPC service database migrated"

# Python Worker migrations
echo "Migrating Python worker database..."
PGPASSWORD=${POSTGRES_PASSWORD:-postgres} psql -h ${POSTGRES_HOST:-localhost} -p ${POSTGRES_PORT:-5432} -U ${POSTGRES_USER:-postgres} -d workerdb -f migrations/python-worker/001_init.up.sql
echo "✓ Python worker database migrated"

echo "All migrations completed successfully!"
