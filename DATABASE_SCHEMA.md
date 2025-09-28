# 🗄️ Database Schema Documentation - Kellogg Music Match

## Overview

The Kellogg Music Match application uses **PostgreSQL 16** with Flyway migrations for professional database versioning. The database features a comprehensive migration system with PWO (Position-Weighted Overlap) distance function for scientifically accurate music taste matching between Kellogg students, integrated with **47,452 MusicBrainz artist records** for enhanced matching accuracy.

## 🏗️ Schema Architecture

### Flyway Migration Management
The database schema uses Flyway for professional database versioning:

- **Migrations**: `backend/db/schema/migrations/V001` through `V019` 
- **Current Version**: V019 includes MusicBrainz upsert function fixes
- **Migration Commands**: `make db-migrate`, `make db-reset`, `make create-migration`
- **Professional Versioning**: Incremental schema changes with rollback support
- **MusicBrainz Integration**: 47,452 deduplicated artist records (V011-V012)

### Key Features
- ✅ **Professional Migration System**: Flyway versioning with audit trail (V001-V019)
- ✅ **Complete User Profiles**: Program and graduation year fields including JV support
- ✅ **Enhanced Validation**: Program constraints for Kellogg programs (2Y, 1Y, MMM, JV, etc.)
- ✅ **SQLC Compatibility**: Type-safe Go code generation with latest queries
- ✅ **PWO Distance Function**: Scientific Position-Weighted Overlap algorithm
- ✅ **MusicBrainz Integration**: 47,452 deduplicated artist records for enhanced matching
- ✅ **Chamfer Distance Algorithm**: Advanced similarity measurement (V014)
- ✅ **Artist Neighbor Optimization**: Performance optimization for distance calculations
- ✅ **Feedback System**: User feedback collection and processing (V006)

## 🧪 Scientific Database Features

### PWO Distance Function
PostgreSQL function implementing Position-Weighted Overlap algorithm:

```sql
-- V010__pwo_metric.sql (current system at V019)
CREATE OR REPLACE FUNCTION pwo_distance(list1 TEXT[], list2 TEXT[], alpha FLOAT8 DEFAULT 0.5)
RETURNS FLOAT8 AS $$
DECLARE
    -- PWO (Position-Weighted Overlap) implementation
    intersection_count INTEGER := 0;
    weighted_overlap FLOAT8 := 0.0;
    union_count INTEGER;
    item TEXT;
    pos1 INTEGER;
    pos2 INTEGER;
    weight FLOAT8;
BEGIN
    -- Handle empty arrays
    IF array_length(list1, 1) IS NULL OR array_length(list2, 1) IS NULL THEN
        RETURN 1.0;
    END IF;

    -- Calculate weighted overlap for shared items
    FOREACH item IN ARRAY list1 LOOP
        pos2 := array_position(list2, item);
        IF pos2 IS NOT NULL THEN
            pos1 := array_position(list1, item);
            -- Position-based weight calculation with alpha parameter
            weight := 1.0 / (1.0 + alpha * ABS(pos1 - pos2));
            weighted_overlap := weighted_overlap + weight;
            intersection_count := intersection_count + 1;
        END IF;
    END LOOP;

    -- Calculate union size (total unique items)
    union_count := (
        SELECT COUNT(DISTINCT item) 
        FROM unnest(list1 || list2) AS item
    );

    -- PWO distance calculation
    IF union_count = 0 THEN
        RETURN 1.0;
    END IF;

    RETURN 1.0 - (weighted_overlap / union_count);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Helper function for similarity scoring  
CREATE OR REPLACE FUNCTION pwo_similarity(list1 TEXT[], list2 TEXT[], alpha FLOAT8 DEFAULT 0.5)
RETURNS FLOAT8 AS $$
BEGIN
    RETURN 1.0 - pwo_distance(list1, list2, alpha);
END;
$$ LANGUAGE plpgsql IMMUTABLE;
```

**Algorithm Details:**
- **Distance = 0.0**: Identical arrays (perfect similarity)
- **Distance = 1.0**: No overlap (no similarity)
- **Alpha Parameter**: Controls position sensitivity (higher alpha = more position-sensitive)
- **Weighted Overlap**: Positions closer in rank contribute more to similarity
- **Similarity Score**: 1.0 - distance for intuitive scoring (higher = more similar)

## 📊 Database Schema

### 🔧 Core Tables

