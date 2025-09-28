# 🛠️ Development Workflow Guide

This guide covers the complete development workflow for the Kellogg Music Match application, including database management, code generation, and testing procedures.

## 🚀 Quick Start

### Prerequisites
- **Go 1.23+**
- **Node.js 18+**
- **Docker & Docker Compose**
- **Make**
- **Ticketmaster API Key** (optional, for concert integration)

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

### PostgreSQL with Flyway Migrations
The project uses PostgreSQL with Flyway for professional database versioning:

```bash
# Start database
docker-compose up -d postgres

# Apply migrations
make db-migrate

# Check migration status
make db-info
```

### Migration Management
The project uses Flyway migration system with versioned SQL files:

```bash
# Create new migration
make create-migration name=add_feature

# Apply pending migrations
make db-migrate

# Reset database (drops and recreates with all migrations)
make db-reset

# View migration history
make db-info
```

### Current Migration Status
- **V001__initial_schema.sql**: Base tables, users, artists, user_artists
- **V002-V009**: Progressive feature additions and improvements  
- **V010__pwo_metric.sql**: Position-Weighted Overlap distance function
- **V011-V012**: MusicBrainz artist database integration (47,452 records)
- **V014**: Chamfer distance algorithm for enhanced similarity
- **V019**: Latest migration with MusicBrainz upsert function fixes

### Database Development Workflow
```bash
# 1. Create new migration file
make create-migration name=add_user_preferences

# 2. Edit the generated migration file
vim database/migrations/V011__add_user_preferences.sql

# 3. Apply migration
make db-migrate

# 4. Generate SQLC code if queries changed
make backend-sqlc

# 5. Test database changes
make backend-test

# 6. Commit changes
git add database/migrations/ backend/db/
git commit -m "Add user preferences feature"
```

### PWO Distance Function
The database includes a PWO (Position-Weighted Overlap) distance function for music similarity:

```sql
-- Test the PWO similarity algorithm
SELECT pwo_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Tool', 'Radiohead'], 0.5);  -- Returns 0.0 (identical)
SELECT pwo_distance(ARRAY['Tool', 'Radiohead'], ARRAY['Radiohead', 'Tool'], 0.5);  -- Returns small value (different order)
SELECT pwo_distance(ARRAY['Tool'], ARRAY['Beatles'], 0.5);                         -- Returns 1.0 (no overlap)
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

### Backend Testing (43 Passing Tests)
The project uses **Ginkgo BDD framework** with **Gomega matchers** for comprehensive testing:

```bash
# Run all backend tests (43 passing tests)
make backend-test

# Run tests with coverage
make backend-test-coverage

# Run specific test suites
cd backend && ginkgo run ./business/tests/

# Install testing dependencies  
make backend-install-tools
```

**Test Architecture:**
- **Ginkgo BDD**: Behavior-driven development with expressive test descriptions
- **Gomega Matchers**: Rich assertion library for readable test expectations
- **MockEventProvider**: Dependency injection for testing Ticketmaster integration
- **Test Coverage**: Comprehensive coverage of business logic and API endpoints
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

**Behavioral Test Coverage:**
- **Music Matching Algorithm**: Validates PWO (Position-Weighted Overlap) distance calculations
- **Identical Preferences**: Similarity = 1.0 for identical artist lists  
- **Different Preferences**: Lower similarity scores for divergent tastes
- **Edge Cases**: Empty lists, normalization, caller exclusion
- **Scientific Accuracy**: Validates PWO algorithm alignment with PostgreSQL

#### 3. **Database Function Testing**
Tests for PostgreSQL PWO distance function:
```bash
# Test pwo_distance function directly
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c \
  "SELECT pwo_distance(ARRAY['Tool'], ARRAY['Tool', 'Radiohead'], 0.5);"
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
- [ ] Music matching finds users with shared artists using PWO similarity
- [ ] **Similarity scores are accurate**: Identical preferences (1.0), different preferences (lower scores)
- [ ] **PostgreSQL pwo_distance function works**: Distance values 0.0-1.0 for test cases
- [ ] Database persistence verified with array handling
- [ ] Frontend connects to backend API
- [ ] All endpoints return expected HTTP status codes
- [ ] **Behavioral tests pass**: Ginkgo test specs validate PWO algorithm behavior

## 🔄 Development Cycle

### Typical Development Flow
1. **Start Environment**: `make dev`
2. **Make Changes**: Edit code in backend/business/ or ui/src/
3. **Test Changes**: Use curl or frontend to test
4. **Check Database**: Verify data persistence
5. **Run Tests**: `make test`
6. **Commit**: Git commit with descriptive message

### Schema Changes Flow
1. **Create Migration**: `make create-migration name=description`
2. **Edit Migration**: Modify the generated file in database/migrations/
3. **Apply Migration**: `make db-migrate`
4. **Generate Code**: `make backend-sqlc` (if queries changed)
5. **Test**: Verify database operations work with `make test`
6. **Commit**: Include migration files and generated code

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

# Note: PostgreSQL image includes PWO distance function
# for Position-Weighted Overlap similarity calculations
```

#### Frontend (Docker Compose)
```bash
API_BASE_URL=http://localhost:8080
```

## 📚 Documentation

### Key Files
- **README.md**: Main project documentation
- **DATABASE.md**: Database setup and schema information
- **DATABASE_SCHEMA.md**: Comprehensive schema documentation with PWO distance function
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