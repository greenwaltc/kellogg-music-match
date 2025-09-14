# 🗄️ Database Schema Documentation - Kellogg Music Match

## Overview

The Kellogg Music Match application uses a PostgreSQL database with a normalized schema designed for efficient storage and querying of user data and music preferences. The database is automatically initialized with the complete schema when the PostgreSQL container starts.

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
2. Query users with overlapping artists
3. Calculate similarity scores using SQL aggregations
4. Return top matches sorted by score

## 🚀 Automatic Database Initialization

### Schema Creation Process

The database schema is **automatically created** when the PostgreSQL StatefulSet starts:

1. **Initialization Script**: `init-database.sh` is embedded in the Kubernetes ConfigMap
2. **Auto-Execution**: Script runs in `/docker-entrypoint-initdb.d/` directory
3. **Idempotent Design**: Uses `CREATE TABLE IF NOT EXISTS` for safe re-runs
4. **Sample Data**: Includes 10 popular artists for testing

### Initialization Components

- **Tables**: users, artists, user_artists
- **Indexes**: Performance-optimized for common queries
- **Functions**: Timestamp updates and name normalization
- **Triggers**: Automatic field management
- **Views**: Common query patterns
- **Sample Data**: Test artists for development

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

### Find Music Matches (Complex Query)
```sql
WITH user_artists_query AS (
    SELECT artist_id FROM user_artists 
    WHERE user_id = (SELECT id FROM users WHERE username = 'alice')
),
match_scores AS (
    SELECT 
        u.id,
        u.first_name || ' ' || u.last_name as name,
        COUNT(DISTINCT ua.artist_id) as total_artists,
        COUNT(DISTINCT uaq.artist_id) as shared_artists,
        ROUND(
            COUNT(DISTINCT uaq.artist_id)::FLOAT / 
            NULLIF(COUNT(DISTINCT ua.artist_id) + 
                   (SELECT COUNT(*) FROM user_artists_query) - 
                   COUNT(DISTINCT uaq.artist_id), 0),
            3
        ) as jaccard_similarity
    FROM users u
    JOIN user_artists ua ON u.id = ua.user_id
    LEFT JOIN user_artists_query uaq ON ua.artist_id = uaq.artist_id
    WHERE u.username != 'alice' AND u.is_active = true
    GROUP BY u.id, u.first_name, u.last_name
    HAVING COUNT(DISTINCT uaq.artist_id) > 0
)
SELECT name, shared_artists as overlap, jaccard_similarity as score
FROM match_scores
ORDER BY jaccard_similarity DESC, shared_artists DESC, name
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
# Port forward PostgreSQL for local access
kubectl port-forward -n kellogg-music-match service/postgres 5432:5432

# Connect with psql
psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match

# Direct pod access
kubectl exec -it -n kellogg-music-match postgres-0 -- psql -U kellogg_user -d kellogg_music_match
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