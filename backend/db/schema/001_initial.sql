-- Kellogg Music Match Database Schema
-- Complete initial schema with all tables, functions, and constraints
-- PostgreSQL database schema
-- 
-- This directory (backend/db/schema/) is the SINGLE SOURCE OF TRUTH for database schema
-- All *.sql files in this directory are automatically consolidated into root DATABASE_SCHEMA.sql
-- Files are processed in alphabetical order (001_initial.sql, 002_add_features.sql, etc.)
-- 
-- To sync: make sync-schema (from project root)
-- Auto-sync: Runs automatically when SQLC generates (make generate-sqlc)
-- Validation: make check-schema-sync

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable plpython3u extension for advanced statistical functions
CREATE EXTENSION IF NOT EXISTS plpython3u;

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

-- Enhanced hybrid similarity function with size penalty for variable-length lists
-- This function calculates a distance metric based on artist preference similarity
-- Returns a distance where 0 = identical preferences, higher values = less similar
CREATE OR REPLACE FUNCTION spearman_distance(arr1 TEXT[], arr2 TEXT[])
RETURNS FLOAT
LANGUAGE plpython3u
AS $$
import scipy.stats
import numpy as np

# Convert PostgreSQL arrays to Python lists
list1 = arr1 if arr1 else []
list2 = arr2 if arr2 else []

# Handle edge cases
if len(list1) == 0 and len(list2) == 0:
    return 0.0  # Both empty, perfect match
    
if len(list1) == 0 or len(list2) == 0:
    return 1.0  # One empty, maximum distance

# If arrays are identical, return perfect match
if list1 == list2:
    return 0.0

# Calculate similarity using a hybrid approach that considers:
# 1. Jaccard similarity (intersection over union)
# 2. Positional similarity (rank correlation for shared items)
# 3. Size penalty for very different list lengths

# Calculate set operations
set1 = set(list1)
set2 = set(list2)
intersection = set1.intersection(set2)
union = set1.union(set2)

# If no common items, return maximum distance
if len(intersection) == 0:
    return 2.0

# Calculate Jaccard similarity
jaccard_similarity = len(intersection) / len(union)

# Calculate size penalty for very different list lengths
len1, len2 = len(list1), len(list2)
size_ratio = min(len1, len2) / max(len1, len2) if max(len1, len2) > 0 else 1.0
# Size penalty ranges from 0 (very different sizes) to 1 (same size)
# Apply penalty more strongly for extreme size differences
if size_ratio < 0.5:  # One list is more than 2x the other
    size_penalty = size_ratio * 0.5  # Stronger penalty
else:
    size_penalty = 0.5 + (size_ratio - 0.5)  # Gentle penalty

# For shared items, calculate positional similarity
if len(intersection) > 1:
    # Extract ranks of shared items in both lists
    shared_ranks1 = []
    shared_ranks2 = []
    
    for item in intersection:
        rank1 = list1.index(item) + 1  # 1-based ranking
        rank2 = list2.index(item) + 1
        shared_ranks1.append(rank1)
        shared_ranks2.append(rank2)
    
    # Calculate Spearman correlation for shared items only
    try:
        correlation, _ = scipy.stats.spearmanr(shared_ranks1, shared_ranks2)
        if np.isnan(correlation):
            correlation = 0.0
        positional_similarity = (correlation + 1.0) / 2.0  # Convert [-1,1] to [0,1]
    except:
        positional_similarity = 0.5  # Default middle value if calculation fails
else:
    # Only one shared item, perfect positional agreement
    positional_similarity = 1.0

# Combine Jaccard, positional similarities, and size penalty
# Adjust weights to account for size penalty
# For music preferences: Jaccard (overlap) is most important, 
# position matters less, and size differences should be penalized
combined_similarity = (0.6 * jaccard_similarity + 0.2 * positional_similarity) * size_penalty

# Convert to distance (0 = identical, 2 = completely different)
distance = 2.0 * (1.0 - combined_similarity)

return max(0.0, min(2.0, float(distance)))
$$;

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
COMMENT ON FUNCTION spearman_distance(TEXT[], TEXT[]) IS 
'Enhanced hybrid similarity function that calculates distance between two text arrays (artist names). Combines Jaccard similarity (60%), positional correlation (20%), and size penalty for variable-length lists. Returns 0 = identical rankings, 2 = completely different. Requires plpython3u extension with scipy.stats.';

