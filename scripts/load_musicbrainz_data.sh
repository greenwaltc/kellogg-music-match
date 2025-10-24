#!/bin/bash

# MusicBrainz Artists Data Loader for Flyway Migration
# This script loads the 50,000 artist CSV data into PostgreSQL as part of V012 migration
# 
# Usage: ./scripts/load_musicbrainz_data.sh [csv_file] [database_url]
# 
# Environment Variables:
#   DATABASE_URL - PostgreSQL connection string (if not provided as argument)
#   POSTGRES_PASSWORD - Database password (for Kubernetes environments)

set -e

# Configuration
CSV_FILE="${1:-musicbrainz_artists_50k.csv}"
DATABASE_URL="${2:-$DATABASE_URL}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if running in Kubernetes environment
if [ -z "$DATABASE_URL" ] && kubectl get pods -n affyne >/dev/null 2>&1; then
    log "Detected Kubernetes environment"
    KUBERNETES_MODE=true
    POSTGRES_POD=$(kubectl get pods -n affyne -l app=postgres -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$POSTGRES_POD" ]; then
        error "No PostgreSQL pod found in affyne namespace"
        exit 1
    fi
    
    log "Using PostgreSQL pod: $POSTGRES_POD"
else
    KUBERNETES_MODE=false
    
    if [ -z "$DATABASE_URL" ]; then
        error "DATABASE_URL must be set when not running in Kubernetes"
        exit 1
    fi
fi

# Verify CSV file exists
if [ ! -f "$PROJECT_ROOT/$CSV_FILE" ]; then
    error "CSV file not found: $PROJECT_ROOT/$CSV_FILE"
    exit 1
fi

# Count lines in CSV (excluding header)
TOTAL_ARTISTS=$(( $(wc -l < "$PROJECT_ROOT/$CSV_FILE") - 1 ))
log "Found $TOTAL_ARTISTS artists in CSV file"

# Function to execute SQL in Kubernetes
exec_k8s_sql() {
    local sql="$1"
    kubectl exec -n affyne "$POSTGRES_POD" -- psql -U postgres -d postgres -c "$sql"
}

# Function to execute SQL with local DATABASE_URL
exec_local_sql() {
    local sql="$1"
    psql "$DATABASE_URL" -c "$sql"
}

# Function to copy CSV data in Kubernetes
copy_k8s_csv() {
    log "Copying CSV file to PostgreSQL pod..."
    kubectl cp "$PROJECT_ROOT/$CSV_FILE" "affyne/$POSTGRES_POD:/tmp/musicbrainz_artists.csv"
    
    log "Loading data using COPY command..."
    kubectl exec -n affyne "$POSTGRES_POD" -- psql -U postgres -d postgres -c "
        CREATE TEMP TABLE temp_musicbrainz_load (
            musicbrainz_id TEXT,
            name TEXT,
            sort_name TEXT,
            artist_type TEXT,
            gender TEXT,
            country TEXT,
            life_span_begin TEXT,
            life_span_end TEXT,
            disambiguation TEXT,
            musicbrainz_score TEXT
        );
        
        COPY temp_musicbrainz_load FROM '/tmp/musicbrainz_artists.csv' 
        WITH (FORMAT csv, HEADER true, DELIMITER ',', QUOTE '\"', ESCAPE '\"');
        
        INSERT INTO artists (
            name, 
            musicbrainz_id, 
            sort_name, 
            artist_type, 
            gender, 
            country, 
            life_span_begin, 
            life_span_end, 
            disambiguation, 
            musicbrainz_score, 
            is_reference,
            created_at
        )
        SELECT 
            TRIM(name),
            CASE WHEN TRIM(musicbrainz_id) = '' THEN NULL ELSE TRIM(musicbrainz_id)::UUID END,
            TRIM(sort_name),
            TRIM(artist_type),
            CASE WHEN TRIM(gender) = '' THEN NULL ELSE TRIM(gender) END,
            CASE WHEN TRIM(country) = '' THEN NULL ELSE TRIM(country) END,
            CASE WHEN TRIM(life_span_begin) = '' THEN NULL ELSE TRIM(life_span_begin)::DATE END,
            CASE WHEN TRIM(life_span_end) = '' THEN NULL ELSE TRIM(life_span_end)::DATE END,
            CASE WHEN TRIM(disambiguation) = '' THEN NULL ELSE TRIM(disambiguation) END,
            CASE WHEN TRIM(musicbrainz_score) = '' THEN NULL ELSE TRIM(musicbrainz_score)::INTEGER END,
            TRUE,
            CURRENT_TIMESTAMP
        FROM temp_musicbrainz_load
        WHERE TRIM(musicbrainz_id) != ''
        ON CONFLICT (musicbrainz_id) DO NOTHING;
        
        DROP TABLE temp_musicbrainz_load;
    "
    
    log "Cleaning up temporary file..."
    kubectl exec -n affyne "$POSTGRES_POD" -- rm -f /tmp/musicbrainz_artists.csv
}

# Main execution
log "Starting MusicBrainz artists data load..."

if [ "$KUBERNETES_MODE" = true ]; then
    # Check if data already exists
    EXISTING_COUNT=$(exec_k8s_sql "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | grep -o '[0-9]\+' | head -1)
    
    if [ "$EXISTING_COUNT" -gt 1000 ]; then
        warn "Found $EXISTING_COUNT reference artists already exist"
        warn "Skipping data load to prevent duplicates"
        exit 0
    fi
    
    log "Found $EXISTING_COUNT existing reference artists"
    copy_k8s_csv
    
    # Verify load
    FINAL_COUNT=$(exec_k8s_sql "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | grep -o '[0-9]\+' | head -1)
    NEW_ARTISTS=$(( FINAL_COUNT - EXISTING_COUNT ))
    
else
    # Local database mode
    EXISTING_COUNT=$(exec_local_sql "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | grep -o '[0-9]\+' | head -1)
    
    if [ "$EXISTING_COUNT" -gt 1000 ]; then
        warn "Found $EXISTING_COUNT reference artists already exist"
        warn "Skipping data load to prevent duplicates"
        exit 0
    fi
    
    log "Found $EXISTING_COUNT existing reference artists"
    log "Loading data using local COPY command..."
    
    exec_local_sql "
        CREATE TEMP TABLE temp_musicbrainz_load (
            musicbrainz_id TEXT,
            name TEXT,
            sort_name TEXT,
            artist_type TEXT,
            gender TEXT,
            country TEXT,
            life_span_begin TEXT,
            life_span_end TEXT,
            disambiguation TEXT,
            musicbrainz_score TEXT
        );
        
        \\copy temp_musicbrainz_load FROM '$PROJECT_ROOT/$CSV_FILE' WITH (FORMAT csv, HEADER true, DELIMITER ',', QUOTE '\"', ESCAPE '\"');
        
        INSERT INTO artists (
            name, 
            musicbrainz_id, 
            sort_name, 
            artist_type, 
            gender, 
            country, 
            life_span_begin, 
            life_span_end, 
            disambiguation, 
            musicbrainz_score, 
            is_reference,
            created_at
        )
        SELECT 
            TRIM(name),
            CASE WHEN TRIM(musicbrainz_id) = '' THEN NULL ELSE TRIM(musicbrainz_id)::UUID END,
            TRIM(sort_name),
            TRIM(artist_type),
            CASE WHEN TRIM(gender) = '' THEN NULL ELSE TRIM(gender) END,
            CASE WHEN TRIM(country) = '' THEN NULL ELSE TRIM(country) END,
            CASE WHEN TRIM(life_span_begin) = '' THEN NULL ELSE TRIM(life_span_begin)::DATE END,
            CASE WHEN TRIM(life_span_end) = '' THEN NULL ELSE TRIM(life_span_end)::DATE END,
            CASE WHEN TRIM(disambiguation) = '' THEN NULL ELSE TRIM(disambiguation) END,
            CASE WHEN TRIM(musicbrainz_score) = '' THEN NULL ELSE TRIM(musicbrainz_score)::INTEGER END,
            TRUE,
            CURRENT_TIMESTAMP
        FROM temp_musicbrainz_load
        WHERE TRIM(musicbrainz_id) != ''
        ON CONFLICT (musicbrainz_id) DO NOTHING;
        
        DROP TABLE temp_musicbrainz_load;
    "
    
    # Verify load
    FINAL_COUNT=$(exec_local_sql "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | grep -o '[0-9]\+' | head -1)
    NEW_ARTISTS=$(( FINAL_COUNT - EXISTING_COUNT ))
fi

success "Successfully loaded $NEW_ARTISTS new reference artists"
success "Total reference artists: $FINAL_COUNT"

# Display some statistics
log "Loading statistics:"
if [ "$KUBERNETES_MODE" = true ]; then
    exec_k8s_sql "
        SELECT 
            'Total Artists' as metric, COUNT(*) as count 
        FROM artists
        UNION ALL
        SELECT 
            'Reference Artists' as metric, COUNT(*) as count 
        FROM artists WHERE is_reference = TRUE
        UNION ALL
        SELECT 
            'User Artists' as metric, COUNT(*) as count 
        FROM artists WHERE is_reference = FALSE
        UNION ALL
        SELECT 
            'Top Countries' as metric, COUNT(DISTINCT country) as count 
        FROM artists WHERE country IS NOT NULL AND is_reference = TRUE;
    "
else
    exec_local_sql "
        SELECT 
            'Total Artists' as metric, COUNT(*) as count 
        FROM artists
        UNION ALL
        SELECT 
            'Reference Artists' as metric, COUNT(*) as count 
        FROM artists WHERE is_reference = TRUE
        UNION ALL
        SELECT 
            'User Artists' as metric, COUNT(*) as count 
        FROM artists WHERE is_reference = FALSE
        UNION ALL
        SELECT 
            'Top Countries' as metric, COUNT(DISTINCT country) as count 
        FROM artists WHERE country IS NOT NULL AND is_reference = TRUE;
    "
fi

success "MusicBrainz artists data load completed successfully!"