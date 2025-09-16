#!/bin/bash
# Development Environment Setup for Kellogg Music Match

set -e

echo "🎵 Kellogg Music Match - Development Setup"
echo "=========================================="

# Function to log messages
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

show_help() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start     Start all services (default)"
    echo "  stop      Stop all services"
    echo "  restart   Restart all services"
    echo "  logs      Show logs for all services"
    echo "  db-only   Start only the database"
    echo "  status    Show service status"
    echo "  cleanup   Stop services and remove volumes"
    echo "  test      Run the test suite"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start         # Start all services"
    echo "  $0 db-only       # Start only PostgreSQL for backend development"
    echo "  $0 logs          # Watch logs from all services"
}

start_all() {
    log "Starting all services..."
    docker-compose up -d
    log "✅ All services started!"
    log "🌐 UI: http://localhost:4200"
    log "🔗 Backend: http://localhost:8080"
    log "🗄️ Database: localhost:5432"
    log ""
    log "Run 'docker-compose logs -f' to see logs"
}

start_db_only() {
    log "Starting PostgreSQL database only..."
    docker-compose up -d postgres
    log "✅ Database started!"
    log "🗄️ Database available at: localhost:5432"
    log "📋 Connection details:"
    log "   Host: localhost"
    log "   Port: 5432"
    log "   Database: kellogg_music_match"
    log "   User: kellogg_user"
    log "   Password: kellogg_secure_pass_2024"
    log ""
    log "Test connection: psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match"
}

stop_services() {
    log "Stopping all services..."
    docker-compose down
    log "✅ All services stopped!"
}

restart_services() {
    log "Restarting all services..."
    docker-compose restart
    log "✅ All services restarted!"
}

show_logs() {
    log "Showing logs (press Ctrl+C to exit)..."
    docker-compose logs -f
}

show_status() {
    log "Service status:"
    docker-compose ps
    echo ""
    log "Volume usage:"
    docker volume ls | grep kellogg || echo "No volumes found"
}

cleanup() {
    log "Stopping services and cleaning up..."
    docker-compose down -v
    log "✅ Cleanup completed!"
}

run_tests() {
    log "Running test suite..."
    ./test-docker-compose.sh
}

# Main script logic
case "${1:-start}" in
    "start")
        start_all
        ;;
    "stop")
        stop_services
        ;;
    "restart")
        restart_services
        ;;
    "logs")
        show_logs
        ;;
    "db-only")
        start_db_only
        ;;
    "status")
        show_status
        ;;
    "cleanup")
        cleanup
        ;;
    "test")
        run_tests
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac