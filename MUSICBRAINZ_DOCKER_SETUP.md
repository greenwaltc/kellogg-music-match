# MusicBrainz Artist Data Integration

## 🎯 **Overview**

The Docker Compose stack now includes automated loading of 50,000 reference artists from MusicBrainz to populate the database during startup. This ensures the application has a rich dataset of artists for matching and recommendations.

## 🏗️ **Architecture**

The MusicBrainz data loading is integrated into the Docker Compose stack with the following services:

```
postgres → flyway → musicbrainz-loader → backend → ui
```

### **Service Dependencies:**
1. **postgres** - Database service with health checks
2. **flyway** - Database schema migrations (creates tables and functions)
3. **musicbrainz-loader** - Loads 50k artists data (NEW)
4. **backend** - API service (waits for data loading to complete)
5. **ui** - Frontend interface

## 📁 **Files Modified**

### **docker-compose.yml**
- Added `musicbrainz-loader` service
- Updated `backend` service dependencies
- Configured automatic data loading on startup

### **Dockerfile.musicbrainz**
- Updated to use Python 3.11 Alpine base image
- Added PostgreSQL client and psycopg2 dependencies
- Includes MusicBrainz CSV data in container

### **Makefile**
- Updated `docker-build` target to include musicbrainz-loader service

## 🚀 **How It Works**

### **1. Database Initialization**
- PostgreSQL starts with health checks
- Flyway runs schema migrations including V012 (MusicBrainz table setup)

### **2. Data Loading Process**
The `musicbrainz-loader` service:
1. Waits for database migrations to complete
2. Executes Python script to load 50k artists
3. Uses batch processing (1000 artists per batch)
4. Provides progress updates during loading
5. Completes with success message

### **3. Application Startup**
- Backend service waits for data loading completion
- UI service starts after backend is ready
- Full application ready with populated database

## 🔧 **Usage**

### **Start Complete Stack:**
```bash
docker-compose up -d
```

### **Build All Services:**
```bash
make docker-build
# or
docker-compose build
```

### **View Loading Progress:**
```bash
docker-compose logs -f musicbrainz-loader
```

### **Check Data Loading Status:**
```bash
# Connect to database and check artist count
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;"
```

## 📊 **Expected Results**

After successful startup:
- **Total Artists**: ~50,000 reference artists from MusicBrainz
- **Data Quality**: High-quality, curated artist data with metadata
- **Performance**: Fast application startup with pre-populated database

## 🏃‍♂️ **Performance**

### **Loading Time**
- Typical loading time: 2-5 minutes for 50k artists
- Batch processing: 1000 artists per batch
- Progress reporting: Every batch completion

### **Resource Usage**
- **Memory**: ~512MB for musicbrainz-loader container
- **Storage**: ~5MB for CSV data
- **Database**: ~100MB for artist data

## 🔍 **Monitoring & Troubleshooting**

### **Check Service Status:**
```bash
docker-compose ps
```

### **View Logs:**
```bash
# All services
docker-compose logs

# Specific service
docker-compose logs musicbrainz-loader
docker-compose logs backend
```

### **Database Verification:**
```bash
# Check artist counts
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "
SELECT 
    COUNT(*) as total_artists,
    COUNT(*) FILTER (WHERE is_reference = TRUE) as reference_artists,
    COUNT(*) FILTER (WHERE is_reference = FALSE) as user_artists
FROM artists;
"

# Check loading completion
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "
SELECT COUNT(*) FROM artists WHERE is_reference = TRUE AND musicbrainz_id IS NOT NULL;
"
```

## 🛠️ **Data Management**

### **Re-load Data:**
If you need to reload the MusicBrainz data:

```bash
# Stop services
docker-compose down

# Remove data volume (CAUTION: This removes all data)
docker volume rm kellogg-music-match_postgres_data

# Start fresh
docker-compose up -d
```

### **Update Artist Data:**
To update to newer MusicBrainz data:
1. Replace `musicbrainz_artists_50k.csv` with new data
2. Rebuild musicbrainz-loader: `docker-compose build musicbrainz-loader`
3. Restart stack: `docker-compose down && docker-compose up -d`

## 🔐 **Security**

- Database credentials are environment variables
- No sensitive data exposed in containers
- Read-only volume mounts for data files
- Containers run with minimal privileges

## 🎯 **Benefits**

1. **Automated Setup**: No manual data loading required
2. **Consistent Environment**: Same data across all deployments
3. **Fast Startup**: Pre-populated database ready for use
4. **Quality Data**: Curated MusicBrainz artist information
5. **Scalable**: Easy to update or extend with more data

---

## 🚀 **Quick Start**

```bash
# Clone/navigate to project
cd kellogg-music-match

# Build all services (including new musicbrainz-loader)
docker-compose build

# Start complete stack with automatic data loading
docker-compose up -d

# Check progress
docker-compose logs -f musicbrainz-loader

# Access application once loading completes
# Frontend: http://localhost:4200
# Backend: http://localhost:8080

# Verify data loaded successfully
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "SELECT COUNT(*) FROM artists WHERE is_reference = TRUE;"
```

The application is now ready with a fully populated database of 50,000 reference artists! 🎉