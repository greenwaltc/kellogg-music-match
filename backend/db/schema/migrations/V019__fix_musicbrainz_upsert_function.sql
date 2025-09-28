-- Fix upsert_artist_with_musicbrainz function to remove references to non-existent updated_at column
-- This fixes the MusicBrainz data loading process that was failing due to missing column references

-- Drop and recreate the function with the fix
DROP FUNCTION IF EXISTS upsert_artist_with_musicbrainz(VARCHAR, UUID, VARCHAR, VARCHAR, VARCHAR, CHAR, DATE, DATE, TEXT, INTEGER, BOOLEAN);

CREATE OR REPLACE FUNCTION upsert_artist_with_musicbrainz(
    p_name VARCHAR(240),
    p_musicbrainz_id UUID DEFAULT NULL,
    p_sort_name VARCHAR(240) DEFAULT NULL,
    p_artist_type VARCHAR(50) DEFAULT NULL,
    p_gender VARCHAR(20) DEFAULT NULL,
    p_country CHAR(2) DEFAULT NULL,
    p_life_span_begin DATE DEFAULT NULL,
    p_life_span_end DATE DEFAULT NULL,
    p_disambiguation TEXT DEFAULT NULL,
    p_musicbrainz_score INTEGER DEFAULT NULL,
    p_is_reference BOOLEAN DEFAULT FALSE
) RETURNS INTEGER AS $$
DECLARE
    artist_id INTEGER;
BEGIN
    -- Try to find existing artist by MusicBrainz ID first
    IF p_musicbrainz_id IS NOT NULL THEN
        SELECT id INTO artist_id 
        FROM artists 
        WHERE musicbrainz_id = p_musicbrainz_id;
        
        IF FOUND THEN
            -- Update existing artist with new data (removed updated_at reference)
            UPDATE artists SET
                name = p_name,
                sort_name = COALESCE(p_sort_name, sort_name),
                artist_type = COALESCE(p_artist_type, artist_type),
                gender = COALESCE(p_gender, gender),
                country = COALESCE(p_country, country),
                life_span_begin = COALESCE(p_life_span_begin, life_span_begin),
                life_span_end = COALESCE(p_life_span_end, life_span_end),
                disambiguation = COALESCE(p_disambiguation, disambiguation),
                musicbrainz_score = COALESCE(p_musicbrainz_score, musicbrainz_score),
                is_reference = p_is_reference
            WHERE id = artist_id;
            
            RETURN artist_id;
        END IF;
    END IF;
    
    -- Try to find by name if no MusicBrainz ID match
    SELECT id INTO artist_id 
    FROM artists 
    WHERE name = p_name;
    
    IF FOUND THEN
        -- Update existing artist, preserving user data if this is reference data (removed updated_at reference)
        UPDATE artists SET
            musicbrainz_id = COALESCE(p_musicbrainz_id, musicbrainz_id),
            sort_name = COALESCE(p_sort_name, sort_name),
            artist_type = COALESCE(p_artist_type, artist_type),
            gender = COALESCE(p_gender, gender),
            country = COALESCE(p_country, country),
            life_span_begin = COALESCE(p_life_span_begin, life_span_begin),
            life_span_end = COALESCE(p_life_span_end, life_span_end),
            disambiguation = COALESCE(p_disambiguation, disambiguation),
            musicbrainz_score = COALESCE(p_musicbrainz_score, musicbrainz_score),
            is_reference = CASE WHEN p_is_reference THEN TRUE ELSE is_reference END
        WHERE id = artist_id;
        
        RETURN artist_id;
    ELSE
        -- Insert new artist
        INSERT INTO artists (
            name, musicbrainz_id, sort_name, artist_type, gender, country,
            life_span_begin, life_span_end, disambiguation, musicbrainz_score, is_reference
        ) VALUES (
            p_name, p_musicbrainz_id, p_sort_name, p_artist_type, p_gender, p_country,
            p_life_span_begin, p_life_span_end, p_disambiguation, p_musicbrainz_score, p_is_reference
        ) RETURNING id INTO artist_id;
        
        RETURN artist_id;
    END IF;
END;
$$ LANGUAGE plpgsql;