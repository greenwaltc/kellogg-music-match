#!/bin/bash

# Post-deployment MusicBrainz data loading for Kubernetes
# Run this after your Pulumi deployment completes

set -e

echo "🎵 Loading MusicBrainz artists into Kubernetes cluster..."

# Check if we're in the right directory
if [ ! -f "musicbrainz_artists_50k.csv" ]; then
    echo "❌ Error: musicbrainz_artists_50k.csv not found in current directory"
    echo "Please run this script from the project root directory"
    exit 1
fi

# Get the PostgreSQL pod name
POSTGRES_POD=$(kubectl get pods -n affyne -l app=postgres -o jsonpath='{.items[0].metadata.name}')

if [ -z "$POSTGRES_POD" ]; then
    echo "❌ Error: No PostgreSQL pod found in affyne namespace"
    echo "Make sure your Pulumi deployment has completed successfully"
    exit 1
fi

echo "📦 Found PostgreSQL pod: $POSTGRES_POD"

# Check if data already exists
EXISTING_COUNT=$(kubectl exec -n affyne "$POSTGRES_POD" -- psql -U kellogg_user -d kellogg_music_match -t -c "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | tr -d ' ')

if [ "$EXISTING_COUNT" -gt 1000 ]; then
    echo "✅ MusicBrainz data already exists ($EXISTING_COUNT reference artists)"
    echo "Skipping data load"
    exit 0
fi

echo "📤 Copying CSV file to PostgreSQL pod..."
kubectl cp musicbrainz_artists_50k.csv "affyne/$POSTGRES_POD:/tmp/musicbrainz_artists.csv"

echo "🔄 Loading MusicBrainz artists data..."
kubectl exec -n affyne "$POSTGRES_POD" -- psql -U kellogg_user -d kellogg_music_match -c "
CREATE TEMP TABLE temp_musicbrainz_load (
    musicbrainz_id TEXT, name TEXT, sort_name TEXT, artist_type TEXT,
    gender TEXT, country TEXT, life_span_begin TEXT, life_span_end TEXT,
    disambiguation TEXT, musicbrainz_score TEXT
);

COPY temp_musicbrainz_load FROM '/tmp/musicbrainz_artists.csv' 
WITH (FORMAT csv, HEADER true, DELIMITER ',', QUOTE '\"', ESCAPE '\"');

INSERT INTO artists (
    name, musicbrainz_id, sort_name, artist_type, gender, country, 
    life_span_begin, life_span_end, disambiguation, musicbrainz_score, 
    is_reference, created_at
)
SELECT DISTINCT ON (TRIM(name))
    TRIM(name),
    CASE WHEN TRIM(musicbrainz_id) = '' THEN NULL ELSE TRIM(musicbrainz_id)::UUID END,
    TRIM(sort_name), TRIM(artist_type),
    CASE WHEN TRIM(gender) = '' THEN NULL ELSE TRIM(gender) END,
    CASE WHEN TRIM(country) = '' THEN NULL ELSE TRIM(country) END,
    CASE WHEN TRIM(life_span_begin) = '' THEN NULL 
         WHEN TRIM(life_span_begin) ~ '^\\\d{4}$' THEN (TRIM(life_span_begin) || '-01-01')::DATE
         WHEN TRIM(life_span_begin) ~ '^\\\d{4}-\\\d{2}$' THEN (TRIM(life_span_begin) || '-01')::DATE
         ELSE TRIM(life_span_begin)::DATE END,
    CASE WHEN TRIM(life_span_end) = '' THEN NULL 
         WHEN TRIM(life_span_end) ~ '^\\\d{4}$' THEN (TRIM(life_span_end) || '-01-01')::DATE
         WHEN TRIM(life_span_end) ~ '^\\\d{4}-\\\d{2}$' THEN (TRIM(life_span_end) || '-01')::DATE
         ELSE TRIM(life_span_end)::DATE END,
    CASE WHEN TRIM(disambiguation) = '' THEN NULL ELSE TRIM(disambiguation) END,
    CASE WHEN TRIM(musicbrainz_score) = '' THEN NULL ELSE TRIM(musicbrainz_score)::INTEGER END,
    TRUE, CURRENT_TIMESTAMP
FROM temp_musicbrainz_load
WHERE TRIM(musicbrainz_id) != '' AND TRIM(name) != ''
ORDER BY TRIM(name), musicbrainz_score DESC NULLS LAST;

DROP TABLE temp_musicbrainz_load;
"

# Verify the load
FINAL_COUNT=$(kubectl exec -n affyne "$POSTGRES_POD" -- psql -U kellogg_user -d kellogg_music_match -t -c "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;" | tr -d ' ')

echo "✅ Successfully loaded MusicBrainz data!"
echo "📊 Total reference artists: $FINAL_COUNT"

# Clean up
kubectl exec -n affyne "$POSTGRES_POD" -- rm -f /tmp/musicbrainz_artists.csv

echo "🎉 MusicBrainz data loading complete!"