-- Kellogg Music Match Database Schema
-- This file is automatically synchronized from backend/db/schema/*.sql files
-- DO NOT EDIT DIRECTLY - Make changes in backend/db/schema/ and run 'make sync-schema'
-- 
-- Schema files are processed in alphabetical order:
-- backend/db/schema/001_initial.sql
-- backend/db/schema/002_spearman_func.sql
-- backend/db/schema/003_spearman_nn_query.sql

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
-- From: backend/db/schema/003_spearman_nn_query.sql
-- -------------------------------------------------------------------------
ALTER TABLE user_artists
ADD COLUMN rank SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE user_artists
ADD CONSTRAINT user_rank_unique UNIQUE (user_id, rank);


