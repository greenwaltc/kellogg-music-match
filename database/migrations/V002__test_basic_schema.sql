-- Kellogg Music Match Database Schema
-- Initial database migration with all tables, functions, and constraints
-- Test version without plpython3u extension

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLES
-- ============================================================================

-- Users table with program and graduation year fields
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash TEXT NOT NULL,
    program VARCHAR(10) CHECK (program IN ('2Y', '1Y', 'MMM', 'MBAi', 'JD-MBA', 'MD-MBA', 'EWMBA', 'JV')),
    graduation_year INTEGER CHECK (graduation_year >= 2025 AND graduation_year <= 2030),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Artists table for normalization (240 char limit)
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(240) UNIQUE NOT NULL CHECK (char_length(name) <= 240),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Junction table for user-artist relationships with ranking
CREATE TABLE user_artists (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    rank SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id),
    CONSTRAINT user_rank_unique UNIQUE (user_id, rank)
);

-- Feedback table for user suggestions
CREATE TABLE feedback (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feedback_text TEXT NOT NULL CHECK (char_length(feedback_text) <= 1000 AND char_length(feedback_text) > 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Users table indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_program ON users(program);
CREATE INDEX idx_users_graduation_year ON users(graduation_year);

-- Artists table indexes
CREATE INDEX idx_artists_name ON artists(name);

-- User-artists junction table indexes
CREATE INDEX idx_user_artists_user_id ON user_artists(user_id);
CREATE INDEX idx_user_artists_artist_id ON user_artists(artist_id);

-- Feedback table indexes
CREATE INDEX idx_feedback_user_id ON feedback(user_id);
CREATE INDEX idx_feedback_created_at ON feedback(created_at);

-- speeds artist_id lookups when joining r1 and r2
CREATE INDEX IF NOT EXISTS user_artists_artist_user ON user_artists (artist_id, user_id);

-- speeds ordered scans for per-user ranks (if you frequently return lists)
CREATE INDEX IF NOT EXISTS user_artists_user_rank ON user_artists (user_id, rank);

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Alternative simpler implementation using basic statistics (fallback)
-- This creates a backup function in case plpython3u isn't available
CREATE OR REPLACE FUNCTION spearman_distance_simple(arr1 TEXT[], arr2 TEXT[])
RETURNS FLOAT
LANGUAGE plpgsql
AS $$
DECLARE
    intersection_count INTEGER;
    union_count INTEGER;
    jaccard_similarity FLOAT;
BEGIN
    -- Handle edge cases
    IF arr1 IS NULL AND arr2 IS NULL THEN
        RETURN 0.0;  -- Both null, perfect match
    END IF;
    
    IF arr1 IS NULL OR arr2 IS NULL THEN
        RETURN 1.0;  -- One null, maximum distance
    END IF;
    
    -- If arrays are identical, return perfect match
    IF arr1 = arr2 THEN
        RETURN 0.0;
    END IF;
    
    -- Calculate Jaccard similarity as a simple alternative
    -- intersection_count = |A ∩ B|
    SELECT COUNT(*)
    INTO intersection_count
    FROM (SELECT unnest(arr1) INTERSECT SELECT unnest(arr2)) AS intersection;
    
    -- union_count = |A ∪ B|
    SELECT COUNT(*)
    INTO union_count
    FROM (SELECT unnest(arr1) UNION SELECT unnest(arr2)) AS union_set;
    
    -- Avoid division by zero
    IF union_count = 0 THEN
        RETURN 0.0;
    END IF;
    
    -- Jaccard similarity = |A ∩ B| / |A ∪ B|
    jaccard_similarity := CAST(intersection_count AS FLOAT) / CAST(union_count AS FLOAT);
    
    -- Convert to distance (1 - similarity)
    RETURN 1.0 - jaccard_similarity;
END;
$$;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger to automatically update updated_at for users
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger to automatically update updated_at for feedback
CREATE TRIGGER update_feedback_updated_at BEFORE UPDATE ON feedback
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS
-- ============================================================================

-- Function comments
COMMENT ON FUNCTION spearman_distance_simple(TEXT[], TEXT[]) IS 
'Simple fallback implementation using Jaccard similarity for text arrays. Calculates 1 - (intersection / union) as distance metric. Pure PostgreSQL implementation without external dependencies.';

-- Column comments
COMMENT ON COLUMN users.program IS 'MBA program type: 2Y, 1Y, MBAi, MMM, EWMBA, JV';
COMMENT ON COLUMN users.graduation_year IS 'Expected graduation year (current year to 2030)';