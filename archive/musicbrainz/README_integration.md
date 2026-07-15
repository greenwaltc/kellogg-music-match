# MusicBrainz Artist Database Integration

Complete implementation for fetching MusicBrainz artists and populating your PostgreSQL database via Flyway migrations.

## 🎯 Overview

This system provides:
- **50,000+ high-quality artists** from MusicBrainz (scored by popularity)
- **Rich metadata**: Countries, types, lifespans, disambiguation
- **Seamless integration** with existing user-submitted artists
- **Flyway migration support** for production deployments

## 📁 Files Created

### Scripts
- `scripts/fetch_musicbrainz_artists.py` - Fetch artists from MusicBrainz API
- `scripts/load_musicbrainz_artists.py` - Python loader (requires psycopg2)
- `scripts/load_artists_k8s.sh` - Kubernetes-native loader (recommended)
- `scripts/requirements.txt` - Python dependencies

### Database
- `backend/db/schema/migrations/V011__musicbrainz_artists.sql` - Schema migration

### Data Files
- `musicbrainz_artists_1k.json` - Sample 1,000 artists (JSON)
- `musicbrainz_artists_1k_converted.csv` - Sample 1,000 artists (CSV)

## 🚀 Quick Start

### 1. Fetch Artists from MusicBrainz

```bash
# Fetch 10,000 high-quality artists (recommended for production)
python3 scripts/fetch_musicbrainz_artists.py --max-artists 10000 --format csv --output artists_10k.csv

# For comprehensive coverage (takes several hours)
python3 scripts/fetch_musicbrainz_artists.py --max-artists 50000 --format csv --output artists_50k.csv
```

### 2. Apply Database Migration

The migration is already applied, but for new deployments:

```bash
# Via Flyway (production)
cd backend && flyway migrate

# Via direct SQL (development)
kubectl exec -i postgres-0 -n kmm -- psql -U kellogg_user -d kellogg_music_match < backend/db/schema/migrations/V011__musicbrainz_artists.sql
```

### 3. Load Artists into Database

```bash
# Using Kubernetes script (recommended)
./scripts/load_artists_k8s.sh artists_10k.csv

# Using Python script (requires port-forward)
kubectl port-forward -n kmm svc/postgres 5432:5432 &
python3 scripts/load_musicbrainz_artists.py artists_10k.csv
```

## 📊 Current Status

### Sample Data Loaded (1,000 artists)
- **Total artists in DB**: 1,001 (1,000 reference + 1 user)
- **Geographic diversity**: 10+ countries (US: 392, GB: 177, DE: 103, etc.)
- **Type diversity**: Persons (672), Groups (288), Orchestras (12), etc.
- **Score range**: 100 (Various Artists) down to 66 (specialized artists)

## 🗄️ Database Schema

### Enhanced Artists Table
```sql
-- Core fields (existing)
id SERIAL PRIMARY KEY
name VARCHAR(240) NOT NULL
created_at TIMESTAMPTZ DEFAULT NOW()

-- MusicBrainz fields (new)
musicbrainz_id UUID                 -- MusicBrainz unique identifier
sort_name VARCHAR(240)              -- Sortable name format
artist_type VARCHAR(50)             -- Person, Group, Orchestra, etc.
gender VARCHAR(20)                  -- For persons
country CHAR(2)                     -- ISO country code
life_span_begin DATE                -- Birth/formation date
life_span_end DATE                  -- Death/dissolution date
disambiguation TEXT                 -- Distinguishing information
musicbrainz_score INTEGER           -- Popularity/relevance score
is_reference BOOLEAN DEFAULT FALSE  -- TRUE for MusicBrainz data
```

### Key Features
- **Dual data sources**: User-submitted + MusicBrainz reference data
- **Smart upserts**: Merges user data with MusicBrainz metadata
- **Optimized indexes**: Fast searching by name, country, type, score
- **Views**: Separate views for reference vs user artists

## 🔧 Integration with Application

### 1. Enhanced Artist Search

```sql
-- Search with metadata-enhanced results
SELECT 
    a.name,
    a.country,
    a.artist_type,
    a.musicbrainz_score,
    a.disambiguation
FROM artists a 
WHERE a.name ILIKE '%beatles%'
ORDER BY a.musicbrainz_score DESC NULLS LAST;
```

### 2. Smart Artist Validation

```sql
-- Check if user-entered artist exists in reference data
SELECT 
    a.id,
    a.name,
    CASE WHEN a.is_reference THEN 'verified' ELSE 'user-submitted' END as status,
    a.disambiguation
FROM artists a 
WHERE a.name = 'Taylor Swift';
```

### 3. Geographic Analytics

```sql
-- Popular artists by country
SELECT 
    a.country,
    COUNT(*) as artist_count,
    AVG(a.musicbrainz_score) as avg_score
FROM artists a 
WHERE a.is_reference = TRUE AND a.country IS NOT NULL
GROUP BY a.country 
ORDER BY avg_score DESC;
```

## 📈 Performance & Scaling

### API Rate Limiting
- **MusicBrainz limit**: ~100 requests/minute
- **Recommended delay**: 0.3-1.0 seconds between requests
- **50k artists**: ~2.5-4 hours to fetch

### Database Performance
- **Indexes**: Optimized for name, score, country, type searches
- **Partitioning**: Consider partitioning by `is_reference` for very large datasets
- **Caching**: High-score artists rarely change, good for caching

### Storage Requirements
- **1,000 artists**: ~1MB database storage
- **10,000 artists**: ~10MB database storage  
- **50,000 artists**: ~50MB database storage

## 🛠️ Maintenance & Updates

### Regular Updates
```bash
# Monthly update of top artists
python3 scripts/fetch_musicbrainz_artists.py --max-artists 10000 --format csv --output artists_update.csv
./scripts/load_artists_k8s.sh artists_update.csv
```

### Monitoring Queries
```sql
-- Check data freshness
SELECT 
    COUNT(*) as total_reference_artists,
    MAX(created_at) as last_added_reference,
    COUNT(*) FILTER (WHERE musicbrainz_score >= 80) as high_score_artists
FROM artists 
WHERE is_reference = TRUE;

-- Check user vs reference ratio
SELECT 
    is_reference,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 1) as percentage
FROM artists 
GROUP BY is_reference;
```

## 🔄 Integration with Flyway

### Production Deployment

1. **Add to Flyway**: The `V011__musicbrainz_artists.sql` migration is ready
2. **Data seeding**: Use init containers or post-deployment jobs
3. **Kubernetes job** for loading artists:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: load-musicbrainz-artists
spec:
  template:
    spec:
      containers:
      - name: artist-loader
        image: your-app:latest
        command: ["/scripts/load_artists_k8s.sh", "/data/artists_10k.csv"]
        volumeMounts:
        - name: artist-data
          mountPath: /data
      volumes:
      - name: artist-data
        configMap:
          name: musicbrainz-artist-data
```

## 🎵 Benefits for Kellogg Music Match

1. **Improved Matching**: Better artist normalization and disambiguation
2. **Enhanced UX**: Auto-complete with popular artists first
3. **Data Quality**: Standardized spellings and metadata
4. **Analytics**: Geographic and demographic insights
5. **Scalability**: Foundation for million+ artist database

## 📋 Next Steps

1. **Fetch larger dataset**: Run overnight for 50k+ artists
2. **Integrate with UI**: Update artist search/autocomplete
3. **Analytics dashboard**: Show user preferences by country/type
4. **Matching algorithm**: Use metadata for better similarity scoring
5. **Regular updates**: Schedule monthly artist database refreshes