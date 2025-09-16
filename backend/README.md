# Kellogg Music Match Backend

A Go backend server generated from OpenAPI specifications with PostgreSQL database integration and clean architecture separation.

## 🏗️ Architecture

- **`generated/`** - OpenAPI generated code (controllers, models, routing)
- **`business/`** - Custom business logic (authentication, matching, storage)  
- **`cmd/`** - Application entry point and service wrappers
- **`openapi.yaml`** - API specification
- **`Makefile`** - Build automation and development tasks

## 🗄️ Database Integration

The backend integrates with PostgreSQL database for:
- **User Management**: Registration, authentication with bcrypt password hashing
- **Music Preferences**: Artist storage and user-artist relationships
- **Matching Algorithm**: Music taste similarity calculations using Jaccard similarity

### Database Configuration
The backend uses these environment variables for database connection:
```bash
KELLOGG_MUSIC_MATCH_DATABASE_HOST=localhost
KELLOGG_MUSIC_MATCH_DATABASE_PORT=5432
KELLOGG_MUSIC_MATCH_DATABASE_NAME=kellogg_music_match
KELLOGG_MUSIC_MATCH_DATABASE_USER=postgres
KELLOGG_MUSIC_MATCH_DATABASE_PASSWORD=postgres123
KELLOGG_MUSIC_MATCH_DATABASE_SSL_MODE=disable
```

> **Note**: These match the Docker Compose configuration for seamless local development.

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- Docker (for OpenAPI generation and containerization)
- Make

### Development Workflow

1. **Start the database:**
   ```bash
   # From project root
   ./dev.sh db-only
   ```

2. **View available commands:**
   ```bash
   make help
   ```

3. **Generate OpenAPI code from specification:**
   ```bash
   make generate
   ```

4. **Build and run locally (with database):**
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
# Validate OpenAPI spec
make openapi-validate

# Generate server code
make generate

# Generate documentation
make openapi-docs
```

### Development
```bash
# Clean build artifacts
make clean

# Build binary
make build

# Run tests
make test

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