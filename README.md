# kellogg-music-match

Monorepo containing:

- `ui/` – Angular front-end (Kellogg Music Match app)
- `backend/` – Go REST API server (in-memory demo)
- `pulumi/` – Kubernetes infrastructure as code

## Quick Start with Docker Compose

The fastest way to run the entire application:

```bash
docker compose up --build
```

Then visit http://localhost:4200

- Frontend: http://localhost:4200
- Backend API: http://localhost:8080

## Backend (Go) API

Run the server locally:

```bash
cd backend
go mod tidy
go run .
```

Server listens on `:8080` by default.

### Endpoints

`POST /register`
Request JSON:
```json
{ 
  "username": "student123", 
  "email": "student@example.com", 
  "firstName": "Student", 
  "lastName": "Name",
  "password": "SecurePass123!"
}
```
Response JSON:
```json
{ 
  "user": {
    "username": "student123",
    "email": "student@example.com", 
    "firstName": "Student", 
    "lastName": "Name",
    "artists": []
  }
}
```

`POST /login`
Request JSON:
```json
{ "username": "student123", "password": "SecurePass123!" }
```
Response JSON:
```json
{ 
  "user": {
    "username": "student123",
    "email": "student@example.com", 
    "firstName": "Student", 
    "lastName": "Name",
    "artists": ["...saved artists..."]
  }
}
```

`POST /findMusicMatches`
Request JSON:
```json
{ "artists": ["Artist A", "Artist B", "..."] }
```
Optional header: `X-User-Email: student@example.com` to associate submitted artists with an existing user (persisting them for future login responses).

Response JSON (top 5 matches):
```json
[
	{ "name": "Peer 1", "overlap": 3, "score": 0.812 },
	{ "name": "Peer 2", "overlap": 2, "score": 0.667 }
]
```

Scoring mixes overlap count and a Jaccard-like ratio, then sorts by score, overlap, and name.

`GET /health`
Simple health check returning status + timestamp.

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

### Notes

- Storage is in-memory only (resets on restart).
- Authentication uses bcrypt password hashing for security
- CORS is configured for local development.
- User data includes separate firstName/lastName fields for better personalization
- Extend by replacing in-memory store with a database (e.g., Postgres) and adding JWT tokens.

## Front-End (Angular)

Run the development server:

```bash
cd ui
npm install
npm start
```

Visit http://localhost:4200

### Features

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

The application enforces strong password security:
- **Length**: Minimum 8 characters
- **Uppercase**: At least one uppercase letter (A-Z)
- **Lowercase**: At least one lowercase letter (a-z)  
- **Numbers**: At least one digit (0-9)
- **Special Characters**: At least one symbol (!@#$%^&*(),.?":{}|<>_)

Real-time visual indicators show which requirements are met as you type.

## Docker

### Quick Start

Run both services with Docker Compose:

```bash
docker compose up --build
```

This will:
- Build both backend and frontend images
- Start backend on http://localhost:8080
- Start frontend on http://localhost:4200
- Configure proper API connectivity between services

### Building Images Individually

```bash
# Backend
docker build -t kellogg-music-match-backend:latest ./backend

# Frontend (with correct API URL)
docker build --build-arg API_BASE_URL=http://localhost:8080 -t kellogg-music-match-ui:latest ./ui
```

### Manual Container Run

```bash
# Start backend
docker run -d -p 8080:8080 --name kmm-backend kellogg-music-match-backend:latest

# Start frontend (connected to backend)
docker run -d -p 4200:80 --name kmm-ui kellogg-music-match-ui:latest
```

### Multi-arch build (example linux/amd64 + linux/arm64)

```bash
docker buildx create --name kmm-builder --use --bootstrap
docker buildx build --platform linux/amd64,linux/arm64 -t yourrepo/kmm-backend:latest ./backend --push
docker buildx build --platform linux/amd64,linux/arm64 -t yourrepo/kmm-ui:latest ./ui --push
```

### Runtime API URL Override

The UI image serves a `config.json` file that can be overridden at runtime for different environments:

```bash
docker run -d -p 4200:80 \
	-v $(pwd)/my-config.json:/usr/share/nginx/html/config.json:ro \
	kellogg-music-match-ui:latest
```

`my-config.json` example:
```json
{ "apiBaseUrl": "https://api.example.com" }
```

## Kubernetes Deployment

Deploy to Kubernetes using Pulumi:

```bash
cd pulumi
pulumi up
```

This creates:
- Backend deployment and service
- Frontend deployment and service  
- Ingress configuration with traefik.me DNS
- Proper service networking and load balancing

See `pulumi/README.md` for detailed deployment instructions.

## Development Setup

### Prerequisites
- Node.js 18+ and npm
- Go 1.22+
- Docker and Docker Compose
- (Optional) kubectl and Pulumi for Kubernetes deployment

### Local Development Workflow

1. **Start Backend**:
   ```bash
   cd backend
   go mod tidy
   go run .
   ```

2. **Start Frontend** (in another terminal):
   ```bash
   cd ui
   npm install
   npm start
   ```

3. **Access Application**:
   - Frontend: http://localhost:4200
   - Backend API: http://localhost:8080
   - Health Check: http://localhost:8080/health

## Architecture

```
┌─────────────────┐    HTTP/REST    ┌─────────────────┐
│   Angular UI    │◄───────────────►│   Go Backend    │
│   (Port 4200)   │                 │   (Port 8080)   │
│                 │                 │                 │
│ • Registration  │                 │ • User Auth     │
│ • Login         │                 │ • Password Hash │
│ • Artist Mgmt   │                 │ • Music Match   │
│ • Match Results │                 │ • In-Memory DB  │
└─────────────────┘                 └─────────────────┘
```

## Future Improvements

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

