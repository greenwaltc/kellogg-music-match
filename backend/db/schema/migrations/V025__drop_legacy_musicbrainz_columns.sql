-- V025__drop_legacy_musicbrainz_columns.sql
-- Purpose: Remove obsolete MusicBrainz-specific columns and indexes now that the
-- application has migrated fully to Spotify-derived artist preferences.
--
-- This migration is SAFE to apply after confirming no code paths or queries
-- reference the following columns on artists:
--   musicbrainz_id, sort_name, artist_type, gender, country,
--   life_span_begin, life_span_end, disambiguation, musicbrainz_score, is_reference
--
-- Rollback strategy (manual): If you need these again, revert to a snapshot
-- prior to V025 or recreate columns manually (see V011/V012/V019 for reference).

BEGIN;

-- Drop dependent indexes / constraints first (if they still exist)
DO $$
BEGIN
    -- Unique / regular indexes
    IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'i' AND c.relname = 'idx_artists_musicbrainz_id') THEN
        EXECUTE 'DROP INDEX idx_artists_musicbrainz_id';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'i' AND c.relname = 'idx_artists_musicbrainz_id_unique') THEN
        EXECUTE 'DROP INDEX idx_artists_musicbrainz_id_unique';
    END IF;
    IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind = 'i' AND c.relname = 'idx_artists_score') THEN
        EXECUTE 'DROP INDEX idx_artists_score';
    END IF;
END $$;

-- Dynamically drop columns only if they exist (idempotent safety)
DO $$
DECLARE
    col TEXT;
    cols TEXT[] := ARRAY[
        'musicbrainz_id','sort_name','artist_type','gender','country',
        'life_span_begin','life_span_end','disambiguation','musicbrainz_score','is_reference'
    ];
BEGIN
    FOREACH col IN ARRAY cols LOOP
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'artists' AND column_name = col
        ) THEN
            EXECUTE format('ALTER TABLE artists DROP COLUMN %I', col);
        END IF;
    END LOOP;
END $$;

COMMIT;

-- NOTE: sqlc models referencing removed columns should be regenerated after applying this migration.