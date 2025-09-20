# 🛠️ Development Workflow Guide

This guide covers the complete development workflow for the Kellogg Music Match application, including database management, code generation, and testing procedures.

## 🚀 Quick Start

### Prerequisites
- **Go 1.22+**
- **Node.js 18+**
- **Docker & Docker Compose**
- **Make**

### Start Development Environment
```bash
# Start all services (database, backend, frontend)
make dev

# Check service status
make status

# View logs
make logs
```

### Access Points
- **Frontend**: http://localhost:4200
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health
- **Database**: localhost:5432 (kellogg_user/kellogg_secure_pass_2024)

## 🗄️ Database Development

### Custom PostgreSQL Setup
The project uses a custom PostgreSQL image with scientific extensions:

```bash
# Build custom PostgreSQL image (includes plpython3u, scipy, numpy)
docker-compose build postgres

# Start database with scientific extensions
docker-compose up -d postgres

# Verify extensions are available
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "SELECT * FROM pg_available_extensions WHERE name='plpython3u';"
```

### Schema Management
The project uses a consolidated schema system with enhanced development pipeline:

```bash
# Schema Synchronization
make sync-schema           # Auto-sync DATABASE_SCHEMA.sql from backend/db/schema/*.sql
make check-schema-sync     # Verify schema files are synchronized

# Database Reset Pipeline (Enhanced Development Workflow)
make db-reset              # Complete database reset with guaranteed fresh schema
make db-schema-verify      # Verify database schema matches expected structure  
make db-force-schema-sync  # Nuclear option: complete reset with schema sync

# Traditional Database Operations
make db-start              # Start PostgreSQL database only
make db-connect            # Connect with psql interactive shell
make db-logs               # Show recent database logs
make db-backup             # Create timestamped backup
```

### Enhanced Reset Guarantees
The development pipeline provides reliable database management:

1. **Complete Volume Removal**: Docker volumes removed for truly fresh state
2. **Schema Synchronization**: Auto-syncs from `backend/db/schema/*.sql` before reset
3. **Structure Verification**: Validates all tables, indexes, and functions exist
4. **SQLC Regeneration**: Ensures Go code matches database structure

### Consolidated Schema Benefits
- **Single Source of Truth**: All schema in `backend/db/schema/001_initial.sql`
- **Complete Profiles**: Users include `program` and `graduation_year` fields
- **Kellogg Validation**: Program constraints (2Y, 1Y, MMM, MBAi, JD-MBA, etc.)
- **Docker Integration**: Schema auto-applied on container initialization

### Schema Development Workflow
```bash
# 1. Edit schema files in backend/db/schema/
vim backend/db/schema/001_initial.sql  # Main consolidated schema

# 2. Synchronize to main schema file (if adding new files)
make sync-schema

# 3. Generate SQLC code from updated schema
make backend-sqlc

# 4. Reset database with new schema
make db-reset

# 5. Verify schema was applied correctly
make db-schema-verify

# 6. Commit changes
git add backend/db/ DATABASE_SCHEMA.sql
git commit -m "Update database schema"
```

### Scientific Distance Function
The database includes a custom `spearman_distance` function for music similarity:

```sql
-- Test the hybrid similarity algorithm
SELECT spearman_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead']);  -- Returns 0 (identical)
SELECT spearman_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead']);              -- Returns ~0.7 (subset)
SELECT spearman_distance(ARRAY['Tool'], ARRAY['Beatles']);                        -- Returns 2.0 (no overlap)
```

### Database Operations
```bash
# Start database only
docker-compose up -d postgres

# Connect to database
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match

# View database logs
docker-compose logs postgres

# Reset database (removes all data)
docker-compose down -v
docker-compose up -d postgres
```

### SQLC Workflow
```bash
# Generate Go code from SQL queries
make backend-sqlc

# Edit queries in backend/db/queries/queries.sql
vim backend/db/queries/queries.sql

# Regenerate after query changes
make backend-sqlc
```

