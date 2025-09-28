# Kellogg Music Match Backend

A Go backend server with consolidated PostgreSQL database schema, SQLC type-safe queries, OpenAPI specification, scientific similarity calculations, and comprehensive behavioral testing for Kellogg student music matching.

## 🏗️ Architecture

- **`generated/`** - OpenAPI generated code (controllers, models, routing)
- **`business/`** - Custom business logic (authentication, matching, database repository)  
  - **`matching.go`** - Music matching engine with hybrid similarity calculations
  - **`database.go`** - UserRepository implementation with enhanced PostgreSQL integration
  - **`business_suite_test.go`** - Ginkgo test suite bootstrap
  - **`matching_behavior_test.go`** - Comprehensive behavioral tests for similarity algorithms
  - **`TESTING.md`** - Behavioral testing documentation
- **`cmd/`** - Application entry point and service wrappers
- **`db/`** - Enhanced database layer with consolidated schema and SQLC integration
  - **`schema/001_initial.sql`** - Consolidated PostgreSQL schema with Kellogg-specific fields
  - **`queries/queries.sql`** - SQLC query definitions with type safety optimizations
  - **`sqlc/`** - Generated type-safe Go code from consolidated schema
- **`openapi.yaml`** - API specification
- **`sqlc.yaml`** - SQLC configuration
- **`Makefile`** - Build automation and development tasks with enhanced database management

## 🗄️ Enhanced Database Integration

The backend uses a consolidated PostgreSQL setup with scientific extensions and Kellogg-specific enhancements:

### Consolidated Schema Features
- **Single Initial Schema**: `db/schema/001_initial.sql` replaces 9 migration files
- **Kellogg Student Profiles**: Complete user profiles with `program` and `graduation_year`
- **Program Validation**: Constraints for Kellogg programs (2Y, 1Y, MMM, MBAi, JD-MBA, MD-MBA, EWMBA, JV)
- **Graduation Year Constraints**: Validation for years 2025-2030
- **Enhanced SQLC Integration**: Optimized queries for Go code generation

### Scientific Database Features
- **Custom PostgreSQL 15 Image**: Built with plpython3u extension and scientific libraries (scipy, numpy)
- **Spearman Distance Function**: PostgreSQL function implementing hybrid similarity algorithm:
  - **Jaccard Similarity** (70% weight): Measures artist overlap between users
  - **Positional Correlation** (30% weight): Considers ranking/order of shared artists
  - **Distance Values**: 0 (identical), 0.7 (subset), 2.0 (no overlap)
- **Text Array Support**: Custom handling for PostgreSQL TEXT[] arrays in Go using pq.StringArray

### Repository Pattern
- **UserRepository Interface**: Clean abstraction for database operations
- **Custom PostgreSQL Implementation**: Full CRUD operations with scientific extensions
- **SQLC Integration**: Type-safe Go code generated from SQL queries
- **Custom Array Scanning**: Overrides for proper PostgreSQL array handling
- **UUID Support**: Proper UUID handling with database constraints

### Database Features
- **User Management**: Registration, authentication with bcrypt password hashing
- **Music Preferences**: Normalized artist storage and user-artist relationships  
- **Scientific Matching Algorithm**: Music taste similarity calculations using hybrid Jaccard + positional correlation
- **Custom Distance Function**: PostgreSQL plpython3u function for accurate similarity scoring
- **Transaction Support**: Proper error handling and data consistency
- **Performance Optimization**: Indexes and optimized queries for common operations

### Database Configuration
The backend uses these environment variables for database connection:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=kellogg_music_match
DB_USER=kellogg_user
DB_PASSWORD=kellogg_secure_pass_2024
DB_SSLMODE=disable
```

> **Note**: These match the Docker Compose configuration for seamless local development.

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- Docker (for OpenAPI generation and containerization)
- Make

### Development Workflow

1. **Start the database:**
   ```bash
   # From project root
   docker-compose up -d postgres
   ```

2. **Generate SQLC code:**
   ```bash
   make sqlc-generate
   ```

3. **Generate OpenAPI code:**
   ```bash
   make generate
   ```

4. **Build and run locally:**
   ```bash
   make run
   ```

5. **Development with live reload:**
   ```bash
   make install-tools  # Install 'air' tool
   make dev           # Runs with database environment variables
   ```

6. **Run with Docker:**
   ```bash
   make docker-run
   ```

## 🔧 Common Tasks

### Code Generation
```bash
# Generate SQLC code from queries
make sqlc-generate

# Generate OpenAPI server code  
make generate

# View OpenAPI documentation
make openapi-docs
```

### Database Operations
```bash
# Enhanced database management (from project root)
make db-reset              # Complete database reset with fresh schema
make db-schema-verify      # Verify database structure matches expected
make sync-schema          # Sync DATABASE_SCHEMA.sql from backend/db/schema/*.sql

# Traditional database operations
docker-compose up -d postgres                    # Start database
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match  # Connect
docker-compose logs postgres                     # View logs

