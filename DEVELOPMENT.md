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

### Schema Management
The project uses a multi-file schema system with automatic synchronization:

```bash
# 1. Edit schema files in backend/db/schema/
vim backend/db/schema/001_initial.sql

# 2. Synchronize to main schema file
make schema-sync

# 3. Generate SQLC code
make backend-sqlc

# 4. Commit changes
git add backend/db/ DATABASE_SCHEMA.sql
git commit -m "Update database schema"
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

# Run tests
make backend-test
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

### Unit Tests
```bash
# Run all tests
make test

# Backend tests only
make backend-test

# Frontend tests only
make ui-test
```

### Integration Testing
```bash
# Start services for testing
make dev

# Run API tests manually
./test-api-endpoints.sh

# Check database state
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "SELECT COUNT(*) FROM users;"
```

### Manual Testing Checklist
- [ ] User registration works with proper UUID format
- [ ] User login validates credentials correctly
- [ ] Artist preferences are stored and retrieved
- [ ] Music matching finds users with shared artists
- [ ] Database persistence verified
- [ ] Frontend connects to backend API
- [ ] All endpoints return expected HTTP status codes

## 🔄 Development Cycle

### Typical Development Flow
1. **Start Environment**: `make dev`
2. **Make Changes**: Edit code in backend/business/ or ui/src/
3. **Test Changes**: Use curl or frontend to test
4. **Check Database**: Verify data persistence
5. **Run Tests**: `make test`
6. **Commit**: Git commit with descriptive message

### Schema Changes Flow
1. **Edit Schema**: Modify files in backend/db/schema/
2. **Sync Schema**: `make schema-sync`
3. **Generate Code**: `make backend-sqlc`
4. **Update Queries**: Edit backend/db/queries/queries.sql if needed
5. **Regenerate**: `make backend-sqlc`
6. **Test**: Verify database operations work
7. **Commit**: Include schema files and generated code

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
```

#### Frontend (Docker Compose)
```bash
API_BASE_URL=http://localhost:8080
```

## 📚 Documentation

### Key Files
- **README.md**: Main project documentation
- **DATABASE.md**: Database setup and schema information
- **DOCKER-COMPOSE-SETUP.md**: Docker environment setup
- **backend/README.md**: Backend-specific documentation
- **ui/README.md**: Frontend-specific documentation

### Generated Documentation
- **Backend API**: http://localhost:8080/docs (when running)
- **Database Schema**: DATABASE_SCHEMA.sql (auto-generated)
- **SQLC Generated Code**: backend/db/sqlc/ (auto-generated)

### Development Scripts
- **Makefile**: Main orchestration and commands
- **backend/Makefile**: Backend-specific commands
- **dev.sh**: Legacy development script (use `make dev` instead)

This guide should be updated as the development workflow evolves. Always refer to the Makefile for the most current commands and targets.