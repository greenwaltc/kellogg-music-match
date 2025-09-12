# kellogg-music-match

Monorepo containing:

- `ui/` – Angular front-end (Kellogg Music Match app)
- `backend/` – Go REST API server (in-memory demo)
- `pulumi/` – (infrastructure as code placeholder)

## Backend (Go) API

Run the server:

```bash
cd backend
go mod tidy
go run .
```

Server listens on `:8080` by default.

### Endpoints

`POST /login`
Request JSON:
```json
{ "email": "student@example.com", "fullName": "Student Name" }
```
Response JSON:
```json
{ "email": "student@example.com", "fullName": "Student Name", "artists": ["...optional..."] }
```

Creates or updates a user (in-memory) and returns existing stored artists if any.

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

### Notes

- Storage is in-memory only (resets on restart).
- No authentication beyond trusting provided email; suitable only for prototyping.
- CORS is wide open (`*`) for local development.
- Extend by replacing in-memory store with a database (e.g., Postgres) and real auth (JWT/OIDC).

## Front-End

```bash
cd ui
npm install
npm start
```

Configure API base URL in `ui/src/environments/environment.ts` if deploying separately.

## Docker

Build images individually:

```bash
docker build -t kmm-backend ./backend
docker build -t kmm-ui ./ui
```

Run both with Docker Compose (frontend on :4200, backend on :8080):

```bash
docker compose up --build
```

Then visit http://localhost:4200

### Multi-arch build (example linux/amd64 + linux/arm64)

```bash
docker buildx create --name kmm-builder --use --bootstrap
docker buildx build --platform linux/amd64,linux/arm64 -t yourrepo/kmm-backend:latest ./backend --push
docker buildx build --platform linux/amd64,linux/arm64 -t yourrepo/kmm-ui:latest ./ui --push
```

### Runtime API URL Override

The UI image serves a `config.json` file (copied at build). Override at runtime by mounting a new file:

```bash
docker run -d -p 4200:80 \
	-v $(pwd)/my-config.json:/usr/share/nginx/html/config.json:ro \
	kmm-ui
```

`my-config.json` example:
```json
{ "apiBaseUrl": "https://api.example.com" }
```


## Future Improvements

- Persist users + artists in a database
- Add real auth (session/JWT) and CSRF protection
- Rate limiting / abuse protection
- Additional recommendation logic (e.g., weighted by ranking order)
- Tests for backend match scoring

