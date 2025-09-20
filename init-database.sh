#!/bin/bash
# PostgreSQL Database Initialization Script for Kellogg Music Match

set -e

# Function to log messages
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Check if we should initialize the database
if [ "$POSTGRES_DB" ] && [ "$POSTGRES_USER" ]; then
    log "Initializing Kellogg Music Match database: $POSTGRES_DB"
    
    # Check if schema file exists and run it
    if [ -f "/docker-entrypoint-initdb.d/DATABASE_SCHEMA.sql" ]; then
        log "Running DATABASE_SCHEMA.sql..."
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "/docker-entrypoint-initdb.d/DATABASE_SCHEMA.sql"
        log "Database initialization completed successfully"
    else
        log "ERROR: DATABASE_SCHEMA.sql not found in /docker-entrypoint-initdb.d/"
        exit 1
    fi
else
    log "Skipping database initialization - missing environment variables"
fi