## 🔧 Backend Development

### Development Workflow
```bash
# Generate OpenAPI and SQLC code
make backend-generate

# Build backend
make backend-build

# Run with hot reload
make backend-dev

### Backend Testing
```bash
# Run all backend tests (Go + Ginkgo behavioral)
make backend-test

# Run quick Go unit tests only
make backend-test-quick

# Run Ginkgo behavioral tests
make backend-test-ginkgo

# Install testing dependencies
make backend-install-tools
```
```

### Code Generation
```bash
# Generate all code (OpenAPI + SQLC)
make backend-generate

# Generate only SQLC code
make backend-sqlc

# Generate only OpenAPI code
cd backend && make generate
```

### API Testing
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

# Music matching
curl -X POST http://localhost:8080/findMusicMatches \
  -H "Content-Type: application/json" \
  -H "X-User-Username: testuser" \
  -d '{"artists":["The Beatles","Taylor Swift"]}'
```

## 🎨 Frontend Development

### Development Workflow
```bash
# Start development server
make ui-dev

# Build for production
make ui-build

# Run tests
make ui-test

# Run linting
make ui-lint
```

### Angular Development
```bash
# Install dependencies
cd ui && npm install

# Start development server with live reload
cd ui && npm start

# Run tests
cd ui && npm test

# Build for production
cd ui && npm run build
```

## 🧪 Testing Strategy

### Comprehensive Test Framework
The project includes multiple testing approaches for thorough validation:

#### 1. **Go Unit Tests**
Traditional Go testing for business logic:
```bash
# Run quick unit tests
make test-quick
make backend-test-quick
```

#### 2. **Ginkgo Behavioral Tests**
Comprehensive behavioral testing using Ginkgo v2 + Gomega:
```bash
# Run all behavioral tests
make test-behavioral
make backend-test-ginkgo

# Run behavioral tests with verbose output
cd backend/business && ~/go/bin/ginkgo run -v .
```

**Behavioral Test Coverage (13 test specs):**
- **Music Matching Algorithm**: Validates Jaccard similarity calculations
- **Identical Preferences**: Score ≥ 0.9 for identical artist lists
- **Subset Relationships**: Score ≈ 0.5 for subset cases
- **Edge Cases**: Empty lists, normalization, caller exclusion
- **Scientific Accuracy**: Validates hybrid algorithm alignment with PostgreSQL

#### 3. **Database Function Testing**
Tests for PostgreSQL scientific functions:
```bash
# Test spearman_distance function directly
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT spearman_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead']);"
```

#### 4. **Integration Testing**
Full end-to-end testing:
```bash
# Run all tests (Go + Ginkgo + UI)
make test

# Test API endpoints
curl -X POST http://localhost:8080/findMusicMatches \
  -H "Content-Type: application/json" \
  -H "X-User-Username: alice" \
  -d '{"artists":["Tool", "Radiohead"]}'
```

### Testing Setup
```bash
# Install all testing dependencies
make install-tools

# Run full test suite
make test

# Run checks (lint + test + format)
make check
```

### Manual Testing for Algorithm Validation
```bash
# 1. Start services
make dev

# 2. Test similarity scenarios
curl -X POST http://localhost:8080/findMusicMatches \
  -H "Content-Type: application/json" \
  -H "X-User-Username: alice" \
  -d '{"artists":["Tool"]}'  # Should find bob with moderate similarity

