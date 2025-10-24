#!/bin/bash
# Load MusicBrainz artists directly into Kubernetes PostgreSQL
# This script copies CSV data into the pod and loads it via COPY command

CSV_FILE="$1"
NAMESPACE="affyne"
POD_NAME="postgres-0"

if [ -z "$CSV_FILE" ]; then
    echo "Usage: $0 <csv_file>"
    echo "Example: $0 musicbrainz_artists_1k_converted.csv"
    exit 1
fi

if [ ! -f "$CSV_FILE" ]; then
    echo "Error: File $CSV_FILE not found"
    exit 1
fi

echo "Loading MusicBrainz artists from $CSV_FILE into Kubernetes PostgreSQL..."

# Copy CSV file to PostgreSQL pod
echo "1. Copying CSV file to PostgreSQL pod..."
kubectl cp "$CSV_FILE" "$NAMESPACE/$POD_NAME:/tmp/artists.csv"

# Create a SQL script to load the data
echo "2. Creating load script..."
cat << 'EOF' > /tmp/load_artists.sql
-- Create temporary table for CSV import
CREATE TEMP TABLE temp_artists (
    id TEXT,
    name TEXT,
    sort_name TEXT,
    type TEXT,
    gender TEXT,
    country TEXT,
    life_span_begin TEXT,
    life_span_end TEXT,
    disambiguation TEXT,
    score TEXT
);

-- Import CSV data
\copy temp_artists FROM '/tmp/artists.csv' WITH (FORMAT csv, HEADER true);

-- Function to parse dates
CREATE OR REPLACE FUNCTION parse_musicbrainz_date(date_str TEXT) RETURNS DATE AS $$
BEGIN
    IF date_str IS NULL OR trim(date_str) = '' THEN
        RETURN NULL;
    END IF;
    
    date_str := trim(date_str);
    
    IF length(date_str) = 4 THEN
        RETURN (date_str || '-01-01')::DATE;
    ELSIF length(date_str) = 7 THEN
        RETURN (date_str || '-01')::DATE;
    ELSIF length(date_str) = 10 THEN
        RETURN date_str::DATE;
    ELSE
        RETURN NULL;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Insert artists using upsert function
INSERT INTO artists (
    name, musicbrainz_id, sort_name, artist_type, gender, country,
    life_span_begin, life_span_end, disambiguation, musicbrainz_score, is_reference
)
SELECT DISTINCT
    trim(t.name),
    CASE WHEN trim(t.id) = '' THEN NULL ELSE trim(t.id)::UUID END,
    CASE WHEN trim(t.sort_name) = '' THEN NULL ELSE trim(t.sort_name) END,
    CASE WHEN trim(t.type) = '' THEN NULL ELSE trim(t.type) END,
    CASE WHEN trim(t.gender) = '' THEN NULL ELSE trim(t.gender) END,
    CASE WHEN trim(t.country) = '' THEN NULL ELSE trim(t.country) END,
    parse_musicbrainz_date(t.life_span_begin),
    parse_musicbrainz_date(t.life_span_end),
    CASE WHEN trim(t.disambiguation) = '' THEN NULL ELSE trim(t.disambiguation) END,
    CASE WHEN trim(t.score) = '' THEN NULL ELSE trim(t.score)::INTEGER END,
    TRUE
FROM temp_artists t
WHERE trim(t.name) IS NOT NULL AND trim(t.name) != ''
ON CONFLICT (musicbrainz_id) 
DO UPDATE SET
    name = EXCLUDED.name,
    sort_name = COALESCE(EXCLUDED.sort_name, artists.sort_name),
    artist_type = COALESCE(EXCLUDED.artist_type, artists.artist_type),
    gender = COALESCE(EXCLUDED.gender, artists.gender),
    country = COALESCE(EXCLUDED.country, artists.country),
    life_span_begin = COALESCE(EXCLUDED.life_span_begin, artists.life_span_begin),
    life_span_end = COALESCE(EXCLUDED.life_span_end, artists.life_span_end),
    disambiguation = COALESCE(EXCLUDED.disambiguation, artists.disambiguation),
    musicbrainz_score = COALESCE(EXCLUDED.musicbrainz_score, artists.musicbrainz_score),
    is_reference = TRUE;

-- Show statistics
SELECT 
    COUNT(*) as total_artists,
    COUNT(*) FILTER (WHERE is_reference = TRUE) as reference_artists,
    COUNT(*) FILTER (WHERE is_reference = FALSE) as user_artists
FROM artists;

-- Show top countries
SELECT country, COUNT(*) as count 
FROM artists 
WHERE country IS NOT NULL AND is_reference = TRUE
GROUP BY country 
ORDER BY count DESC 
LIMIT 10;

-- Show top artist types
SELECT artist_type, COUNT(*) as count 
FROM artists 
WHERE artist_type IS NOT NULL AND is_reference = TRUE
GROUP BY artist_type 
ORDER BY count DESC;

-- Clean up
DROP FUNCTION parse_musicbrainz_date(TEXT);
EOF

# Copy SQL script to pod
echo "3. Copying SQL script to pod..."
kubectl cp "/tmp/load_artists.sql" "$NAMESPACE/$POD_NAME:/tmp/load_artists.sql"

# Execute the SQL script
echo "4. Loading artists into database..."
kubectl exec -i "$POD_NAME" -n "$NAMESPACE" -- psql -U kellogg_user -d kellogg_music_match -f /tmp/load_artists.sql

# Clean up temporary files
echo "5. Cleaning up..."
kubectl exec "$POD_NAME" -n "$NAMESPACE" -- rm -f /tmp/artists.csv /tmp/load_artists.sql
rm -f /tmp/load_artists.sql

echo "✅ MusicBrainz artists loaded successfully!"