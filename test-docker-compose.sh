#!/bin/bash
# Docker Compose Testing Script for Kellogg Music Match

set -e

echo "🎵 Kellogg Music Match - Docker Compose Test"
echo "============================================="

# Function to log messages
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Function to check if a service is healthy
check_service_health() {
    local service_name=$1
    local max_attempts=30
    local attempt=1
    
    log "Checking health of $service_name service..."
    
    while [ $attempt -le $max_attempts ]; do
        if docker-compose ps $service_name | grep -q "healthy\|Up"; then
            log "✅ $service_name is healthy!"
            return 0
        fi
        
        log "⏳ Waiting for $service_name to be healthy (attempt $attempt/$max_attempts)..."
        sleep 10
        attempt=$((attempt + 1))
    done
    
    log "❌ $service_name failed to become healthy after $max_attempts attempts"
    return 1
}

# Function to test database connection
test_database_connection() {
    log "Testing database connection..."
    
    if docker-compose exec -T postgres psql -U kellogg_user -d kellogg_music_match -c "SELECT 1;" > /dev/null 2>&1; then
        log "✅ Database connection successful!"
        return 0
    else
        log "❌ Database connection failed!"
        return 1
    fi
}

# Function to test database schema
test_database_schema() {
    log "Testing database schema..."
    
    # Check if tables exist
    local tables=$(docker-compose exec -T postgres psql -U kellogg_user -d kellogg_music_match -t -c "SELECT tablename FROM pg_tables WHERE schemaname = 'public';" | grep -E "(users|artists|user_artists)" | wc -l)
    
    if [ "$tables" -eq 3 ]; then
        log "✅ All required tables exist!"
        return 0
    else
        log "❌ Missing required tables (found $tables, expected 3)"
        return 1
    fi
}

# Function to test sample data
test_sample_data() {
    log "Testing sample data..."
    
    local artist_count=$(docker-compose exec -T postgres psql -U kellogg_user -d kellogg_music_match -t -c "SELECT COUNT(*) FROM artists;" | xargs)
    
    if [ "$artist_count" -gt 0 ]; then
        log "✅ Sample data loaded! Found $artist_count artists."
        return 0
    else
        log "❌ No sample data found!"
        return 1
    fi
}

# Main test function
run_tests() {
    log "Starting Docker Compose services..."
    docker-compose up -d
    
    log "Waiting for services to start..."
    sleep 10
    
    # Test PostgreSQL
    if check_service_health "postgres"; then
        test_database_connection && test_database_schema && test_sample_data
    fi
    
    # Test Backend (if it's running)
    if docker-compose ps backend | grep -q "Up"; then
        log "✅ Backend service is running!"
    else
        log "⚠️  Backend service is not running (may need database integration)"
    fi
    
    # Test UI
    if docker-compose ps ui | grep -q "Up"; then
        log "✅ UI service is running!"
        log "🌐 UI available at: http://localhost:4200"
    else
        log "❌ UI service is not running"
    fi
    
    log "📊 Service status:"
    docker-compose ps
    
    log "🎯 Test completed!"
}

# Cleanup function
cleanup() {
    log "Stopping services..."
    docker-compose down
}

# Check if --cleanup flag is provided
if [ "$1" = "--cleanup" ]; then
    cleanup
    exit 0
fi

# Trap cleanup on script exit
trap cleanup EXIT

# Run the tests
run_tests