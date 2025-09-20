# 🎵 Kellogg Music Match

A professional full-stack music taste matching application designed for Kellogg students, featuring Go backend, Angular frontend, PostgreSQL database with scientific extensions, and automated infrastructure deployment.

## 🏗️ Architecture Overview

```
kellogg-music-match/
├── backend/              # Go backend with OpenAPI generation
├── ui/                  # Angular frontend application  
├── pulumi/              # Infrastructure as Code (Pulumi)
├── DATABASE_SCHEMA.sql  # PostgreSQL database schema
├── init-database.sh     # Database initialization script
├── dev.sh              # Development environment helper
├── Makefile            # Top-level orchestration
└── docker-compose.yml  # Local development environment
```

### 🔧 Backend
- **Go 1.22+** with OpenAPI-generated server
- **Clean Architecture** - Generated code separated from business logic
- **Custom PostgreSQL Integration** - Complete database migration with scientific extensions
- **UserRepository Interface** - Clean abstraction layer for database operations
- **Scientific Similarity Engine** - Hybrid Jaccard + positional correlation algorithm for music matching
- **REST API** with authentication, user management, and music matching
- **Comprehensive Testing** - Unit tests + Ginkgo behavioral tests for algorithm validation
- **Docker** containerization with multi-stage builds

### 🗄️ Database
- **Custom PostgreSQL 15** with **plpython3u extension** and scientific libraries (scipy, numpy)
- **Scientific Similarity Calculations** - Custom `spearman_distance` function with hybrid Jaccard + positional correlation algorithm
- **SQLC Integration** - Type-safe Go code generated from SQL queries
- **Consolidated Schema Management** - Single migration file in `backend/db/schema/001_initial.sql`
- **Automatic Schema Synchronization** - `DATABASE_SCHEMA.sql` auto-generated from schema files
- **Enhanced Database Pipeline** - Reset, verification, and sync guarantees in development workflow
- **UserRepository Interface** - Clean abstraction layer for database operations
- **UUID Support** - Proper UUID format with performance indexes
- **User Management** - Complete profile support including program and graduation year
- **Music Matching** - Artist relationships with scientifically accurate similarity scoring
- **Performance Optimized** - Comprehensive indexes and foreign key constraints

### 🎨 Frontend  
- **Angular 17+** with reactive forms and modern UI
- **Real-time validation** for password complexity and user input
- **Responsive design** optimized for music discovery
- **Docker** containerization with Nginx

### ☁️ Infrastructure
- **Pulumi** Infrastructure as Code
- **Kubernetes deployment** with StatefulSet for PostgreSQL
- **Cloud deployment** ready (AWS/Azure/GCP)
- **Automated provisioning** and configuration management

## 🚀 Quick Start

### Prerequisites
- **Go 1.22+**
- **Node.js 18+** 
- **Docker & Docker Compose**
- **PostgreSQL client tools** (optional, for direct database access)
- **Make**

### 1. Initial Setup
```bash
# Clone and setup the project
git clone <repository-url>
cd kellogg-music-match
make setup
```

### 2. Development Environment

#### Option A: Full Docker Environment (Recommended)
```bash
# Start everything (database, backend, frontend)
make dev

# Or use docker-compose directly
docker-compose up -d
```

#### Option B: Individual Services
```bash
# Start only PostgreSQL database
docker-compose up -d postgres

# In separate terminals:
make backend-dev  # Backend with live reload
make ui-dev       # Frontend with live reload
```

### 3. Access the Application
- **Frontend:** http://localhost:4200
- **Backend API:** http://localhost:8080  
- **Database:** localhost:5432 (user: `kellogg_user`, db: `kellogg_music_match`)
- **Health Check:** http://localhost:8080/health

### 4. Test the Setup
```bash
# Health check
curl http://localhost:8080/health

# Test user registration with full profile
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@kellogg.northwestern.edu","password":"TestPassword123!","firstName":"Test","lastName":"User","program":"2Y","graduationYear":2026}'

# Test user login  
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"TestPassword123!"}'

# Test music matching
curl -X POST http://localhost:8080/findMusicMatches \
  -H "Content-Type: application/json" \
  -H "X-User-Username: testuser" \
  -d '{"artists":["The Beatles","Taylor Swift"]}'
```

## 🛠️ Development Commands

### 📋 General Operations
```bash
make help           # Show all available commands
make info           # Project information
make status         # Application status
make health         # Health check
```

