# Configuration Guide

This document describes the configurable environment variables for the Kellogg Music Match application.

## Configuration System

The application uses a centralized configuration system that loads all settings from environment variables with sensible defaults. Configuration is defined in `backend/config/config.go`.

## Environment Variables

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Port for the HTTP server |

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL server hostname |
| `DB_PORT` | `5432` | PostgreSQL server port |
| `DB_NAME` | `kellogg_music_match` | Database name |
| `DB_USER` | `kellogg_user` | Database username |
| `DB_PASSWORD` | `kellogg_secure_pass_2024` | Database password |
| `DB_SSLMODE` | `disable` | SSL mode for database connection |

### CORS Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | `http://localhost:4200,http://kmm-ui.traefik.me` | Comma-separated list of allowed origins |
| `CORS_ALLOWED_METHODS` | `GET, POST, PUT, DELETE, OPTIONS` | Allowed HTTP methods |
| `CORS_ALLOWED_HEADERS` | `Content-Type, Authorization, X-User-Username` | Allowed HTTP headers |
| `CORS_ALLOW_CREDENTIALS` | `true` | Whether to allow credentials in CORS requests |

### Artist Validation Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ARTIST_MIN_COUNT` | `5` | Minimum number of artists required for matching |
| `ARTIST_MAX_COUNT` | `20` | Maximum number of artists allowed for matching |
| `ARTIST_MAX_NAME_LENGTH` | `240` | Maximum length for artist names |
| `ARTIST_SEARCH_MAX_LENGTH` | `240` | Maximum length for artist search queries |
| `ARTIST_SEARCH_LIMIT` | `10` | Default limit for artist search results |

### Ticketmaster API Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `TICKETMASTER_CONSUMER_KEY` | *(empty)* | Ticketmaster Discovery API consumer key |
| `TICKETMASTER_CONSUMER_SECRET` | *(empty)* | Ticketmaster Discovery API consumer secret |
| `TICKETMASTER_BASE_URL` | `https://app.ticketmaster.com/discovery/v2` | Ticketmaster API base URL |
| `TICKETMASTER_TIMEOUT` | `10s` | HTTP timeout for Ticketmaster API requests |

### Debug Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DEBUG_ENABLED` | `false` | Enable debug logging and features |

### Legacy Variables (for backward compatibility)

| Variable | Description |
|----------|-------------|
| `PORT` | Alternative to `SERVER_PORT` |
| `DATABASE_URL` | Alternative connection string format |

## Example Configuration

### Development Environment

```bash
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=dev_user
export DB_PASSWORD=dev_password
export DEBUG_ENABLED=true
export ARTIST_MIN_COUNT=3
export ARTIST_MAX_COUNT=15

# Ticketmaster API (optional for concert integration)
export TICKETMASTER_CONSUMER_KEY=your_consumer_key_here
export TICKETMASTER_CONSUMER_SECRET=your_consumer_secret_here
```

### Production Environment

```bash
export SERVER_PORT=8080
export DB_HOST=postgres.production.com
export DB_PORT=5432
export DB_USER=prod_user
export DB_PASSWORD=secure_production_password
export DB_SSLMODE=require
export DEBUG_ENABLED=false
export CORS_ALLOWED_ORIGINS=https://music-match.kellogg.edu
```

## Kubernetes Configuration

The Pulumi deployment in `pulumi/main.go` includes all environment variables as ConfigMap entries. The deployment automatically configures:

- All database connection settings
- CORS policies for the web UI
- Artist validation constraints (5-20 artists)
- Security settings

## Configuration Loading

The configuration is loaded once at application startup via `config.Load()`. All services receive their configuration through dependency injection, making the system testable and maintainable.

## Validation Changes

The recent updates include:

1. **Artist Count Validation**: Now configurable (default 5-20 artists)
2. **All Hardcoded Values Removed**: Every configuration value can be overridden via environment variables
3. **Centralized Configuration**: Single source of truth in `backend/config/config.go`
4. **Type Safety**: Integer and boolean environment variables are properly parsed with fallbacks

## Testing Configuration

To test different configurations:

```bash
# Test with custom artist limits
ARTIST_MIN_COUNT=3 ARTIST_MAX_COUNT=15 ./server

# Test with debug mode
DEBUG_ENABLED=true ./server

# Test with custom CORS settings
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://test.local ./server
```