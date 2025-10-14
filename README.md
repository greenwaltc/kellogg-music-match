# 🎵 Kellogg Music Match

A professional full-stack music taste matching application for Kellogg students. It features a Go backend, Angular frontend, PostgreSQL database, real-time concert discovery, and a **rank-weighted overlap similarity engine** with Spotify time‑range awareness. (The original scientific PWO distance PostgreSQL function is still available for historical/analytical comparison but is no longer used for live match scoring.)

├── pulumi/              # Infrastructure as Code (Pulumi)
├── database/            # Flyway migrations and configuration
├── scripts/             # Deployment and utility scripts
├── Makefile            # Top-level orchestration
└── docker-compose.yml  # Local development environment
```

### 🔧 Backend
- **Go 1.23+** with OpenAPI-generated server
- **Clean Architecture** - Generated code separated from business logic
- **Ticketmaster Integration** - Concert discovery API with dependency inversion
- **PostgreSQL Integration** - Flyway migrations (legacy PWO distance function retained for analysis)
- **Rank-Weighted Overlap Similarity Engine** (current production scorer with per-overlap theoretical max normalization)
- **Spotify Time Range Support** (`short_term`, `medium_term`, `long_term`) + configurable defaults
- **Configurable Limits** (`limit`, `overlapsLimit`, server-side caps to prevent abuse)
- **In-Memory Similarity Cache** (30s TTL; auto-invalidated on new Spotify artist snapshot)
- **UserRepository Interface** - Clean abstraction layer for database operations
- **REST API** with auth, user management, music matching, concert discovery
- **Comprehensive Testing** - Unit + Ginkgo behavioral tests (normalization, overlap truncation, time range, JWT, password reset, etc.)
- **Docker** multi-stage builds

### 🗄️ Database
- **PostgreSQL 16** with legacy `pwo_distance` (kept for benchmarking) and supporting artist/user schema
- **Rank/Overlap Source Data** – Spotify top artists persisted per time range to drive Go-level similarity
- **SQLC Integration** - Type-safe Go code generated from SQL queries
- **Flyway Migrations** - Professional database versioning (V019 current)
- **MusicBrainz Integration** - 47,452 artist records loaded and deduplicated
- **UserRepository Interface** - Clean abstraction layer for database operations
- **UUID Support** - Proper UUID format with performance indexes
- **User Management** - Complete profile including program and graduation year
- **Performance Optimized** - Indexes and FK constraints for matching + events queries

### 🎨 Frontend  
- **Angular 17+** with reactive forms and modern UI
- **Chicago Events Page** - Standalone component with infinite scroll and artist search
- **Real-time validation** for password complexity and user input
- **State Management** - Robust user session handling with automatic match clearing on user change
- **Responsive design** optimized for music discovery and concert browsing
- **Consistent Theming** - Light/dark mode support across all components
- **Docker** containerization with Nginx
- **Concert Integration** - Complete UI for browsing 6 months of Chicago area events
 - **Matches Refresh** - Manual (desktop button) & pull-to-refresh (mobile) with client+server rate limiting

### 🎵 Concert Integration
- **Ticketmaster API** - Live concert and event discovery with 6-month configurable date range
- **Chicago Events Page** - Dedicated UI component with infinite scroll and search functionality
- **Real-time Search** - Case-insensitive artist filtering with debounced input
- **Pagination Support** - Efficient data loading with configurable page sizes
- **Automated Sync** - 24-hour scheduled synchronization of concert data
- **Dependency Inversion** - Clean architecture with EventProvider interface
- **Configuration Management** - Environment-based API credentials and date ranges
- **Geographic Targeting** - Configurable location-based event search (default: Chicago, IL)
- **Comprehensive Testing** - MockEventProvider for testing without API calls
- **API Abstraction** - Clean separation between business logic and external APIs

### ☁️ Infrastructure
- **Pulumi** Infrastructure as Code
- **Kubernetes deployment** with StatefulSet for PostgreSQL
- **Docker Compose** - Complete local development environment
- **Cloud deployment** ready (AWS/Azure/GCP)
- **K3s Support** - Local Kubernetes development with image import scripts
- **Automated provisioning** and configuration management

## 🚀 Quick Start

### Prerequisites
- **Go 1.23+**
- **Node.js 18+** 
- **Docker & Docker Compose**
- **PostgreSQL client tools** (optional)
- **Make**
- **Ticketmaster API Key** (see [TICKETMASTER_INTEGRATION.md](TICKETMASTER_INTEGRATION.md))

### 1. Initial Setup
```bash
# Clone and setup the project
git clone https://github.com/greenwaltc/kellogg-music-match.git
cd kellogg-music-match

