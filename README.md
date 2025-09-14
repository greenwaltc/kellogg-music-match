# 🎵 Kellogg Music Match

A professional full-stack music taste matching application with Go backend, Angular frontend, and automated infrastructure deployment.

## 🏗️ Architecture Overview

```
kellogg-music-match/
├── backend/          # Go backend with OpenAPI generation
├── ui/              # Angular frontend application
├── pulumi/          # Infrastructure as Code (Pulumi)
├── Makefile         # Top-level orchestration
└── docker-compose.yml  # Local development environment
```

### 🔧 Backend
- **Go 1.22+** with OpenAPI-generated server
- **Clean Architecture** - Generated code separated from business logic
- **REST API** with authentication, user management, and music matching
- **Docker** containerization with multi-stage builds

### 🎨 Frontend  
- **Angular 17+** with reactive forms and modern UI
- **Real-time validation** for password complexity and user input
- **Responsive design** optimized for music discovery
- **Docker** containerization with Nginx

### ☁️ Infrastructure
- **Pulumi** Infrastructure as Code
- **Cloud deployment** ready (AWS/Azure/GCP)
- **Automated provisioning** and configuration management

## 🚀 Quick Start

### Prerequisites
- **Go 1.22+**
- **Node.js 18+** 
- **Docker & Docker Compose**
- **Make**

### 1. Initial Setup
```bash
# Clone and setup the project
git clone <repository-url>
cd kellogg-music-match
make setup
```

### 2. Development Environment
```bash
# Option 1: Full Docker environment (recommended)
make docker-run

# Option 2: Local development with live reload
make dev

# Option 3: Component-specific development
make backend-dev  # Backend only
make ui-dev       # Frontend only
```

### 3. Access the Application
- **Frontend:** http://localhost:4200
- **Backend API:** http://localhost:8080  
- **Health Check:** http://localhost:8080/health

## 🛠️ Development Commands

### 📋 General Operations
```bash
make help           # Show all available commands
make info           # Project information
make status         # Application status
make health         # Health check
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
make docker-build       # Build all Docker images
make docker-run         # Start full application
make docker-stop        # Stop all services
make docker-logs        # View application logs
make docker-restart     # Restart services
```

### 🔧 Backend Development
```bash
make backend-help           # Backend-specific commands
make backend-generate       # Generate OpenAPI code
make backend-build          # Build backend
make backend-test           # Run backend tests
make backend-dev            # Development with live reload
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

