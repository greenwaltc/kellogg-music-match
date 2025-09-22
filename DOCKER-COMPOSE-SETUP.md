# 🐳 Docker Compose Setup - Custom PostgreSQL with Scientific Extensions

## ✅ Complete Local Development Environment

This Docker Compose setup provides a complete local development environment with a **custom PostgreSQL image** featuring scientific extensions for advanced music similarity calculations. The setup mirrors production deployment with custom database functions, Go backend, and Angular frontend.

### 🧪 **Custom PostgreSQL Database Configuration**

```yaml
postgres:
  build:
    context: .
    dockerfile: postgres.dockerfile          # Custom build with scientific libraries
  image: kellogg-music-match-postgres:latest
  container_name: kmm-postgres
  environment:
    POSTGRES_USER: kellogg_user              # Production-matching credentials
    POSTGRES_PASSWORD: kellogg_secure_pass_2024
    POSTGRES_DB: kellogg_music_match
    PGDATA: /var/lib/postgresql/data/pgdata
  volumes:
    - ./DATABASE_SCHEMA.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro  # Auto-initialization
    - ./init-database.sh:/docker-entrypoint-initdb.d/02-init.sh:ro        # Auto-initialization
    - postgres_data:/var/lib/postgresql/data  # Persistent storage
  ports:
    - "5432:5432"                            # Host access for development
  healthcheck:                               # Service health monitoring
    test: ["CMD-SHELL", "pg_isready -U kellogg_user -d kellogg_music_match"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 30s
```

### 🔬 **Database Features**

The PostgreSQL database includes:

- **Flyway Migrations**: Professional database versioning system
- **PWO Distance Function**: Position-Weighted Overlap similarity algorithm  
- **TEXT Array Support**: Efficient handling for artist preference arrays
- **Performance Indexes**: Optimized for similarity calculations

**Custom Dockerfile (`postgres.dockerfile`):**
```dockerfile
FROM postgres:15

# Install Python dependencies as root
USER root
RUN apt-get update && apt-get install -y \
    python3-pip \
    python3-dev \
    python3-numpy \
    python3-scipy \
    postgresql-plpython3-15 \
    && rm -rf /var/lib/apt/lists/*

# Verify scientific libraries
RUN python3 -c "import scipy.stats; import numpy; print('✅ scipy and numpy are available')"

USER postgres
```

### 🔧 **Backend Integration with Scientific Functions**

The backend uses SQLC for type-safe database operations and integrates with the custom PostgreSQL scientific functions:

```yaml
backend:
  build: ./backend
  image: kellogg-music-match-backend:latest
  container_name: kmm-backend
  environment:
    DB_HOST: postgres                    # Docker Compose service name
    DB_PORT: 5432
    DB_NAME: kellogg_music_match
    DB_USER: kellogg_user
    DB_PASSWORD: kellogg_secure_pass_2024
    DB_SSLMODE: disable                 # Appropriate for local development
  depends_on:
    postgres:
      condition: service_healthy        # Wait for DB to be ready
  ports:
    - "8080:8080"                       # API access
```

**Backend Features:**
- **SQLC Integration**: Type-safe PostgreSQL operations
- **Custom Array Handling**: pq.StringArray for TEXT[] arrays
- **PWO Similarity**: Leverages pwo_distance function for Position-Weighted Overlap
- **Behavioral Testing**: Ginkgo tests for algorithm validation

### 🎨 **Frontend Integration**

```yaml
ui:
  build: 
    context: ./ui
    args:
      API_BASE_URL: http://localhost:8080
  image: kellogg-music-match-ui:latest
  container_name: kmm-ui
  ports:
    - "4200:80"                         # Frontend access
```

### 📊 **Development Workflow & Verification**

#### Quick Start Commands
```bash
# Start all services
make dev
# or
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs backend
docker-compose logs postgres

# Access database directly
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match
```

#### API Testing with Scientific Similarity
```bash
# Health check
curl http://localhost:8080/health

# User registration
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@kellogg.northwestern.edu","password":"TestPassword123!","firstName":"Test","lastName":"User","program":"2Y","graduationYear":2026}'

# User login
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"TestPassword123!"}'

# Music matching with scientific similarity calculation
curl -X POST http://localhost:8080/findMusicMatches \
  -H "Content-Type: application/json" \
  -H "X-User-Username: testuser" \
  -d '{"artists":["Tool","Radiohead"]}'

# Test PWO distance function directly
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT pwo_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead'], 0.5);"
```

#### Verification Results
- ✅ **PostgreSQL Database**: Running with Flyway migration system
- ✅ **PWO Functions**: pwo_distance and pwo_similarity operational
- ✅ **Database Connection**: PostgreSQL with SQLC integration and array handling
- ✅ **Migration System**: Flyway professional database versioning
- ✅ **UserRepository Pattern**: Clean database abstraction layer with PWO functions
- ✅ **Type Safety**: SQLC-generated Go code for all database operations
- ✅ **Behavioral Testing**: Ginkgo tests validating PWO algorithm accuracy
- ✅ **Backend Service**: Running and connected to database with PWO calculations
- ✅ **UI Service**: Running at http://localhost:4200
- ✅ **Health Checks**: PostgreSQL health monitoring working

### 🚀 **Development Workflow Commands**

```bash
# Build custom PostgreSQL image and start all services
make dev

# Start only database (for backend development)
make db-start

# Test behavioral algorithms
make test-behavioral

# Show service status
make status

# View logs
make logs

# Run all tests (Go + Ginkgo behavioral)
make test

# Stop and cleanup
make docker-clean
```

### 🧪 **PWO Function Testing**

```bash
# Test pwo_distance function with various scenarios
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT 'Identical' as scenario, pwo_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead'], 0.5) as distance;"

docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT 'Different order' as scenario, pwo_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Radiohead', 'Tool'], 0.5) as distance;"

docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT 'No overlap' as scenario, pwo_distance(ARRAY['Tool'], ARRAY['Beatles'], 0.5) as distance;"

# Check migration status
make db-info
```

### 🔄 **Local ↔ Kubernetes Consistency**

| Configuration | Docker Compose | Pulumi/Kubernetes |
|---------------|----------------|-------------------|
| **Database Image** | `postgres:15` | `postgres:15` |
| **Migration System** | Flyway | Flyway |
| **Database Name** | `kellogg_music_match` | `kellogg_music_match` |
| **Database User** | `kellogg_user` | `kellogg_user` |
| **Database Password** | `kellogg_secure_pass_2024` | `kellogg_secure_pass_2024` |
| **PWO Functions** | `pwo_distance`, `pwo_similarity` | `pwo_distance`, `pwo_similarity` |
| **Migration Source** | `database/migrations/` | `database/migrations/` (via ConfigMap) |
| **Data Persistence** | Named volume | StatefulSet PVC |

### 📋 **Database Connection Details**

For direct database access during development:

```bash
# Command line connection
psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match

# Connection string
postgresql://kellogg_user:kellogg_secure_pass_2024@localhost:5432/kellogg_music_match
```

### 🎯 **Ready for Advanced Development**

You can now:

1. **Test locally** with scientific similarity calculations using `make dev`
2. **Develop backend** with custom PostgreSQL functions using `make db-start`
3. **Run behavioral tests** to validate algorithm accuracy with `make test-behavioral`
4. **Deploy to Kubernetes** with `pulumi up` (same custom configuration)

The database schema, scientific functions, and algorithm behavior will be **identical** between local development and Kubernetes deployment! 🎵🗄️🧪