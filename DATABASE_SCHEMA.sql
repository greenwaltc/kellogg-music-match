-- Kellogg Music Match Database Schema
-- PostgreSQL database schema for user management and music taste matching

-- ============================================================================
-- EXTENSION AND CONFIGURATION
-- ============================================================================

-- Enable UUID generation extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Set timezone for consistent timestamps
SET timezone = 'UTC';

-- ============================================================================
-- TABLES
-- ============================================================================

-- Users table: Core user information and authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- bcrypt hash
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    
    -- Constraints
    CONSTRAINT users_username_length CHECK (LENGTH(username) >= 1),
    CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT users_name_length CHECK (LENGTH(first_name) >= 1 AND LENGTH(last_name) >= 1)
);

-- Artists table: Normalized artist storage
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    normalized_name VARCHAR(255) UNIQUE NOT NULL, -- Lowercase, trimmed for matching
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT artists_name_length CHECK (LENGTH(TRIM(name)) >= 1)
);

-- User-Artist junction table: Many-to-many relationship
CREATE TABLE user_artists (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Composite primary key prevents duplicates
    PRIMARY KEY (user_id, artist_id)
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Users table indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- Artists table indexes
CREATE INDEX idx_artists_name ON artists(name);
CREATE INDEX idx_artists_normalized_name ON artists(normalized_name);

-- User-Artists junction table indexes
CREATE INDEX idx_user_artists_user_id ON user_artists(user_id);
CREATE INDEX idx_user_artists_artist_id ON user_artists(artist_id);
CREATE INDEX idx_user_artists_added_at ON user_artists(added_at);

-- ============================================================================
-- FUNCTIONS AND TRIGGERS
-- ============================================================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at on users table
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to normalize artist names
CREATE OR REPLACE FUNCTION normalize_artist_name(artist_name TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(TRIM(artist_name));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger to automatically set normalized_name on artists
CREATE OR REPLACE FUNCTION set_artist_normalized_name()
RETURNS TRIGGER AS $$
BEGIN
    NEW.normalized_name = normalize_artist_name(NEW.name);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_artist_normalized_name
    BEFORE INSERT OR UPDATE ON artists
    FOR EACH ROW
    EXECUTE FUNCTION set_artist_normalized_name();

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- View to get users with their artist preferences
CREATE VIEW user_profiles AS
SELECT 
    u.id,
    u.username,
    u.email,
    u.first_name,
    u.last_name,
    u.created_at,
    u.last_login,
    u.is_active,
    COALESCE(
        ARRAY_AGG(a.name ORDER BY ua.added_at) FILTER (WHERE a.name IS NOT NULL),
        ARRAY[]::VARCHAR[]
    ) as artists
FROM users u
LEFT JOIN user_artists ua ON u.id = ua.user_id
LEFT JOIN artists a ON ua.artist_id = a.id
WHERE u.is_active = true
GROUP BY u.id, u.username, u.email, u.first_name, u.last_name, u.created_at, u.last_login, u.is_active;

-- View for music matching analysis
CREATE VIEW user_artist_overlap AS
SELECT 
    u1.id as user1_id,
    u1.username as user1_username,
    u1.first_name || ' ' || u1.last_name as user1_name,
    u2.id as user2_id,
    u2.username as user2_username,
    u2.first_name || ' ' || u2.last_name as user2_name,
    COUNT(DISTINCT ua1.artist_id) as user1_artist_count,
    COUNT(DISTINCT ua2.artist_id) as user2_artist_count,
    COUNT(DISTINCT CASE WHEN ua1.artist_id = ua2.artist_id THEN ua1.artist_id END) as shared_artists,
    ROUND(
        COUNT(DISTINCT CASE WHEN ua1.artist_id = ua2.artist_id THEN ua1.artist_id END)::FLOAT / 
        NULLIF(COUNT(DISTINCT ua1.artist_id) + COUNT(DISTINCT ua2.artist_id) - 
               COUNT(DISTINCT CASE WHEN ua1.artist_id = ua2.artist_id THEN ua1.artist_id END), 0),
        3
    ) as jaccard_similarity
FROM users u1
CROSS JOIN users u2
LEFT JOIN user_artists ua1 ON u1.id = ua1.user_id
LEFT JOIN user_artists ua2 ON u2.id = ua2.user_id
WHERE u1.id != u2.id 
  AND u1.is_active = true 
  AND u2.is_active = true
GROUP BY u1.id, u1.username, u1.first_name, u1.last_name, u2.id, u2.username, u2.first_name, u2.last_name
HAVING COUNT(DISTINCT ua1.artist_id) > 0 AND COUNT(DISTINCT ua2.artist_id) > 0;

-- ============================================================================
-- SAMPLE DATA (Optional - for development/testing)
-- ============================================================================

-- Insert sample users (passwords are bcrypt hashes for "password123")
INSERT INTO users (username, email, first_name, last_name, password_hash) VALUES
('alice', 'alice@example.com', 'Alice', 'Johnson', '$2a$10$rQ3Bz8zKjP9XnF5L7yKjUeVGH6M8xR9nQ4P2wS1oE7yT3cB4vL6sA'),
('bob', 'bob@example.com', 'Bob', 'Smith', '$2a$10$rQ3Bz8zKjP9XnF5L7yKjUeVGH6M8xR9nQ4P2wS1oE7yT3cB4vL6sA'),
('charlie', 'charlie@example.com', 'Charlie', 'Brown', '$2a$10$rQ3Bz8zKjP9XnF5L7yKjUeVGH6M8xR9nQ4P2wS1oE7yT3cB4vL6sA');

-- Insert sample artists
INSERT INTO artists (name) VALUES
('Taylor Swift'),
('The Beatles'),
('Radiohead'),
('Beyoncé'),
('Ed Sheeran'),
('Adele'),
('Drake'),
('Billie Eilish'),
('Post Malone'),
('Ariana Grande');

-- Create sample user-artist relationships
INSERT INTO user_artists (user_id, artist_id) VALUES
-- Alice likes Taylor Swift, The Beatles, Radiohead
((SELECT id FROM users WHERE username = 'alice'), (SELECT id FROM artists WHERE name = 'Taylor Swift')),
((SELECT id FROM users WHERE username = 'alice'), (SELECT id FROM artists WHERE name = 'The Beatles')),
((SELECT id FROM users WHERE username = 'alice'), (SELECT id FROM artists WHERE name = 'Radiohead')),

-- Bob likes The Beatles, Beyoncé, Ed Sheeran  
((SELECT id FROM users WHERE username = 'bob'), (SELECT id FROM artists WHERE name = 'The Beatles')),
((SELECT id FROM users WHERE username = 'bob'), (SELECT id FROM artists WHERE name = 'Beyoncé')),
((SELECT id FROM users WHERE username = 'bob'), (SELECT id FROM artists WHERE name = 'Ed Sheeran')),

-- Charlie likes Radiohead, Billie Eilish, Post Malone
((SELECT id FROM users WHERE username = 'charlie'), (SELECT id FROM artists WHERE name = 'Radiohead')),
((SELECT id FROM users WHERE username = 'charlie'), (SELECT id FROM artists WHERE name = 'Billie Eilish')),
((SELECT id FROM users WHERE username = 'charlie'), (SELECT id FROM artists WHERE name = 'Post Malone'));

-- ============================================================================
-- VERIFICATION QUERIES
-- ============================================================================

-- Check all users and their artists
-- SELECT * FROM user_profiles ORDER BY username;

-- Find music matches for a specific user
-- SELECT 
--     user2_name as match_name,
--     shared_artists as overlap,
--     jaccard_similarity as score
-- FROM user_artist_overlap 
-- WHERE user1_username = 'alice' 
--   AND shared_artists > 0
-- ORDER BY jaccard_similarity DESC, shared_artists DESC, user2_name;

-- Get artist popularity (how many users like each artist)
-- SELECT 
--     a.name,
--     COUNT(ua.user_id) as user_count
-- FROM artists a
-- LEFT JOIN user_artists ua ON a.id = ua.artist_id
-- GROUP BY a.id, a.name
-- ORDER BY user_count DESC, a.name;