# Build and start full environment
make dev
```

### 2. Configure Ticketmaster API (Optional)
```bash
export TICKETMASTER_CONSUMER_KEY="your_key_here"
export TICKETMASTER_CONSUMER_SECRET="your_secret_here"
```

### 3. Access the Application
- **Frontend**: http://localhost:4200
- **Backend API**: http://localhost:8080 
- **Health Check**: http://localhost:8080/health
- **Database**: localhost:5432 (user: kellogg_user)

### 4. Development Commands
```bash
make test        # Run all tests
make status      # Application health information
make docker-stop # Stop containers
make clean       # Remove build artifacts
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

# Test Chicago Events API (6 months of events)
curl "http://localhost:8080/chicago/events?limit=5&offset=0"

# Search Chicago events by artist
curl "http://localhost:8080/chicago/events?limit=10&artist=john&offset=0"
```

## � Refreshing Matches

You can manually refresh your music matches to incorporate newly synced Spotify data or changed listening windows.

Desktop:
- Use the Refresh button in the Matches page controls bar (next to range/basis controls). Disabled briefly after use.

Mobile:
- Pull down from the very top of the Matches page until the "Release to refresh" indicator appears, then release.

Rate Limiting:
- Backend: Maximum 3 refresh requests per 10 seconds per user (returns HTTP 429 when exceeded).
- Frontend: Client blocks refresh attempts closer than ~4 seconds apart and caps bursts to 3 per 10 seconds, mirroring backend constraints.

If you exceed limits you'll see the button disabled or (for API calls) a 429 response; simply wait a few seconds and try again.

Rate Limit Headers (returned by /findMusicMatches):

| Header | Meaning |
| ------ | ------- |
| X-RateLimit-Limit | Maximum requests allowed in the current window (3) |
| X-RateLimit-Remaining | Requests left before reaching the limit |
| X-RateLimit-Window | Window size (e.g. 10s) |
| Retry-After | Seconds until you can retry (only present on 429) |

Example 429 response:
```
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 3
X-RateLimit-Remaining: 0
X-RateLimit-Window: 10s
Retry-After: 6
Content-Type: application/json

{"message":"too many match requests - retry shortly"}
```

UI Behavior:
- Warning toast on client-side throttle (fast manual/pull refresh attempts).
- Warning toast when server returns 429 (includes Retry-After guidance when available).


## �🛠️ Development Commands

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

# Database management (Flyway migration system)
make db-reset              # Complete database reset with Flyway migrations
make db-migrate            # Apply pending Flyway migrations
make db-clean              # Clean database schema
make create-migration name=add_feature  # Create new Flyway migration file

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
- **UserRepository Interface**: Clean abstraction layer
- **Flyway Migration System**: Incremental versioning (V001-V019)
- **Legacy PWO Function**: Still present for analytical/historical comparison; production similarity now in Go layer (rank-weighted overlap normalization)
- **Type Safety**: SQLC generated structs and query methods
- **Environment Variables**: Configurable connection parameters (DB_HOST, DB_PORT, DB_NAME, etc.)

### Sample Data
The development database includes:
- **Test Users**: Configurable through registration with full profile support
- **Music Preferences**: User-artist relationships persisted for Spotify-derived ranks
- **Legacy PWO Function**: Available if you want to benchmark against prior distance-based scoring

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
-- **`backend/db/schema/migrations/`**: Flyway migration files (V001 through V019)
  - `V001__initial.sql`: Core database structure with users, artists tables
  - `V010__pwo_metric.sql`: Legacy PWO distance function implementation (not used for current scoring)
  - `V011-V012__musicbrainz_artists.sql`: MusicBrainz integration (47,452 artists)
  - `V019__fix_musicbrainz_upsert_function.sql`: Latest migration
- **`backend/db/queries/queries.sql`**: SQLC query definitions for type-safe Go code generation
- **`backend/sqlc.yaml`**: SQLC configuration for code generation
- **`DATABASE_SCHEMA.md`**: Comprehensive documentation with examples and queries
- **`postgres.dockerfile`**: PostgreSQL image configuration

## � API Documentation

### Core Endpoints

#### Authentication & User Management
```bash
# User Registration
POST /register
Content-Type: application/json
{
  "username": "string",
  "email": "string", 
  "password": "string",
  "firstName": "string",
  "lastName": "string",
  "program": "string",      # e.g., "2Y", "1Y", "MMM"
  "graduationYear": 2026
}