COMMENT ON FUNCTION spearman_distance_simple(TEXT[], TEXT[]) IS 
'Simple fallback implementation using Jaccard similarity for text arrays. Calculates 1 - (intersection / union) as distance metric. Pure PostgreSQL implementation without external dependencies.';

-- Column comments
COMMENT ON COLUMN users.program IS 'MBA program type: 2Y, 1Y, MBAi, MMM, EWMBA, JV';
COMMENT ON COLUMN users.graduation_year IS 'Expected graduation year (current year to 2030)';


-- ============================================================================
-- Position-weighted Overlap Metric (PWO)
-- ============================================================================

-- PWO similarity between two users' ranked artist lists.
-- alpha in (0,1]; returns [0,1].
-- Uses artist_id & rank directly from user_artists (no need to materialize arrays).
CREATE OR REPLACE FUNCTION pwo_similarity(alpha double precision, u1 uuid, u2 uuid)
RETURNS double precision
LANGUAGE sql
STABLE
AS $$
WITH r1 AS (
  SELECT artist_id, rank
  FROM user_artists
  WHERE user_id = u1
),
r2 AS (
  SELECT artist_id, rank
  FROM user_artists
  WHERE user_id = u2
),
inter AS (
  -- intersection with both ranks
  SELECT r1.rank AS r1, r2.rank AS r2
  FROM r1
  JOIN r2 USING (artist_id)
),
lens AS (
  SELECT (SELECT count(*) FROM r1) AS n1,
         (SELECT count(*) FROM r2) AS n2
),
m AS (
  SELECT LEAST(n1, n2) AS m FROM lens
),
num AS (
  -- numerator: sum_x alpha^(r1(x)-1) * alpha^(r2(x)-1)
  SELECT COALESCE(SUM(POWER(alpha, (r1 - 1)) * POWER(alpha, (r2 - 1))), 0.0) AS num
  FROM inter
),
den AS (
  -- denominator: sum_{i=1..m} alpha^{2(i-1)}
  SELECT
    CASE
      WHEN (SELECT m FROM m) = 0 THEN 0.0
      WHEN alpha = 1 THEN (SELECT m FROM m)::double precision
      ELSE (1 - POWER(alpha * alpha, (SELECT m FROM m))) / (1 - alpha * alpha)
    END AS den
)
SELECT CASE WHEN den.den = 0 THEN 0.0 ELSE num.num / den.den END
FROM num, den;
$$;

-- returns double precision, so sqlc will generate float64
CREATE OR REPLACE FUNCTION pwo_distance(alpha float8, u1 uuid, u2 uuid)
RETURNS double precision
LANGUAGE sql
STABLE
AS $$
  SELECT 1::float8 - pwo_similarity(alpha, u1, u2)
$$;

-- Harmonic PWO: sum_x (1/r1(x))*(1/r2(x)) / sum_{i=1..m} (1/i^2)
CREATE OR REPLACE FUNCTION pwo_similarity_harmonic(u1 uuid, u2 uuid)
RETURNS double precision
LANGUAGE sql
STABLE
AS $$
WITH r1 AS (SELECT artist_id, rank FROM user_artists WHERE user_id = u1),
     r2 AS (SELECT artist_id, rank FROM user_artists WHERE user_id = u2),
     inter AS (SELECT r1.rank AS r1, r2.rank AS r2 FROM r1 JOIN r2 USING (artist_id)),
     lens AS (SELECT (SELECT count(*) FROM r1) AS n1, (SELECT count(*) FROM r2) AS n2),
     m AS (SELECT LEAST(n1, n2) AS m FROM lens),
     num AS (SELECT COALESCE(SUM(1.0/r1 * 1.0/r2), 0.0) AS num FROM inter),
     den AS (
       SELECT COALESCE(SUM(1.0/(i*i)), 0.0) AS den
       FROM generate_series(1, (SELECT m FROM m)) AS s(i)
     )
SELECT CASE WHEN den.den = 0 THEN 0.0 ELSE num.num / den.den END
FROM num, den;
$$;
