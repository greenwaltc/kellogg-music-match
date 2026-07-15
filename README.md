# Kellogg Music Match

A full-stack music taste matching application for Kellogg students. Users connect their Spotify account, and the app finds classmates with similar music taste using a rank-weighted overlap similarity engine. It also surfaces upcoming Chicago-area concerts via the Ticketmaster API so matched users can find shows to attend together.

## How It Works

1. Students register with their Kellogg profile (program, graduation year) and connect Spotify.
2. The backend stores each user's Spotify top artists per listening window (`short_term`, `medium_term`, `long_term`).
3. Match scores are computed in Go using rank-weighted overlap: artists ranked highly by both users contribute more, and scores are normalized to [0, 1]. See [docs/music_matching.md](docs/music_matching.md) for the algorithm details.
4. A dedicated events page shows six-plus months of Chicago concerts, synced from Ticketmaster every 24 hours, with search and infinite scroll.

Track-based matching also exists behind the `MATCH_TRACKS_ENABLED` feature flag (see [docs/track_matching_strategy.md](docs/track_matching_strategy.md)).

## Tech Stack

| Layer | Technology |
| ----- | ---------- |
| Backend | Go 1.24, OpenAPI-generated server, SQLC, JWT auth, Prometheus + OpenTelemetry |
| Frontend | Angular 17 (PWA with service worker), Nginx |
| Database | PostgreSQL 16 with Flyway migrations |
| Integrations | Spotify API, Ticketmaster Discovery API, SendGrid (email), Web Push / APNS / FCM (notifications) |
| Infrastructure | Docker Compose (local), Pulumi + Kubernetes/k3s (deployment) |

## Project Structure

```
kellogg-music-match/
├── backend/             # Go backend
│   ├── openapi.yaml     # API specification (source of generated server code)
│   ├── business/        # Business logic: matching, auth, repository, concerts
│   ├── db/              # Flyway migrations and SQLC queries
│   └── cmd/             # Application entry point
├── ui/                  # Angular frontend
│   └── src/app/         # Components and services
├── database/            # Flyway configuration
├── pulumi/              # Infrastructure as Code
├── scripts/             # Deployment and utility scripts
├── docs/                # Design docs (matching algorithm, etc.)
├── Makefile             # Top-level orchestration
└── docker-compose.yml   # Local development environment
```

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Make
- Go 1.24+ and Node.js 18+ (only for development outside containers)

### Setup

```bash
git clone https://github.com/greenwaltc/kellogg-music-match.git
cd kellogg-music-match

# Configure credentials (Spotify is required for matching;
# Ticketmaster, SendGrid, and push keys are optional)
cp .env_template .env
# edit .env

# Build and start everything (Postgres, Flyway migrations, backend, UI)
make dev
```

### Access

- Frontend: http://localhost:4200
- Backend API: http://localhost:8080
- Health check: http://localhost:8080/health
- Database: `localhost:5432` (user `kellogg_user`, db `kellogg_music_match`)

## Environment Variables

Copy `.env_template` to `.env` and fill in the values you need:

| Variable | Purpose |
| -------- | ------- |
| `SPOTIFY_CLIENT_ID` / `SPOTIFY_CLIENT_SECRET` / `SPOTIFY_REDIRECT_URI` | Spotify OAuth for top-artist sync (core matching feature) |
| `TICKETMASTER_CONSUMER_KEY` / `TICKETMASTER_CONSUMER_SECRET` | Concert discovery ([TICKETMASTER_INTEGRATION.md](TICKETMASTER_INTEGRATION.md)) |
| `SENDGRID_API_KEY` | Transactional email, e.g. password reset ([SENDGRID_SETUP.md](SENDGRID_SETUP.md)) |
| `VAPID_PUBLIC_KEY` / `VAPID_PRIVATE_KEY` / `VAPID_SUBJECT` | Web push notifications (generate with `make generate-vapid-keys`) |
| `APNS_*` / `FCM_*` | Optional native mobile push |
| `DB_PASSWORD` | Database password override |

Additional backend knobs (defaults set in `docker-compose.yml`): `MATCH_TRACKS_ENABLED`, `MATCHING_ARTIST_TOPN`, `MATCHING_TRACK_TOPN`, and `TICKETMASTER_*` settings for location, radius, and date range.

## Development Commands

Run `make help` for the full list. The most common:

```bash
make dev              # Start the full Docker Compose environment
make status           # Service status and backend health
make test             # Run all tests
make build            # Regenerate OpenAPI + SQLC code
make docker-logs      # Tail container logs
make clean            # Stop containers and remove volumes

# Database
make db-migrate       # Apply Flyway migrations
make db-connect       # psql shell into the database

# Component-specific (forwarded to backend Makefile / ui npm scripts)
make backend-test     # Go unit tests + Ginkgo behavioral tests
make backend-dev      # Backend with live reload (air)
make ui-start         # Angular dev server
make ui-test          # Frontend unit tests (Karma/Jasmine)
make ui-e2e           # Playwright end-to-end tests
```

To debug the backend with Delve, use `make docker-run-debug` (debugger listens on `127.0.0.1:2345`; a launch config is provided in `.vscode/`).

## API Overview

The API is defined in `backend/openapi.yaml`. Highlights:

- `POST /register`, `POST /login` — auth with bcrypt password hashing and JWT tokens
- `POST /findMusicMatches?range=medium_term&limit=10&overlapsLimit=5` — find similar users; supports Spotify time ranges and is rate-limited (3 requests per 10 seconds per user, with `X-RateLimit-*` headers)
- `GET /chicago/events?limit=10&offset=0&artist=...` — paginated concert listing with case-insensitive artist search
- `GET /health` — service and database health

Match responses include a normalized `score`, the overlap count, and structured `overlaps` with both users' artist ranks:

```json
{
  "name": "Alice Johnson",
  "program": "2Y",
  "graduationYear": 2025,
  "overlap": 7,
  "score": 0.83,
  "overlaps": [
    { "name": "Phoebe Bridgers", "anchorRank": 1, "otherRank": 2 }
  ]
}
```

## Deployment

Local development uses Docker Compose (`make dev`). For Kubernetes:

```bash
cd pulumi/

pulumi config set postgres:password <secure-password>
pulumi config set ticketmaster:consumerKey <api-key> --secret
pulumi config set ticketmaster:consumerSecret <api-secret> --secret

pulumi up
```

For local k3s clusters, `make k3s-deploy` builds images, imports them into the cluster, and applies the manifests. See [K3S_LOCAL_IMAGES_GUIDE.md](K3S_LOCAL_IMAGES_GUIDE.md) and [pulumi/DEPLOYMENT.md](pulumi/DEPLOYMENT.md).

## Documentation

- [DOCS.md](DOCS.md) — documentation index
- [docs/music_matching.md](docs/music_matching.md) — matching algorithm design and rationale
- [backend/README.md](backend/README.md) — backend architecture and testing
- [DATABASE.md](DATABASE.md) / [DATABASE_SCHEMA.md](DATABASE_SCHEMA.md) — database setup and schema reference
- [FLYWAY-MIGRATION-SETUP.md](FLYWAY-MIGRATION-SETUP.md) — migration system
- [TICKETMASTER_INTEGRATION.md](TICKETMASTER_INTEGRATION.md) — concert sync service
- [SENDGRID_SETUP.md](SENDGRID_SETUP.md) — email configuration