# User Login
POST /login
Content-Type: application/json
{
  "username": "string",
  "password": "string"
}
```

#### Music Matching (Rank-Weighted Overlap)
Consumes stored Spotify top artists for the requesting user; request body is ignored (kept only for backward compatibility).

```bash
POST /findMusicMatches?range=medium_term&limit=10&overlapsLimit=5
X-User-Username: testuser
Content-Type: application/json
{}
```

Query Parameters:
- `range` (optional) one of `short_term|medium_term|long_term` (default configured server-side)
- `limit` (optional) maximum users to return (server-capped)
- `overlapsLimit` (optional) truncate overlapping artist list per match

Scoring Overview:
1. Raw weight per overlapping artist = `1 / (anchorRank + otherRank)`
2. Theoretical max for that overlap if both had the better rank = `1 / (2 * min(anchorRank, otherRank))`
3. Sum raw weights → rawSimilarity; sum theoretical maxima → maxPossible
4. Normalized score = `rawSimilarity / maxPossible` clamped to [0,1]

Response (example):
```json
[
  {
    "name": "Alice Johnson",
    "program": "2Y",
    "graduationYear": 2025,
    "overlap": 7,
    "score": 0.83,
    "artists": ["Phoebe Bridgers", "Taylor Swift", "Radiohead"],
    "overlaps": [
      { "name": "Phoebe Bridgers", "anchorRank": 1, "otherRank": 2 },
      { "name": "Taylor Swift",    "anchorRank": 2, "otherRank": 1 }
    ]
  }
]
```

See [docs/music_matching.md](docs/music_matching.md) for deeper discussion and comparison to the legacy PWO distance.

#### Artist Search
```bash
# Search Artists (MusicBrainz Database - 47,452 artists)
GET /artists/search?query={search_term}&limit={count}

# Returns: Artist names and IDs matching the search term
```

### Chicago Events API

#### Get Chicago Events (6-month range)
```bash
# Get Events with Pagination
GET /chicago/events?limit={count}&offset={start}

# Search Events by Artist (case-insensitive)
GET /chicago/events?limit={count}&offset={start}&artist={search_term}

Response format:
{
  "events": [
    {
      "id": "string",
      "name": "string", 
      "date": "2025-10-15T19:30:00Z",
      "venue": {
        "name": "string",
        "address": {
          "street": "string",
          "city": "Chicago", 
          "state": "IL",
          "country": "US"
        }
      },
      "artists": [
        {
          "name": "string",
          "genres": ["string"]
        }
      ],
      "priceRange": {
        "min": 0,
        "max": 0,
        "currency": "USD"
      },
      "ticketUrl": "string"
    }
  ],
  "hasMore": true,
  "totalCount": 953
}
```

#### Concert Search & Discovery
```bash
# Search Concerts by Artist
GET /concerts/search?artist={artist_name}

# Get Concert Details  
GET /concerts/{eventId}
```

#### Health & Status
```bash
# Application Health Check
GET /health

# Returns service status and database connectivity
```

### Configuration Parameters

#### Ticketmaster Integration
- `TICKETMASTER_CONSUMER_KEY` - API consumer key (required)
- `TICKETMASTER_CONSUMER_SECRET` - API consumer secret (required)
- `TICKETMASTER_DATE_RANGE_MONTHS` - Event date range (default: 6 months)
- `TICKETMASTER_DEFAULT_CITY` - Search city (default: "Chicago")
- `TICKETMASTER_DEFAULT_STATE` - Search state (default: "IL")
- `TICKETMASTER_MAX_RESULTS` - Max results per API call (default: 200)

#### Backend Configuration
- `SERVER_PORT` - Backend server port (default: 8080)
- `DB_HOST`, `DB_PORT`, `DB_NAME` - Database connection settings
- `CORS_ALLOWED_ORIGINS` - Frontend URL for CORS (default: http://localhost:4200)

### Current Data Status
- **Chicago Events**: 953+ events (September 2025 - March 2026)
- **MusicBrainz Artists**: 47,452 deduplicated artist records
- **Auto-sync**: Events refreshed every 24 hours
- **Search Performance**: Indexed for fast artist and date filtering

## �🔄 Development Workflows

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
- **Identical Preferences**: Validates maximum similarity scores (1.0) for identical artist lists
- **Position Sensitivity**: Tests position-weighted similarity for different orderings
- **No Overlap**: Confirms zero matches for completely different preferences
- **Weighted Overlap**: Validates PWO calculation for partial artist overlap

#### 2. **Database Function Validation**
- **Distance = 0.0**: Maximum scores for identical arrays (perfect similarity)
- **Distance = 1.0**: No matches for completely different preferences
- **Position Weighting**: Validates position-sensitive similarity scoring
- **Scientific Accuracy**: Validates PWO (Position-Weighted Overlap) algorithm

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
make dev              # Full Docker Compose environment
make status           # Check all services health
```

### Kubernetes with Pulumi (Recommended)
```bash
cd pulumi/

# Configure secrets
pulumi config set postgres:password <secure-password>
pulumi config set ticketmaster:consumerKey <api-key> --secret
pulumi config set ticketmaster:consumerSecret <api-secret> --secret

# Deploy infrastructure
pulumi up
```

### Local Kubernetes (K3s)
```bash
# Build and import images for local cluster
make build-all
scripts/k3s-image-import.sh

# Deploy to local K3s
cd pulumi/
pulumi up --stack local
```

### Docker Compose Production
```bash
make build-all        # Build production images
docker-compose up -d  # Deploy with production config
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

