#!/bin/bash
# Flyway Migration Management Script
# Provides common migration operations for local development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FLYWAY_CONFIG="$PROJECT_ROOT/database/flyway.conf"
MIGRATIONS_DIR="$PROJECT_ROOT/backend/db/schema/migrations"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "🗄️  Flyway Migration Management"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  migrate          Apply pending migrations"
    echo "  info             Show migration status"
    echo "  validate         Validate migrations"
    echo "  clean            Clean database (development only)"
    echo "  baseline         Mark database as baseline"
    echo "  repair           Repair migration history"
    echo "  create <name>    Create new migration file"
    echo "  docker-migrate   Run migrations using Docker Compose"
    echo "  docker-info      Show status using Docker Compose"
    echo ""
    echo "Environment options:"
    echo "  --local          Use local database (localhost:5432)"
    echo "  --docker         Use Docker Compose database"
    echo ""
    echo "Examples:"
    echo "  $0 migrate              # Apply migrations to local database"
    echo "  $0 docker-migrate       # Apply migrations using Docker Compose"
    echo "  $0 create add_indexes   # Create new migration file"
    echo "  $0 info --docker        # Show status for Docker database"
}

# Check if flyway is installed locally
check_flyway() {
    if ! command -v flyway &> /dev/null; then
        echo -e "${YELLOW}Warning: Flyway CLI not found locally. Using Docker mode.${NC}"
        return 1
    fi
    return 0
}

# Get database URL based on environment
get_database_url() {
    local env=${1:-docker}
    if [[ "$env" == "local" ]]; then
        echo "jdbc:postgresql://localhost:5432/kellogg_music_match"
    else
        echo "jdbc:postgresql://postgres:5432/kellogg_music_match"
    fi
}

# Run flyway command locally
run_flyway_local() {
    local command=$1
    local env=${2:-local}
    shift 2
    
    local url=$(get_database_url "$env")
    
    echo -e "${BLUE}Running Flyway $command locally...${NC}"
    flyway \
        -configFiles="$FLYWAY_CONFIG" \
        -url="$url" \
        -user=kellogg_user \
        -password=kellogg_secure_pass_2024 \
        -locations="filesystem:$MIGRATIONS_DIR" \
        "$command" \
        "$@"
}

# Run flyway command via Docker Compose
run_flyway_docker() {
    local command=$1
    shift
    
    echo -e "${BLUE}Running Flyway $command via Docker Compose...${NC}"
    cd "$PROJECT_ROOT"
    
    # Ensure database is running
    docker-compose up -d postgres
    echo "Waiting for database to be ready..."
    sleep 5
    
    # Run flyway command
    docker-compose run --rm \
        -v "$MIGRATIONS_DIR:/flyway/sql:ro" \
        -v "$FLYWAY_CONFIG:/flyway/conf/flyway.conf:ro" \
        flyway \
        -url=jdbc:postgresql://postgres:5432/kellogg_music_match \
        -user=kellogg_user \
        -password=kellogg_secure_pass_2024 \
        -schemas=public \
        -locations=filesystem:/flyway/sql \
        "$command" \
        "$@"
}

# Create new migration file
create_migration() {
    local name=$1
    if [[ -z "$name" ]]; then
        echo -e "${RED}Error: Migration name is required${NC}"
        echo "Usage: $0 create <migration_name>"
        exit 1
    fi
    
    # Get next version number
    local next_version=$(find "$MIGRATIONS_DIR" -name "V*.sql" | \
                        sed 's/.*V\([0-9]*\).*/\1/' | \
                        sort -n | tail -1)
    next_version=$((next_version + 1))
    
    # Format version with leading zeros
    local version=$(printf "%03d" "$next_version")
    local filename="V${version}__${name}.sql"
    local filepath="$MIGRATIONS_DIR/$filename"
    
    # Create migration file with template
    cat > "$filepath" << EOF
-- Migration: ${name}
-- Version: V${version}
-- Description: Add description here

-- Add your migration SQL here

EOF
    
    echo -e "${GREEN}Created migration file: $filename${NC}"
    echo "Edit: $filepath"
}

# Main command handling
case ${1:-help} in
    migrate)
        if check_flyway && [[ "$2" != "--docker" ]]; then
            run_flyway_local migrate "${2:-local}"
        else
            run_flyway_docker migrate
        fi
        ;;
    info)
        if check_flyway && [[ "$2" != "--docker" ]]; then
            run_flyway_local info "${2:-local}"
        else
            run_flyway_docker info
        fi
        ;;
    validate)
        if check_flyway && [[ "$2" != "--docker" ]]; then
            run_flyway_local validate "${2:-local}"
        else
            run_flyway_docker validate
        fi
        ;;
    clean)
        echo -e "${YELLOW}Warning: This will delete all database objects!${NC}"
        read -p "Are you sure? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if check_flyway && [[ "$2" != "--docker" ]]; then
                run_flyway_local clean "${2:-local}"
            else
                run_flyway_docker clean
            fi
        else
            echo "Operation cancelled."
        fi
        ;;
    baseline)
        if check_flyway && [[ "$2" != "--docker" ]]; then
            run_flyway_local baseline "${2:-local}"
        else
            run_flyway_docker baseline
        fi
        ;;
    repair)
        if check_flyway && [[ "$2" != "--docker" ]]; then
            run_flyway_local repair "${2:-local}"
        else
            run_flyway_docker repair
        fi
        ;;
    create)
        create_migration "$2"
        ;;
    docker-migrate)
        run_flyway_docker migrate
        ;;
    docker-info)
        run_flyway_docker info
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        echo ""
        usage
        exit 1
        ;;
esac