### 🗄️ Database Operations
```bash
# Start all services (including database)
make dev
# or
docker-compose up -d

# Start PostgreSQL database only
docker-compose up -d postgres

# Database management (new enhanced pipeline)
make db-reset              # Complete database reset with fresh schema
make db-schema-verify      # Verify database structure matches expected
make db-force-schema-sync  # Nuclear option: complete reset with schema sync
make sync-schema           # Sync DATABASE_SCHEMA.sql from backend files

# Schema management
make create-migration name=add_feature  # Create new migration file
make check-schema-sync     # Verify schema files are synchronized

# Direct database access
docker exec -it kmm-postgres psql -U kellogg_user -d kellogg_music_match

# View logs
docker-compose logs backend
docker-compose logs postgres

# Database access
docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match

# Stop and cleanup
docker-compose down
docker-compose down -v  # Remove volumes too
```

### 🏗️ Build & Test
```bash
make build          # Build both backend and UI
make test           # Run all tests
make check          # Run all checks (lint, test, format)
make clean          # Clean all build artifacts
make schema-sync    # Synchronize database schema files
```

### 🐳 Docker Operations
```bash
make dev                # Start all services (recommended)
docker-compose up -d    # Start all services
docker-compose ps       # Show service status
docker-compose logs     # View application logs
docker-compose down     # Stop all services
docker-compose build    # Rebuild images
```

### 🔧 Backend Development
```bash
make backend-help           # Backend-specific commands
make backend-generate       # Generate SQLC and OpenAPI code
make backend-build          # Build backend
make backend-test           # Run all backend tests (Go + Ginkgo)
make backend-test-quick     # Run quick Go unit tests only
make backend-test-ginkgo    # Run Ginkgo behavioral tests
make backend-dev            # Development with live reload
make backend-sqlc           # Generate SQLC code from queries
```

### 🎨 Frontend Development  
```bash
make ui-build          # Build UI for production
make ui-dev            # Start development server
make ui-test           # Run UI tests
make ui-lint           # Lint UI code
```

### ☁️ Infrastructure Management
```bash
make infra-preview     # Preview infrastructure changes
make infra-deploy      # Deploy infrastructure
make infra-destroy     # Destroy infrastructure
make infra-output      # Show infrastructure outputs
```

## 🗄️ Database Information

### Database Schema
The application uses PostgreSQL with a comprehensive schema including:

- **Users Table**: Authentication and profile management with bcrypt password hashing and UUID primary keys
- **Artists Table**: Normalized artist storage with automatic name normalization and conflict resolution
- **User-Artists Table**: Many-to-many relationships for music preferences with cascade deletes
- **SQLC Integration**: Type-safe Go code generated from SQL queries in backend/db/queries/
- **Schema Management**: Multi-file schema system with automatic synchronization
- **Indexes**: Performance optimization for common operations including UUID lookups
- **Constraints**: Data integrity with foreign keys, check constraints, and unique indexes

### Database Architecture
- **UserRepository Interface**: Clean abstraction layer separating business logic from database operations
- **Custom PostgreSQL Implementation**: Full CRUD operations with scientific extensions (plpython3u, scipy, numpy)
- **Spearman Distance Function**: PostgreSQL function implementing hybrid similarity algorithm:
  - **Jaccard Similarity** (70% weight): Measures artist overlap between users
  - **Positional Correlation** (30% weight): Considers ranking/order of shared artists  
  - **Distance Values**: 0 (identical), 0.7 (subset), 2.0 (no overlap)
- **Type Safety**: SQLC generates type-safe Go structs and methods from SQL queries
- **Environment Variables**: Configurable connection parameters (DB_HOST, DB_PORT, DB_NAME, etc.)

### Sample Data
The development database includes:
- **10 Popular Artists**: Taylor Swift, The Beatles, Radiohead, Beyoncé, etc.
- **3 Test Users**: alice, bob, charlie (password: `password123`)
- **Sample Relationships**: Pre-configured music preferences for testing similarity algorithms
- **Spearman Distance Function**: PostgreSQL function for scientific similarity calculations

### Database Connection
```bash
# Docker Compose (local development)
Host: localhost
Port: 5432
Database: kellogg_music_match
Username: kellogg_user
Password: kellogg_secure_pass_2024

# Kubernetes (deployment)
Host: postgres.kellogg-music-match.svc.cluster.local
Port: 5432
Database: kellogg_music_match
Username: kellogg_user
Password: [from kubernetes secret]
```

### Database Files
- **`DATABASE_SCHEMA.sql`**: Complete PostgreSQL schema with tables, indexes, functions, and sample data
- **`backend/db/schema/*.sql`**: Multi-file schema source (single source of truth)
  - `001_initial.sql`: Base tables and relationships
  - `002_spearman_func.sql`: Initial similarity function
  - `003_add_rank.sql`: Ranking support
  - `004_spearman_distance.sql`: Hybrid Jaccard + positional similarity algorithm
