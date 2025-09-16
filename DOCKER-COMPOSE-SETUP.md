# 🐳 Docker Compose Setup - PostgreSQL Integration

## ✅ Complete Local Development Environment

This Docker Compose setup provides a complete local development environment that mirrors the production Kubernetes deployment with PostgreSQL database, Go backend, and Angular frontend.

### 🗄️ **PostgreSQL Database Configuration**

```yaml
postgres:
  image: postgres:15-alpine                    # Same as production
  container_name: kmm-postgres
  environment:
    POSTGRES_USER: kellogg_user               # Production-matching credentials
    POSTGRES_PASSWORD: kellogg_secure_pass_2024
    POSTGRES_DB: kellogg_music_match
    PGDATA: /var/lib/postgresql/data/pgdata
  volumes:
    - ./DATABASE_SCHEMA.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro  # Auto-initialization
    - ./init-database.sh:/docker-entrypoint-initdb.d/02-init.sh:ro        # Auto-initialization
    - postgres_data:/var/lib/postgresql/data  # Persistent storage
  ports:
    - "5432:5432"                             # Host access for development
  healthcheck:                                # Service health monitoring
    test: ["CMD-SHELL", "pg_isready -U kellogg_user -d kellogg_music_match"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 30s
```

### 🔧 **Backend Integration with SQLC**

The backend uses SQLC for type-safe database operations and receives proper environment variables:

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

#### API Testing
```bash
# Health check
curl http://localhost:8080/health

# User registration
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"TestPassword123!","firstName":"Test","lastName":"User"}'

# User login
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"TestPassword123!"}'

# Music matching
curl -X POST http://localhost:8080/findMusicMatches \
  -H "Content-Type: application/json" \
  -H "X-User-Username: testuser" \
  -d '{"artists":["The Beatles","Taylor Swift"]}'
```

#### Verification Results
- ✅ **Database Connection**: PostgreSQL with SQLC integration
- ✅ **Schema Management**: Multi-file schema system with synchronization
- ✅ **UserRepository Pattern**: Clean database abstraction layer
- ✅ **Type Safety**: SQLC-generated Go code for all database operations
- ✅ **Sample Data**: 10 artists and 3 users with relationships loaded
- ✅ **Backend Service**: Running and connected to database
- ✅ **UI Service**: Running at http://localhost:4200
- ✅ **Health Checks**: PostgreSQL health monitoring working

### 🚀 **Development Workflow Commands**

```bash
# Start all services
./dev.sh start

# Start only database (for backend development)
./dev.sh db-only

# Show service status
./dev.sh status

# View logs
./dev.sh logs

# Run tests
./dev.sh test

# Stop and cleanup
./dev.sh cleanup
```

### 🔄 **Local ↔ Kubernetes Consistency**

| Configuration | Docker Compose | Pulumi/Kubernetes |
|---------------|----------------|-------------------|
| **Database Image** | `postgres:15-alpine` | `postgres:15-alpine` |
| **Database Name** | `kellogg_music_match` | `kellogg_music_match` |
| **Database User** | `kellogg_user` | `kellogg_user` |
| **Database Password** | `kellogg_secure_pass_2024` | `kellogg_secure_pass_2024` |
| **Schema Source** | `DATABASE_SCHEMA.sql` | `DATABASE_SCHEMA.sql` (via ConfigMap) |
| **Initialization** | `init-database.sh` | `init-database.sh` (via ConfigMap) |
| **Data Persistence** | Named volume | StatefulSet PVC |

### 📋 **Database Connection Details**

For direct database access during development:

```bash
# Command line connection
psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match

# Connection string
postgresql://kellogg_user:kellogg_secure_pass_2024@localhost:5432/kellogg_music_match
```

### 🎯 **Ready for Development**

You can now:

1. **Test locally** with `./dev.sh start`
2. **Develop backend** with `./dev.sh db-only` (just database)
3. **Deploy to Kubernetes** with `pulumi up` (same configuration)

The database schema and data will be **identical** between local development and Kubernetes deployment! 🎵🗄️