# 3. Verify database state
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT u.username, ua.artist_name FROM users u JOIN user_artists ua ON u.id = ua.user_id ORDER BY u.username, ua.artist_name;"
```

### Manual Testing Checklist
- [ ] User registration works with proper UUID format
- [ ] User login validates credentials correctly
- [ ] Artist preferences are stored and retrieved
- [ ] Music matching finds users with shared artists using scientific similarity
- [ ] **Similarity scores are accurate**: Identical preferences (≥0.9), subsets (≈0.5), no overlap (no matches)
- [ ] **PostgreSQL spearman_distance function works**: Distance values 0, 0.7, 2.0 for test cases
- [ ] Database persistence verified with custom array handling
- [ ] Frontend connects to backend API
- [ ] All endpoints return expected HTTP status codes
- [ ] **Behavioral tests pass**: All 13 Ginkgo test specs validate algorithm behavior

## 🔄 Development Cycle

### Typical Development Flow
1. **Start Environment**: `make dev`
2. **Make Changes**: Edit code in backend/business/ or ui/src/
3. **Test Changes**: Use curl or frontend to test
4. **Check Database**: Verify data persistence
5. **Run Tests**: `make test`
6. **Commit**: Git commit with descriptive message

### Schema Changes Flow
1. **Edit Schema**: Modify files in backend/db/schema/ (especially 004_spearman_distance.sql for algorithm changes)
2. **Sync Schema**: `make schema-sync`
3. **Generate Code**: `make backend-sqlc`
4. **Update Queries**: Edit backend/db/queries/queries.sql if needed
5. **Regenerate**: `make backend-sqlc`
6. **Test**: Verify database operations work with `make test-behavioral`
7. **Commit**: Include schema files, generated code, and postgres.dockerfile

### Code Generation Flow
1. **Edit OpenAPI**: Modify backend/openapi.yaml
2. **Generate**: `make backend-generate`
3. **Implement**: Add business logic in backend/business/
4. **Test**: Verify endpoints work correctly
5. **Commit**: Include generated and business logic code

## 🔧 Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Check database logs
docker-compose logs postgres

# Restart database
docker-compose restart postgres
```

#### Code Generation Issues
```bash
# Clean and regenerate
make clean
make backend-generate

# Check for SQLC errors
cd backend && sqlc generate
```

#### Port Conflicts
```bash
# Check what's using ports
lsof -i :8080  # Backend port
lsof -i :4200  # Frontend port
lsof -i :5432  # Database port

# Stop conflicting services
docker-compose down
```

#### Build Issues
```bash
# Clean all build artifacts
make clean

# Rebuild everything
make build

# Check Docker images
docker images | grep kellogg-music-match
```

### Environment Variables
Ensure these environment variables are set correctly:

#### Backend (Docker Compose)
```bash
DB_HOST=postgres
DB_PORT=5432
DB_NAME=kellogg_music_match
DB_USER=kellogg_user
DB_PASSWORD=kellogg_secure_pass_2024
DB_SSLMODE=disable

# Note: Custom PostgreSQL image includes plpython3u extension
# and scientific libraries (scipy, numpy) for similarity calculations
```

#### Frontend (Docker Compose)
```bash
API_BASE_URL=http://localhost:8080
```

## 📚 Documentation

### Key Files
- **README.md**: Main project documentation
- **DATABASE.md**: Database setup and schema information
- **DATABASE_SCHEMA.md**: Comprehensive schema documentation with spearman_distance function
- **DOCKER-COMPOSE-SETUP.md**: Docker environment setup with custom PostgreSQL
- **backend/README.md**: Backend-specific documentation with testing framework
- **backend/business/TESTING.md**: Detailed Ginkgo behavioral testing documentation
- **ui/README.md**: Frontend-specific documentation
- **postgres.dockerfile**: Custom PostgreSQL image with scientific libraries

### Generated Documentation
- **Backend API**: http://localhost:8080/docs (when running)
- **Database Schema**: DATABASE_SCHEMA.sql (auto-generated)
- **SQLC Generated Code**: backend/db/sqlc/ (auto-generated)

### Development Scripts
- **Makefile**: Main orchestration and commands
- **backend/Makefile**: Backend-specific commands
- **dev.sh**: Legacy development script (use `make dev` instead)

This guide should be updated as the development workflow evolves. Always refer to the Makefile for the most current commands and targets.