# 📚 Documentation Index

## Quick Navigation

### 🚀 **Start Here**
- **[README.md](README.md)** - Complete application overview, setup, and API documentation
- **[Makefile](Makefile)** - All development commands and workflows

### 🔌 **Integration Guides**
- **[TICKETMASTER_INTEGRATION.md](TICKETMASTER_INTEGRATION.md)** - Chicago Events API, sync service, and Ticketmaster configuration

### 🗄️ **Database Documentation**
- **[DATABASE.md](DATABASE.md)** - PostgreSQL setup, schema management, and migration system
- **[DATABASE_SCHEMA.md](DATABASE_SCHEMA.md)** - Complete schema reference with examples
- **[FLYWAY-MIGRATION-SETUP.md](FLYWAY-MIGRATION-SETUP.md)** - Migration system details
- **[POSTGRESQL-PERSISTENCE.md](POSTGRESQL-PERSISTENCE.md)** - Data persistence and storage configuration

### 🐳 **Infrastructure & Deployment**
- **[docker-compose.yml](docker-compose.yml)** - Local development environment
- **[pulumi/](pulumi/)** - Infrastructure as Code for cloud deployment
- **[K3S_LOCAL_IMAGES_GUIDE.md](K3S_LOCAL_IMAGES_GUIDE.md)** - Local Kubernetes development
- **[KUBERNETES_MUSICBRAINZ_SETUP.md](KUBERNETES_MUSICBRAINZ_SETUP.md)** - K8s-specific setup

### 🎵 **MusicBrainz Integration**
- **[MUSICBRAINZ_INTEGRATION.md](MUSICBRAINZ_INTEGRATION.md)** - Artist database integration (47K+ artists)
- **[MUSICBRAINZ_DOCKER_SETUP.md](MUSICBRAINZ_DOCKER_SETUP.md)** - Docker setup for MusicBrainz data

## Application Features

### ✅ **Currently Implemented**
1. **Chicago Events API** - 6-month event discovery with search and pagination
2. **Music Matching** - PWO algorithm-based similarity scoring  
3. **User Management** - Registration, authentication, and profiles
4. **Artist Database** - 47,452 MusicBrainz artists with search
5. **Real-time UI** - Angular frontend with infinite scroll and search
6. **Automated Sync** - 24-hour event synchronization
7. **Database Management** - Flyway migrations with PostgreSQL 16

### 📈 **Current Data Status**
- **Chicago Events**: 953+ events (September 2025 - March 2026)
- **Artists**: 47,452 MusicBrainz records (deduplicated)
- **Database**: 19 migration files (V001-V019)
- **API Endpoints**: 9 REST endpoints with OpenAPI specification

## Development Workflow

### 🏃‍♂️ **Quick Start**
```bash
# 1. Start everything
make dev

# 2. Check status  
make status

# 3. View Chicago events
curl http://localhost:8080/chicago/events?limit=5
```

### 🧪 **Testing**
```bash
# Backend tests (Go + Ginkgo)
make backend-test

# UI tests 
make ui-test

# Integration tests
make test-integration
```

### 🔧 **Code Generation**
```bash
# Generate OpenAPI server code
make backend-generate

# Generate SQLC database code  
make backend-generate-sqlc
```

## File Organization

```
kellogg-music-match/
├── README.md                    # 📖 Main documentation
├── Makefile                     # 🛠️ Development commands
├── docker-compose.yml           # 🐳 Local environment
├── backend/                     # 🔧 Go backend
│   ├── openapi.yaml            # 📋 API specification
│   ├── business/               # 💼 Business logic
│   ├── generated/              # 🤖 Generated OpenAPI code
│   └── db/                     # 🗄️ SQLC queries & migrations
├── ui/                         # 🎨 Angular frontend
│   └── src/app/                # 🏠 Components & services
├── database/                   # 🗄️ Flyway configuration
├── scripts/                    # 📜 Utility scripts
└── pulumi/                     # ☁️ Infrastructure code
```

## Getting Help

### 📞 **Command Help**
```bash
make help              # All available commands
make backend-help      # Backend-specific commands  
make events-status     # Chicago Events status
make health           # Application health check
```

### 🔍 **Debugging**
```bash
# View logs
docker-compose logs backend
docker-compose logs postgres

# Database access
make db-connect

# Event data verification
make events-sample
```

---
*Last updated: September 2025 - Reflects Chicago Events implementation and 6-month Ticketmaster integration*