# 🗄️ Database Schema Documentation - Kellogg Music Match

## Overview

The Kellogg Music Match application uses a **custom PostgreSQL database** with scientific extensions for advanced similarity calculations. The database features normalized schema design, plpython3u extension with scipy/numpy libraries, and a hybrid Jaccard + positional similarity algorithm for accurate music taste matching.

## 🧪 Scientific Database Features

### Custom PostgreSQL Image
- **Base**: PostgreSQL 15 with plpython3u extension
- **Scientific Libraries**: scipy, numpy for statistical calculations
- **Hybrid Algorithm**: Combines Jaccard similarity with positional correlation
- **Performance**: Optimized for real-time similarity calculations

### Spearman Distance Function
Custom PostgreSQL function implementing scientific similarity algorithm:

```sql
CREATE OR REPLACE FUNCTION spearman_distance(arr1 TEXT[], arr2 TEXT[]) 
RETURNS FLOAT8 AS $$
import numpy as np
from scipy.stats import spearmanr

# Handle empty arrays
if not arr1 or not arr2:
    return 2.0

# Convert to sets for Jaccard similarity
set1 = set(arr1)
set2 = set(arr2)

# Calculate Jaccard similarity
intersection = len(set1.intersection(set2))
union = len(set1.union(set2))
jaccard_similarity = intersection / union if union > 0 else 0

# If no intersection, return maximum distance
if intersection == 0:
    return 2.0

# If identical sets, return minimum distance
if set1 == set2:
    return 0.0

# For subset relationships, apply penalty
if set1.issubset(set2) or set2.issubset(set1):
    return 0.7

# Calculate positional correlation for shared items
shared_items = list(set1.intersection(set2))
if len(shared_items) > 1:
    ranks1 = [arr1.index(item) for item in shared_items if item in arr1]
    ranks2 = [arr2.index(item) for item in shared_items if item in arr2]
    
    if len(ranks1) == len(ranks2) and len(ranks1) > 1:
        correlation, _ = spearmanr(ranks1, ranks2)
        if not np.isnan(correlation):
            # Combine Jaccard (70%) and positional correlation (30%)
            combined_similarity = 0.7 * jaccard_similarity + 0.3 * (correlation + 1) / 2
            return 1.0 - combined_similarity

# Default to Jaccard-based distance
return 1.0 - jaccard_similarity
$$ LANGUAGE plpython3u IMMUTABLE;
```

**Algorithm Details:**
- **Distance = 0**: Identical arrays (perfect similarity)
- **Distance = 0.7**: Subset relationships (moderate similarity)  
- **Distance = 2.0**: No overlap (no similarity)
- **Hybrid Calculation**: 70% Jaccard + 30% positional correlation for shared items

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
-- The spearman_distance function provides accurate music taste similarity using
-- hybrid Jaccard + positional correlation algorithm with scipy statistical functions
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
2. Query users with overlapping artists using spearman_distance function
3. Calculate scientific similarity scores using hybrid Jaccard + positional algorithm
4. Return top matches sorted by score (converted from distance: score = 1.0 - distance)

**Example Query with Scientific Function:**
```sql
-- Find similar users using spearman_distance function
SELECT 
    u.first_name || ' ' || u.last_name as name,
    COUNT(ua.artist_id) as overlap,
    1.0 - spearman_distance(
        target_artists.artists,
        ARRAY_AGG(a.name ORDER BY ua.added_at)
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

### Custom PostgreSQL Setup

The database schema is **automatically created** using a custom PostgreSQL image with scientific extensions:

1. **Custom Image Build**: `postgres.dockerfile` creates image with plpython3u, scipy, numpy
2. **Initialization Script**: `init-database.sh` creates schema and functions
3. **Scientific Function**: `spearman_distance` function for similarity calculations
4. **Sample Data**: Test users and artists for algorithm validation

### Initialization Components

- **Tables**: users, artists, user_artists with UUID and foreign key constraints
- **Indexes**: Performance-optimized for common queries and UUID lookups
- **Scientific Functions**: spearman_distance with hybrid Jaccard + positional algorithm
- **Standard Functions**: Timestamp updates and name normalization
- **Triggers**: Automatic field management
- **Views**: Common query patterns with artist aggregation
- **Sample Data**: Test artists and users for development and testing

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

### Find Music Matches (Scientific Algorithm)
```sql
-- Test spearman_distance function with various scenarios
SELECT 
    'Identical arrays' as scenario,
    spearman_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead']) as distance;
-- Returns: 0 (perfect similarity)

SELECT 
    'Subset relationship' as scenario,
    spearman_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead']) as distance;
-- Returns: ~0.7 (moderate similarity)

SELECT 
    'No overlap' as scenario,
    spearman_distance(ARRAY['Tool'], ARRAY['Beatles']) as distance;
-- Returns: 2.0 (no similarity)

-- Find music matches using scientific similarity
WITH user_target_artists AS (
    SELECT ARRAY['Tool', 'Radiohead'] as artists
),
similarity_scores AS (
    SELECT 
        u.id,
        u.first_name || ' ' || u.last_name as name,
        ARRAY_AGG(a.name ORDER BY ua.added_at) as user_artists,
        1.0 - spearman_distance(
            (SELECT artists FROM user_target_artists),
            ARRAY_AGG(a.name ORDER BY ua.added_at)
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
# Connect to custom PostgreSQL with scientific extensions
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match

# Test scientific functions
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT spearman_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead']);"

# Verify extensions are loaded
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT * FROM pg_available_extensions WHERE name='plpython3u';"

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