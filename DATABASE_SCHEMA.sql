-- Kellogg Music Match Database Schema
-- This file is automatically synchronized from backend/db/schema/*.sql files
-- DO NOT EDIT DIRECTLY - Make changes in backend/db/schema/ and run 'make sync-schema'
-- 
-- Schema files are processed in alphabetical order:
-- backend/db/schema/001_initial.sql
-- backend/db/schema/002_spearman_func.sql
-- backend/db/schema/003_add_rank.sql
-- backend/db/schema/004_spearman_distance.sql
-- backend/db/schema/005_artist_name_limit.sql

-- ============================================================================
-- CONSOLIDATED SCHEMA (Auto-generated from backend/db/schema/*.sql)
-- ============================================================================

-- -------------------------------------------------------------------------
-- From: backend/db/schema/001_initial.sql
-- -------------------------------------------------------------------------
-- Schema for Kellogg Music Match application
-- PostgreSQL database schema
-- 
-- This directory (backend/db/schema/) is the SINGLE SOURCE OF TRUTH for database schema
-- All *.sql files in this directory are automatically consolidated into root DATABASE_SCHEMA.sql
-- Files are processed in alphabetical order (001_initial.sql, 002_add_features.sql, etc.)
-- 
-- To sync: make sync-schema (from project root)
-- Auto-sync: Runs automatically when SQLC generates (make generate-sqlc)
-- Validation: make check-schema-sync

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Artists table for normalization
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Junction table for user-artist relationships
CREATE TABLE user_artists (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id)
);

-- Indexes for performance
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_artists_name ON artists(name);
CREATE INDEX idx_user_artists_user_id ON user_artists(user_id);
CREATE INDEX idx_user_artists_artist_id ON user_artists(artist_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
-- -------------------------------------------------------------------------
-- From: backend/db/schema/002_spearman_func.sql
-- -------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS plpython3u;

CREATE OR REPLACE FUNCTION spearman_distance(list1 TEXT[], list2 TEXT[])
RETURNS FLOAT AS $$
    # 1. Find the union of all preferences
    all_items = set(list1) | set(list2)
    n = len(all_items)

    # If lists are too small, correlation is undefined. Return max distance.
    if n <= 1:
        return 2.0

    # 2. Create rank dictionaries for each list
    ranks1 = {item: i + 1 for i, item in enumerate(list1)}
    ranks2 = {item: i + 1 for i, item in enumerate(list2)}

    # 3. Calculate sum of squared differences (d^2)
    sum_sq_diff = 0
    for item in all_items:
        # Assign a penalty rank (n + 1) if an item is missing
        rank1 = ranks1.get(item, n + 1)
        rank2 = ranks2.get(item, n + 1)
        diff = rank1 - rank2
        sum_sq_diff += diff ** 2

    # 4. Calculate Spearman's rank correlation coefficient (rho)
    rho = 1 - (6 * sum_sq_diff) / (n * (n**2 - 1))

    # 5. Convert correlation to distance
    distance = 1 - rho
    return distance

$$ LANGUAGE plpython3u;
-- -------------------------------------------------------------------------
-- From: backend/db/schema/003_add_rank.sql
-- -------------------------------------------------------------------------
ALTER TABLE user_artists
ADD COLUMN rank SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE user_artists
ADD CONSTRAINT user_rank_unique UNIQUE (user_id, rank);


-- -------------------------------------------------------------------------
-- From: backend/db/schema/004_spearman_distance.sql
-- -------------------------------------------------------------------------
-- Enable plpython3u extension for advanced statistical functions
CREATE EXTENSION IF NOT EXISTS plpython3u;

-- Create Spearman rank correlation distance function for text arrays (artist names)
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

# Combine Jaccard and positional similarities
# Weight Jaccard more heavily as it's more important for music preferences
combined_similarity = 0.7 * jaccard_similarity + 0.3 * positional_similarity

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

-- Create a comment explaining the function
COMMENT ON FUNCTION spearman_distance(TEXT[], TEXT[]) IS 
'Calculates Spearman rank correlation distance between two text arrays (artist names). Returns 1 - correlation coefficient as distance metric (0 = identical rankings, 1 = no correlation, 2 = opposite rankings). Requires plpython3u extension with scipy.stats.';

COMMENT ON FUNCTION spearman_distance_simple(TEXT[], TEXT[]) IS 
'Simple fallback implementation using Jaccard similarity for text arrays. Calculates 1 - (intersection / union) as distance metric. Pure PostgreSQL implementation without external dependencies.';
-- -------------------------------------------------------------------------
-- From: backend/db/schema/005_artist_name_limit.sql
-- -------------------------------------------------------------------------
-- Migration to update artist name limit to 240 characters
-- This ensures consistency with the frontend character limit

-- First, check if there are any existing artists longer than 240 characters
-- (This query is for information only, will be removed in production)
-- SELECT name, length(name) FROM artists WHERE length(name) > 240;

-- Update the artists table to enforce 240 character limit
ALTER TABLE artists ALTER COLUMN name TYPE VARCHAR(240);

-- Add constraint to ensure artist names don't exceed 240 characters
ALTER TABLE artists ADD CONSTRAINT check_artist_name_length CHECK (char_length(name) <= 240);