- **`backend/db/queries/queries.sql`**: SQLC query definitions for type-safe Go code generation
- **`backend/sqlc.yaml`**: SQLC configuration for code generation
- **`init-database.sh`**: Initialization script for automatic setup
- **`DATABASE_SCHEMA.md`**: Comprehensive documentation with examples and queries
- **`postgres.dockerfile`**: Custom PostgreSQL image with scientific libraries

## 🔄 Development Workflows

### Quick Development Cycle
```bash
# 1. Make changes to code
# 2. Test locally
make dev

# 3. Run checks
make check

# 4. Test with Docker
make docker-run
```

### Backend API Development
```bash
# 1. Update OpenAPI specification
vim backend/openapi.yaml

# 2. Regenerate server code
make backend-generate

# 3. Implement business logic
vim backend/business/*.go

# 4. Test changes
make backend-test
make backend-run
```

### Frontend Development
```bash
# 1. Start development server
make ui-dev

# 2. Make changes with live reload
# 3. Run tests
make ui-test

# 4. Build for production
make ui-build
```

### Infrastructure Changes
```bash
# 1. Preview changes
make infra-preview

# 2. Deploy to staging
make deploy-staging

# 3. Deploy to production
make deploy-prod
```

## 🧪 Testing Strategy

### Comprehensive Test Coverage
- **Go Unit Tests**: Traditional Go testing for business logic and API handlers
- **Ginkgo Behavioral Tests**: Comprehensive behavioral testing using Ginkgo v2 + Gomega
- **Database Integration Tests**: End-to-end testing with PostgreSQL scientific functions
- **Algorithm Validation**: Specific tests validating similarity calculation accuracy

### Test Categories

#### 1. **Music Matching Algorithm Tests**
```bash
make test-behavioral  # Run Ginkgo behavioral tests
```
- **Identical Preferences**: Validates maximum similarity scores (≥0.9) for identical artist lists
- **Subset Relationships**: Tests moderate similarity (≈0.5) for subset cases
- **No Overlap**: Confirms zero matches for completely different preferences
- **Partial Overlap**: Validates Jaccard similarity calculation for partial artist overlap

#### 2. **Database Function Validation**
- **Distance = 0**: High scores for identical arrays (perfect similarity)
- **Distance = 0.7**: Moderate scores for subset relationships 
- **Distance = 2.0**: No matches for completely different preferences
- **Scientific Accuracy**: Validates hybrid Jaccard + positional algorithm