# Schema development workflow
make sqlc-generate        # Generate Go code from queries after schema changes
```

### Development
```bash
# Clean build artifacts
make clean

# Build binary
make build

# Run tests
make test

# Run quick unit tests only
make test-quick  

# Run Ginkgo behavioral tests
make test-ginkgo

# Run behavioral tests (alias)
make test-behavioral

# Format code
make format

# Run linting
make lint

# Full check (clean + lint + test)
make check
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Start full application
make docker-run

# Stop application
make docker-stop

# View logs
make docker-logs
```

## 🧪 Testing Framework

The backend includes comprehensive testing using both Go's built-in testing and Ginkgo behavioral testing framework.

### Test Categories

#### 1. **Go Unit Tests**
Traditional Go tests for individual functions and business logic:
```bash
make test-quick  # Fast unit tests only
```

#### 2. **Ginkgo Behavioral Tests**
Comprehensive behavioral testing using Ginkgo v2 + Gomega for algorithm validation:
```bash
make test-ginkgo  # Run behavioral tests
```

**Test Coverage:**
- **Music Matching Algorithm**: Validates Jaccard similarity calculations
- **Edge Cases**: Empty lists, normalization, caller exclusion
- **Database Function Alignment**: Tests scenarios matching PostgreSQL distance function
- **Scientific Accuracy**: Validates similarity scores for known test cases

#### 3. **Algorithm Validation Tests**
Specific tests that validate the hybrid similarity algorithm behavior:
- **Identical Preferences**: Score ≥ 0.9 for identical artist lists
- **Subset Relationships**: Score ≈ 0.5 for subset cases like {Tool} vs {Tool, Radiohead}
- **No Overlap**: No matches returned for completely different preferences
- **Partial Overlap**: Correct Jaccard calculations (e.g., 1/3 ≈ 0.33)

### Testing Setup
```bash
# Install testing dependencies
make install-tools

# Run all tests (Go + Ginkgo)
make test

# Run full checks (lint + test + format)
make check
```

### Test Files
- **`business_suite_test.go`**: Ginkgo test suite bootstrap
- **`matching_behavior_test.go`**: Comprehensive behavioral tests (13 test specs)
- **`TESTING.md`**: Detailed testing documentation and expected results

### Quick Workflows
```bash
# Quick development cycle
make quick  # generate + build + run

# Full CI workflow
make ci     # deps + validate + generate + format + lint + test + build
```

## 📁 Project Structure

```
backend/
├── openapi.yaml          # API specification
├── Makefile              # Build automation
├── .air.toml             # Live reload configuration
├── go.mod & go.sum       # Go dependencies
├── Dockerfile            # Container configuration
│
├── generated/            # 🔧 OpenAPI Generated (DO NOT EDIT)
│   ├── api*.go           # HTTP controllers
│   ├── model_*.go        # Request/response models
│   └── *.go              # Routing, helpers, etc.
│
├── business/             # 💼 Business Logic (CUSTOM)
│   ├── store.go          # Data storage
│   ├── auth_service.go   # Authentication logic
│   ├── health_service.go # Health checks
│   ├── matching_service.go # Music matching logic
│   └── matching.go       # Matching algorithms
│
└── cmd/                  # 🚀 Application Entry (CUSTOM)
    ├── main.go           # Startup & dependency injection
    └── wrappers.go       # OpenAPI service wrappers
```

## 🔄 Regenerating Code

When you update `openapi.yaml`:

1. **Validate changes:**
   ```bash
   make openapi-validate
   ```

2. **Regenerate code:**
   ```bash
   make generate
   ```

3. **Test changes:**
   ```bash
   make test
   ```

The `generate` target safely regenerates all OpenAPI code while preserving your custom business logic.

## 🌐 API Endpoints

- **Health Check:** `GET /health`
- **User Registration:** `POST /register`
- **User Login:** `POST /login`
- **Find Music Matches:** `POST /findMusicMatches`

Full API documentation is available in `openapi.yaml` or generate HTML docs with:
```bash
make openapi-docs
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
go test ./... -cover

# Test specific package
go test ./business/... -v
```

## 🚢 Deployment

### Local Development
```bash
make docker-run
```

### Production
The Dockerfile creates an optimized production build:
```bash
docker build -t kellogg-music-match-backend .
docker run -p 8080:8080 kellogg-music-match-backend
```

## 🛠️ Development Tools

Install recommended tools:
```bash
make install-tools
```

This installs:
- `air` - Live reload for development
- `golangci-lint` - Comprehensive linting (manual install)

## 📋 Code Standards

- Generated code in `generated/` should **never** be manually edited
- Business logic goes in `business/` package
- All custom code should have tests
- Run `make format` before committing
- Validate with `make check` before pushing

## 🤝 Contributing

1. Make changes to business logic or OpenAPI spec
2. Run `make ci` to ensure everything works
3. Commit and push changes

The build system ensures generated and custom code stay properly separated!