#### `users` table
Stores user profile and authentication information.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true
);
```

**Key Features:**
- UUID primary key for distributed systems compatibility
- Unique constraints on username and email
- bcrypt password hash storage
- Automatic timestamp management
- Soft delete capability with `is_active` flag

#### `artists` table
Normalized storage for artist information.

```sql
CREATE TABLE artists (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    normalized_name VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

**Key Features:**
- Auto-incrementing integer ID for efficiency
- Original and normalized name storage for case-insensitive matching
- Automatic normalization via database triggers

#### `user_artists` table
Junction table implementing many-to-many relationship between users and artists.

```sql
CREATE TABLE user_artists (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, artist_id)
);
```

**Key Features:**
- Composite primary key prevents duplicate relationships
- Cascade deletes maintain referential integrity
- Timestamp tracking for preference history

### 📈 Performance Optimizations

#### Indexes
```sql
-- User table indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- Artist table indexes
CREATE INDEX idx_artists_name ON artists(name);
CREATE INDEX idx_artists_normalized_name ON artists(normalized_name);

-- Junction table indexes
CREATE INDEX idx_user_artists_user_id ON user_artists(user_id);
CREATE INDEX idx_user_artists_artist_id ON user_artists(artist_id);
CREATE INDEX idx_user_artists_added_at ON user_artists(added_at);
```

#### Database Functions & Triggers
```sql
-- Automatic timestamp updates
CREATE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Artist name normalization
CREATE FUNCTION normalize_artist_name(artist_name TEXT) RETURNS TEXT AS $$
BEGIN
    RETURN LOWER(TRIM(artist_name));
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Scientific similarity calculation (see "Scientific Database Features" section above)
-- The pwo_distance function provides accurate music taste similarity using
-- Position-Weighted Overlap algorithm for position-sensitive similarity scoring
```

### 👁️ Views for Common Queries

#### `user_profiles` view
Combines user data with their artist preferences:

```sql
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
```

## 🔄 API Data Mapping

### User Registration (`POST /register`)

**Request:** `RegisterRequest`
```json
{
  "username": "student123",
  "email": "student@kellogg.northwestern.edu",
  "firstName": "Jane",
  "lastName": "Doe",
  "password": "SecurePass123!"
}
```

**Database Operations:**
1. Insert into `users` table with bcrypt password hash
2. Return user profile without password

### User Authentication (`POST /login`)

**Request:** `LoginRequest`
```json
{
  "username": "student123",
  "password": "SecurePass123!"
}
```

**Database Operations:**
1. Query user by username
2. Verify password hash with bcrypt
3. Update `last_login` timestamp
4. Return user profile with artists from `user_profiles` view

### Music Matching (`POST /findMusicMatches`)

**Request:** `ArtistsRequest`
```json
{
  "artists": ["Taylor Swift", "The Beatles", "Radiohead"]
}
```

**Database Operations:**
1. Normalize and find/create artists in `artists` table
2. Query users with overlapping artists using PWO distance function
3. Calculate similarity scores using Position-Weighted Overlap algorithm
4. Return top matches sorted by score (converted from distance: score = 1.0 - distance)

**Example Query with PWO Function:**
```sql
-- Find similar users using pwo_distance function
SELECT 
    u.first_name || ' ' || u.last_name as name,
    COUNT(ua.artist_id) as overlap,
    1.0 - pwo_distance(
        target_artists.artists,
        ARRAY_AGG(a.name ORDER BY ua.added_at),
        0.5  -- alpha parameter for position sensitivity
    ) as score
FROM users u
JOIN user_artists ua ON u.id = ua.user_id
JOIN artists a ON ua.artist_id = a.id
CROSS JOIN (SELECT ARRAY['Taylor Swift', 'The Beatles'] as artists) target_artists
WHERE u.username != 'target_user'
GROUP BY u.id, u.first_name, u.last_name, target_artists.artists
HAVING COUNT(ua.artist_id) > 0
ORDER BY score DESC, overlap DESC;
```

## 🚀 Automatic Database Initialization

### Database Setup

The database schema is managed using **Flyway migrations** for professional versioning:

1. **Migration System**: Flyway manages incremental schema updates (V001-V019)
2. **PWO Function**: V010 migration includes Position-Weighted Overlap distance function
3. **MusicBrainz Integration**: V011-V012 migrations include 47,452 deduplicated artist records
4. **Advanced Algorithms**: V014+ includes Chamfer distance and performance optimizations
3. **Professional Versioning**: Audit trail and rollback support
4. **Development Commands**: `make db-migrate`, `make db-reset`, `make create-migration`

### Database Components

- **Tables**: users, artists, user_artists with UUID and foreign key constraints
- **Indexes**: Performance-optimized for common queries and UUID lookups
- **PWO Functions**: pwo_distance and pwo_similarity for Position-Weighted Overlap calculations
- **Standard Functions**: Timestamp updates and name normalization
- **Triggers**: Automatic field management
- **Views**: Common query patterns with artist aggregation
- **Migration System**: Professional database versioning with Flyway

## 📝 Common Database Queries

### Find User with Artists
```sql
SELECT * FROM user_profiles WHERE username = 'alice';
```

### Add Artist Preference
```sql
-- Insert artist if not exists
INSERT INTO artists (name) VALUES ('New Artist') 
ON CONFLICT (name) DO NOTHING;

-- Add user-artist relationship
INSERT INTO user_artists (user_id, artist_id) 
SELECT 
    (SELECT id FROM users WHERE username = 'alice'),
    (SELECT id FROM artists WHERE name = 'New Artist')
ON CONFLICT DO NOTHING;
```

### Test PWO Distance Function
```sql
-- Test pwo_distance function with various scenarios
SELECT 
    'Identical arrays' as scenario,
    pwo_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead'], 0.5) as distance;
-- Returns: 0.0 (perfect similarity)

SELECT 
    'Different order' as scenario,
    pwo_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Radiohead', 'Tool'], 0.5) as distance;
-- Returns: small positive value (position-sensitive)

SELECT 
    'No overlap' as scenario,
    pwo_distance(ARRAY['Tool'], ARRAY['Beatles'], 0.5) as distance;
-- Returns: 1.0 (no similarity)

-- Find music matches using PWO similarity
WITH user_target_artists AS (
    SELECT ARRAY['Tool', 'Radiohead'] as artists
),
similarity_scores AS (
    SELECT 
        u.id,
        u.first_name || ' ' || u.last_name as name,
        ARRAY_AGG(a.name ORDER BY ua.added_at) as user_artists,
        1.0 - pwo_distance(
            (SELECT artists FROM user_target_artists),
            ARRAY_AGG(a.name ORDER BY ua.added_at),
            0.5  -- alpha parameter for position sensitivity
        ) as score,
        COUNT(a.name) as total_artists
    FROM users u
    JOIN user_artists ua ON u.id = ua.user_id
    JOIN artists a ON ua.artist_id = a.id
    WHERE u.username != 'target_user' AND u.is_active = true
    GROUP BY u.id, u.first_name, u.last_name
    HAVING COUNT(a.name) > 0
)
SELECT name, score, total_artists as overlap
FROM similarity_scores
WHERE score > 0  -- Only return users with some similarity
ORDER BY score DESC, overlap DESC, name
LIMIT 5;
```

## 🔒 Security Features

### Password Security
- **bcrypt hashing** with automatic salt generation
- **No plaintext storage** of passwords
- **Password validation** enforced at application layer

### Data Integrity
- **Foreign key constraints** with cascade deletes
- **Unique constraints** on usernames and emails
- **Input validation** via CHECK constraints
- **Email format validation** using regex patterns

### Performance Security
- **Optimized indexes** for efficient queries
- **Normalized design** reduces data redundancy
- **Prepared statements** prevent SQL injection

## 🛠️ Database Access & Maintenance

### Development Access
```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match

# Test PWO distance functions
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT pwo_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead'], 0.5);"

# Check migration status
make db-info

# For Kubernetes deployment:
kubectl port-forward -n kellogg-music-match service/postgres 5432:5432
psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match
```

### Backup & Recovery
```bash
# Full database backup
kubectl exec -it postgres-0 -n kellogg-music-match -- pg_dump -U kellogg_user kellogg_music_match > backup.sql

# Restore from backup
kubectl exec -i postgres-0 -n kellogg-music-match -- psql -U kellogg_user kellogg_music_match < backup.sql
```

### Monitoring Queries
```sql
-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- User activity statistics
SELECT COUNT(*) as total_users, 
       COUNT(*) FILTER (WHERE last_login > CURRENT_DATE - INTERVAL '7 days') as active_users
FROM users WHERE is_active = true;

-- Popular artists ranking
SELECT a.name, COUNT(ua.user_id) as user_count
FROM artists a
LEFT JOIN user_artists ua ON a.id = ua.artist_id
GROUP BY a.id, a.name
ORDER BY user_count DESC
LIMIT 10;
```

## 🏗️ Infrastructure Integration

### Kubernetes Configuration
- **StatefulSet Deployment**: Ensures stable network identity
- **Persistent Volume**: 10Gi storage for data persistence
- **ConfigMap Integration**: Schema initialization script
- **Secret Management**: Secure credential storage
- **Health Checks**: Automated liveness and readiness probes

### Environment Variables
The backend receives these database connection variables:
```bash
DB_HOST=postgres.kellogg-music-match.svc.cluster.local
DB_PORT=5432
DB_NAME=kellogg_music_match
DB_USER=kellogg_user
DB_PASSWORD=[from postgres-secret]
DB_SSLMODE=disable
```

### Sample Artists Included
The initialization includes these artists for testing:
- Taylor Swift, The Beatles, Radiohead
- Beyoncé, Ed Sheeran, Adele  
- Drake, Billie Eilish, Post Malone, Ariana Grande

## 🎯 Migration Readiness

The database is now **fully prepared** for migrating from in-memory storage:

✅ **Schema Created**: All tables, indexes, and constraints in place  
✅ **Sample Data**: Test artists available immediately  
✅ **Performance Optimized**: Indexes and views for efficient queries  
✅ **Security Implemented**: Password hashing and data validation  
✅ **Auto-Initialization**: No manual setup required  
✅ **Cloud-Agnostic**: Works on any Kubernetes cluster  

---

🎵 **Database ready for production use!** 🗄️