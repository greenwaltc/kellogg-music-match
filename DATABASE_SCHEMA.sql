-- Kellogg Music Match Database Schema
-- This file is automatically synchronized from backend/db/schema/*.sql files
-- DO NOT EDIT DIRECTLY - Make changes in backend/db/schema/ and run 'make sync-schema'
-- 
-- Schema files are processed in alphabetical order:
-- backend/db/schema/001_initial.sql
-- backend/db/schema/002_spearman_func.sql
-- backend/db/schema/003_add_rank.sql
-- backend/db/schema/004_spearman_distance.sql

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

-- Create Spearman rank correlation distance function
-- This function calculates the Spearman rank correlation coefficient between two arrays
-- and returns 1 - correlation as a distance metric (0 = identical, 2 = completely opposite)
CREATE OR REPLACE FUNCTION spearman_distance(arr1 INTEGER[], arr2 INTEGER[])
RETURNS FLOAT
LANGUAGE plpython3u
AS $$
import scipy.stats
import numpy as np

# Convert PostgreSQL arrays to Python lists
list1 = arr1 if arr1 else []
list2 = arr2 if arr2 else []

# Handle edge cases
if len(list1) == 0 or len(list2) == 0:
    return 1.0  # Maximum distance for empty arrays

if len(list1) != len(list2):
    return 1.0  # Maximum distance for different length arrays

# Convert to numpy arrays
np_arr1 = np.array(list1, dtype=float)
np_arr2 = np.array(list2, dtype=float)

# Handle cases where all values are the same (no variance)
if np.var(np_arr1) == 0 or np.var(np_arr2) == 0:
    if np.array_equal(np_arr1, np_arr2):
        return 0.0  # Perfect match
    else:
        return 1.0  # Different constant values

try:
    # Calculate Spearman rank correlation coefficient
    correlation, p_value = scipy.stats.spearmanr(np_arr1, np_arr2)
    
    # Handle NaN result (shouldn't happen with above checks, but safety first)
    if np.isnan(correlation):
        return 1.0
    
    # Convert correlation to distance: distance = 1 - correlation
    # This gives us a distance metric where:
    # - 0 = perfect positive correlation (identical rankings)
    # - 1 = no correlation
    # - 2 = perfect negative correlation (opposite rankings)
    distance = 1.0 - correlation
    
    # Ensure distance is in valid range [0, 2]
    return max(0.0, min(2.0, float(distance)))
    
except Exception as e:
    # If any error occurs, return maximum distance
    plpy.warning(f"Error calculating Spearman distance: {str(e)}")
    return 1.0
$$;

-- Alternative simpler implementation using basic statistics (fallback)
-- This creates a backup function in case plpython3u isn't available
CREATE OR REPLACE FUNCTION spearman_distance_simple(arr1 INTEGER[], arr2 INTEGER[])
RETURNS FLOAT
LANGUAGE plpgsql
AS $$
DECLARE
    len1 INTEGER;
    len2 INTEGER;
    i INTEGER;
    sum_d_squared FLOAT := 0;
    n INTEGER;
    correlation FLOAT;
    distance FLOAT;
BEGIN
    -- Get array lengths
    len1 := COALESCE(array_length(arr1, 1), 0);
    len2 := COALESCE(array_length(arr2, 1), 0);
    
    -- Handle edge cases
    IF len1 = 0 OR len2 = 0 OR len1 != len2 THEN
        RETURN 1.0;
    END IF;
    
    n := len1;
    
    -- If arrays are identical, return perfect correlation (distance = 0)
    IF arr1 = arr2 THEN
        RETURN 0.0;
    END IF;
    
    -- Calculate sum of squared differences in ranks
    -- Note: This is a simplified version assuming arrays contain ranks directly
    FOR i IN 1..n LOOP
        sum_d_squared := sum_d_squared + POWER(arr1[i] - arr2[i], 2);
    END LOOP;
    
    -- Spearman correlation formula: ρ = 1 - (6 * Σd²) / (n * (n² - 1))
    IF n > 1 THEN
        correlation := 1.0 - (6.0 * sum_d_squared) / (n * (n * n - 1));
        distance := 1.0 - correlation;
        
        -- Ensure distance is in valid range [0, 2]
        RETURN GREATEST(0.0, LEAST(2.0, distance));
    ELSE
        RETURN 0.0;
    END IF;
END;
$$;

-- Create a comment explaining the function
COMMENT ON FUNCTION spearman_distance(INTEGER[], INTEGER[]) IS 
'Calculates Spearman rank correlation distance between two integer arrays. Returns 1 - correlation coefficient as distance metric (0 = identical rankings, 1 = no correlation, 2 = opposite rankings). Requires plpython3u extension with scipy.stats.';

COMMENT ON FUNCTION spearman_distance_simple(INTEGER[], INTEGER[]) IS 
'Simple fallback implementation of Spearman distance calculation using pure PostgreSQL. Less robust than the Python version but doesn''t require external dependencies.';
