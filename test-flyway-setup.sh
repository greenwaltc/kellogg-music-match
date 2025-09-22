#!/bin/bash

# Test Flyway Migration Setup
# This script tests the complete Flyway migration workflow

set -e

echo "🚀 Testing Flyway Migration Setup"
echo "================================="

# Ensure we're in the right directory
cd "$(dirname "$0")"

echo "📁 Current directory: $(pwd)"

# Clean up any existing containers
echo "🧹 Cleaning up existing containers..."
docker-compose down -v 2>/dev/null || true

# Start with a clean database
echo "🗄️  Starting fresh PostgreSQL instance..."
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 10

# Run Flyway migration manually to test
echo "🔄 Running Flyway migrations..."
docker run --rm \
  --network kellogg-music-match_default \
  -v $(pwd)/database/migrations:/flyway/sql \
  -v $(pwd)/database/flyway.conf:/flyway/conf/flyway.conf \
  flyway/flyway:latest \
  info

echo "✅ Checking migration status..."
docker run --rm \
  --network kellogg-music-match_default \
  -v $(pwd)/database/migrations:/flyway/sql \
  -v $(pwd)/database/flyway.conf:/flyway/conf/flyway.conf \
  flyway/flyway:latest \
  migrate

echo "📊 Final migration status..."
docker run --rm \
  --network kellogg-music-match_default \
  -v $(pwd)/database/migrations:/flyway/sql \
  -v $(pwd)/database/flyway.conf:/flyway/conf/flyway.conf \
  flyway/flyway:latest \
  info

echo ""
echo "🎉 Flyway migration test completed successfully!"
echo "💡 To test the full stack: docker-compose up"

# Clean up
echo "🧹 Cleaning up test containers..."
docker-compose down