#### 3. **Edge Case Testing**
- Empty artist lists and single artist preferences
- Case-insensitive and whitespace-tolerant matching
- User exclusion (users don't match themselves)
- Result ordering verification (descending by score, then overlap)

### Running Tests
```bash
# All tests (Go + Ginkgo behavioral)
make test

# Quick unit tests only  
make test-quick

# Behavioral tests only
make test-behavioral

# Backend-specific tests
make backend-test
make backend-test-ginkgo

# Full checks (lint + test + format)
make check
```

### Unit Tests
```bash
make test-unit         # Run all unit tests
make backend-test      # Backend unit tests
make ui-test          # Frontend unit tests
```

### Integration Tests
```bash
make test-integration  # Full integration test suite
```

### End-to-End Tests
```bash
make test-e2e         # Complete user workflow tests
```

## 🚢 Deployment

### Local Development
```bash
make docker-run       # Full local environment
```

### Staging Environment
```bash
make deploy-staging   # Deploy to staging with full checks
```

### Production Environment
```bash
make deploy-prod      # Production deployment workflow
```

## 📊 Monitoring & Maintenance

### Health Monitoring
```bash
make status           # Full application status
make health           # Backend health check
make logs             # View application logs
```

### Maintenance Tasks
```bash
make clean            # Clean all artifacts
make docker-clean     # Clean Docker resources
make infra-refresh    # Refresh infrastructure state
```

## 🔧 Configuration

### Environment Variables
- **Backend:** Configuration in `backend/go.mod` and Dockerfile
- **Frontend:** Build-time configuration in `ui/src/environments/`
- **Infrastructure:** Pulumi configuration in `pulumi/`

### Docker Configuration
- **Backend:** `backend/Dockerfile` with multi-stage build
- **Frontend:** `ui/Dockerfile` with Nginx serving
- **Compose:** `docker-compose.yml` for local development

## 📁 Project Structure Details

### Backend (`backend/`)
```
backend/
├── Makefile              # Backend-specific automation
├── openapi.yaml          # API specification
├── generated/            # OpenAPI generated code
├── business/             # Custom business logic
├── cmd/                  # Application entry point
└── README.md             # Backend documentation
```

### Frontend (`ui/`)
```
ui/
├── src/                  # Angular application source
├── docker/               # Docker configuration
├── package.json          # Node.js dependencies
└── Dockerfile           # Container configuration
```

### Infrastructure (`pulumi/`)
```
pulumi/
├── main.go              # Infrastructure definition
├── Pulumi.yaml          # Pulumi project configuration
└── README.md            # Infrastructure documentation
```

## 🤝 Contributing

1. **Setup development environment:**
   ```bash
   make setup
   ```

2. **Make changes following the architecture:**
   - Backend business logic in `backend/business/`
   - Frontend components in `ui/src/app/`
   - Infrastructure in `pulumi/`

3. **Test changes:**
   ```bash
   make check
   ```

4. **Submit for review:**
   ```bash
   make ci  # Run full CI workflow
   ```

## 📋 Available Make Targets

Run `make help` for a complete list of available commands organized by category:

- **🏗️ Build & Development:** `build`, `dev`, `test`, `check`, `clean`
- **🐳 Docker Operations:** `docker-build`, `docker-run`, `docker-stop`
- **🔧 Backend:** `backend-*` (forwarded to backend Makefile)
- **🎨 Frontend:** `ui-build`, `ui-dev`, `ui-test`, `ui-lint`
- **☁️ Infrastructure:** `infra-deploy`, `infra-preview`, `infra-destroy`
- **📊 Monitoring:** `status`, `health`, `logs`
- **🚀 Deployment:** `deploy-local`, `deploy-staging`, `deploy-prod`

## 🆘 Troubleshooting

### Common Issues

**Port conflicts:**
```bash
make docker-stop    # Stop all services
make status         # Check what's running
```

**Docker issues:**
```bash
make docker-clean   # Clean Docker resources
make docker-build   # Rebuild images
```

**Build failures:**
```bash
make clean          # Clean all artifacts
make deps           # Update dependencies
make build          # Rebuild
```

### Getting Help
- `make help` - All available commands
- `make info` - Project information  
- `make status` - Current application status
- Component-specific help: `make backend-help`, etc.

---

## 🔍 API Documentation

The backend uses OpenAPI 3.0 specification located in `backend/openapi.yaml`. The API includes:

### Authentication Endpoints
- `POST /register` - User registration with comprehensive validation
- `POST /login` - User authentication with bcrypt password verification

### Matching Endpoints  
- `POST /findMusicMatches` - Find users with similar music taste

### Health Endpoints
- `GET /health` - Service health check

### Authentication & Security

- **User Registration**: Secure password-based registration with comprehensive validation
- **Password Requirements**: 
  - Minimum 8 characters
  - At least one uppercase letter
  - At least one lowercase letter
  - At least one number
  - At least one special character (!@#$%^&*(),.?":{}|<>_)
- **Password Hashing**: Uses bcrypt for secure password storage
- **Form Validation**: Real-time password complexity feedback and confirmation matching

For complete API documentation, see `backend/openapi.yaml` or run the development server and visit the API explorer.

## 🎨 Frontend Features

- **User Registration**: Comprehensive form with real-time validation
  - Username, email, first name, last name fields
  - Password complexity requirements with visual feedback
  - Password confirmation with mismatch detection
  - Show/hide password visibility toggles
- **User Login**: Secure authentication with username/password
- **Artist Management**: Add/remove favorite artists (1-10 artists)
- **Music Matching**: Find users with similar music taste
- **Responsive Design**: Works on desktop and mobile devices
- **Real-time Validation**: Immediate feedback on form inputs

### Password Requirements

The application enforces strong password security with real-time visual indicators showing which requirements are met as you type.

## 🏗️ Architecture

```
┌─────────────────┐    HTTP/REST    ┌─────────────────┐
│   Angular UI    │◄───────────────►│   Go Backend    │
│   (Port 4200)   │                 │   (Port 8080)   │
│                 │                 │                 │
│ • Registration  │                 │ • OpenAPI Gen   │
│ • Login         │                 │ • Business Logic│
│ • Artist Mgmt   │                 │ • Clean Arch    │
│ • Match Results │                 │ • In-Memory DB  │
└─────────────────┘                 └─────────────────┘
```

## 🚀 Future Improvements

- **Database Integration**: Replace in-memory storage with PostgreSQL/MongoDB
- **JWT Authentication**: Add token-based auth for stateless API access  
- **Rate Limiting**: Implement API rate limiting and abuse protection
- **Email Verification**: Add email verification for new registrations
- **Social Login**: Support OAuth with Google/GitHub/Spotify
- **Advanced Matching**: Weighted algorithms considering artist popularity
- **Real-time Features**: WebSocket for live match notifications
- **Testing**: Comprehensive unit and integration test suites
- **Monitoring**: Add logging, metrics, and health monitoring
- **Security Enhancements**: CSRF protection, input sanitization, security headers

