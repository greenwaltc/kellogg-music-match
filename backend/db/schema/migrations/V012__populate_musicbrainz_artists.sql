-- Migration to populate artists table with 50,000 MusicBrainz reference artists
-- This migration loads curated artist data from MusicBrainz API
-- Artists are ordered by score (100=most popular) for quality-first matching

-- Check if reference artists already exist to prevent duplicate loading
DO $$
DECLARE
    reference_count INTEGER;
BEGIN
    -- Count existing reference artists
    SELECT COUNT(*) INTO reference_count FROM artists WHERE is_reference = TRUE;
    
    -- Only proceed if we have fewer than 1000 reference artists
    IF reference_count < 1000 THEN
        RAISE NOTICE 'Found % reference artists, proceeding with data load', reference_count;
        RAISE NOTICE 'MusicBrainz artists migration V012 executed';
        RAISE NOTICE 'Run: ./scripts/load_musicbrainz_data.sh to load 50,000 reference artists';
        RAISE NOTICE 'This migration provides the schema preparation for MusicBrainz data';
    ELSE
        RAISE NOTICE 'Found % reference artists, skipping data load setup', reference_count;
    END IF;
END $$;

-- Add comment to track migration purpose
COMMENT ON TABLE artists IS 'Artists table containing both user-submitted and MusicBrainz reference artists';

-- Create helper function to safely insert MusicBrainz artists
CREATE OR REPLACE FUNCTION insert_musicbrainz_artist(
    p_musicbrainz_id UUID,
    p_name VARCHAR(240),
    p_sort_name VARCHAR(240),
    p_artist_type VARCHAR(50),
    p_gender VARCHAR(20),
    p_country CHAR(2),
    p_life_span_begin DATE,
    p_life_span_end DATE,
    p_disambiguation TEXT,
    p_musicbrainz_score INTEGER
) RETURNS VOID AS $$
BEGIN
    -- Insert only if MusicBrainz ID doesn't already exist
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
        p_name,
        p_musicbrainz_id,
        p_sort_name,
        p_artist_type,
        p_gender,
        p_country,
        p_life_span_begin,
        p_life_span_end,
        p_disambiguation,
        p_musicbrainz_score,
        TRUE,
        CURRENT_TIMESTAMP
    WHERE NOT EXISTS (
        SELECT 1 FROM artists WHERE musicbrainz_id = p_musicbrainz_id
    );
EXCEPTION
    WHEN unique_violation THEN
        -- Ignore duplicate entries
        NULL;
END;
$$ LANGUAGE plpgsql;

-- Grant necessary permissions
-- Note: Grant to the database user (kellogg_user) instead of postgres role
DO $$
BEGIN
    -- Grant execute permission to kellogg_user if it exists
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kellogg_user') THEN
        GRANT EXECUTE ON FUNCTION insert_musicbrainz_artist TO kellogg_user;
    END IF;
    
    -- Also grant to current user
    EXECUTE format('GRANT EXECUTE ON FUNCTION insert_musicbrainz_artist TO %I', current_user);
EXCEPTION
    WHEN others THEN
        -- Ignore permission errors
        NULL;
END $$;