-- Migration to enhance artists table with MusicBrainz data
-- Adds MusicBrainz metadata fields and creates reference artist data

-- Add MusicBrainz fields to existing artists table
ALTER TABLE artists ADD COLUMN IF NOT EXISTS musicbrainz_id UUID;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS sort_name VARCHAR(240);
ALTER TABLE artists ADD COLUMN IF NOT EXISTS artist_type VARCHAR(50);
ALTER TABLE artists ADD COLUMN IF NOT EXISTS gender VARCHAR(20);
ALTER TABLE artists ADD COLUMN IF NOT EXISTS country CHAR(2);
ALTER TABLE artists ADD COLUMN IF NOT EXISTS life_span_begin DATE;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS life_span_end DATE;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS disambiguation TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS musicbrainz_score INTEGER;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS is_reference BOOLEAN DEFAULT FALSE;

-- Add indexes for MusicBrainz fields
CREATE INDEX IF NOT EXISTS idx_artists_musicbrainz_id ON artists(musicbrainz_id);
CREATE INDEX IF NOT EXISTS idx_artists_type ON artists(artist_type);
CREATE INDEX IF NOT EXISTS idx_artists_country ON artists(country);
CREATE INDEX IF NOT EXISTS idx_artists_score ON artists(musicbrainz_score DESC);
CREATE INDEX IF NOT EXISTS idx_artists_reference ON artists(is_reference);

-- Add unique constraint for MusicBrainz ID (where not null)
CREATE UNIQUE INDEX IF NOT EXISTS idx_artists_musicbrainz_id_unique 
ON artists(musicbrainz_id) WHERE musicbrainz_id IS NOT NULL;

-- Create view for reference artists (MusicBrainz sourced)
CREATE OR REPLACE VIEW reference_artists AS
SELECT 
    id,
    name,
    sort_name,
    artist_type,
    gender,
    country,
    life_span_begin,
    life_span_end,
    disambiguation,
    musicbrainz_score,
    musicbrainz_id
FROM artists 
WHERE is_reference = TRUE 
ORDER BY musicbrainz_score DESC NULLS LAST, sort_name;

-- Create view for user artists (user-submitted)
CREATE OR REPLACE VIEW user_submitted_artists AS
SELECT 
    id,
    name,
    created_at
FROM artists 
WHERE is_reference = FALSE 
ORDER BY created_at DESC;

-- Function to find or create artist with MusicBrainz data
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
            -- Update existing artist with new data
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
                is_reference = p_is_reference,
                updated_at = NOW()
            WHERE id = artist_id;
            
            RETURN artist_id;
        END IF;
    END IF;
    
    -- Try to find by name if no MusicBrainz ID match
    SELECT id INTO artist_id 
    FROM artists 
    WHERE name = p_name;
    
    IF FOUND THEN
        -- Update existing artist, preserving user data if this is reference data
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
            is_reference = CASE WHEN p_is_reference THEN TRUE ELSE is_reference END,
            updated_at = NOW()
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