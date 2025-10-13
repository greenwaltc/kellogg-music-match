# Kellogg Music Match UI

Angular standalone (v17+) application for Kellogg student registration, authentication, and music taste matching with enhanced profile management.

## Features

### 🔐 Authentication & Registration
- **Enhanced Student Registration**: 
  - Personal Details: Username, email, first name, last name
  - Kellogg Profile: Program selection and graduation year
  - Password Security: Strong password requirements with real-time validation
- **Program Selection**: Dropdown for Kellogg programs (2Y, 1Y, MMM, MBAi, JD-MBA, MD-MBA, EWMBA, JV)
- **Graduation Year**: Validation for years 2025-2030
- **Strong Password Requirements**: 
  - Minimum 8 characters
  - Uppercase, lowercase, number, and special character (!@#$%^&*(),.?":{}|<>_)
  - Real-time validation with visual feedback
  - Password strength indicator (Weak/Fair/Good/Strong)
- **Password Confirmation**: Real-time matching validation
- **Show/Hide Password**: Toggle visibility for both password fields
- **Login**: Secure username/password authentication with session management

### 🎵 Music Features  
- **Artist Management**: Dynamic list (1–10 artists) with add/remove functionality
- **Music Matching**: Find users with similar taste via Spotify-derived top artists (body currently ignored)
- **Time Range Toggle**: `short`, `medium`, `long` term Spotify ranges (persisted in localStorage + URL param)
- **Match Limits**: Adjustable `limit` (users) and `overlapsLimit` (max overlapping artists shown per match)
- **Structured Overlaps**: Each overlap includes `name`, `anchorRank`, `otherRank` (rank metadata for richer UI)
- **Skeleton Loading**: Shimmer placeholders reduce layout shift while fetching
- **Reduced Motion Support**: Animations suppressed when `prefers-reduced-motion` is set

### 🎨 User Experience
- **Real-time Validation**: Immediate feedback on all form inputs
- **Responsive Design**: Works seamlessly on desktop and mobile
- **Error Handling**: Clear error messages with retry guidance
- **Auth Persistence**: localStorage keeps users logged in across sessions
- **Angular Routing**: `/` login, `/artists` favorites, `/matches` results

## Quick Start

```bash
cd ui
npm install
npm start
```
Visit http://localhost:4200/ in your browser.

### Password Requirements

When registering, your password must meet these requirements:
- ✅ **8+ characters**: Minimum length for security
- ✅ **Uppercase letter**: At least one A-Z
- ✅ **Lowercase letter**: At least one a-z  
- ✅ **Number**: At least one digit 0-9
- ✅ **Special character**: At least one symbol (!@#$%^&*(),.?":{}|<>_)

Visual indicators show which requirements are met as you type, and a strength meter helps you create a secure password.

## API Integration

The frontend integrates with the Go backend API for all data operations.

### Authentication Endpoints

**POST /register**
```json
{
  "username": "student123",
  "email": "student@kellogg.northwestern.edu", 
  "firstName": "Jane",
  "lastName": "Doe",
  "password": "SecurePass123!"
}
**POST /register**
```json
{
  "username": "student123",
  "email": "student@kellogg.northwestern.edu",
  "firstName": "John",
  "lastName": "Doe", 
  "password": "SecurePass123!",
  "program": "2Y",
  "graduationYear": 2026
}
```

**POST /login**  
```json
{
  "username": "student123",
  "password": "SecurePass123!"
}
```

### Music Endpoints

**POST /findMusicMatches?range=medium_term&limit=10&overlapsLimit=5**
```json
{}
```

Query Parameters:
- `range` (optional) short_term | medium_term | long_term (default from backend config)
- `limit` (optional) number of users (server-capped)
- `overlapsLimit` (optional) truncate overlap list per user match

Response (example – truncated overlaps):
```json
[
  {
    "name": "Alice Johnson",
    "overlap": 7,
    "score": 0.83,
    "overlaps": [
      { "name": "Phoebe Bridgers", "anchorRank": 1, "otherRank": 2 },
      { "name": "Taylor Swift",    "anchorRank": 2, "otherRank": 1 }
    ]
  }
]
```

Notes:
- Request body is currently ignored (kept only for backward compatibility); Spotify top artists drive similarity.
- Scores are normalized rank-weighted overlap values in [0,1].

## Configuration

### API base URL
- The UI uses a single `ApiBaseService` for all HTTP calls.
- Base URL resolution order:
  1) `window.__kmmConfig.apiBaseUrl` loaded from `/config.json` at runtime (written by the container entrypoint), otherwise
  2) `'/api'` fallback (default NGINX reverse proxy to backend).

### Runtime Configuration (config.json)
At container start, `docker/entrypoint.sh` writes `/usr/share/nginx/html/config.json` from environment. Example:

```json
{
  "apiBaseUrl": "/api",
  "vapidPublicKey": "<your VAPID public key>",
  "artistMinCount": 5,
  "artistMaxCount": 20,
  "spotifyClientId": "...",
  "spotifyRedirectUri": "https://your.app/spotify/callback"
}
```

You can override this at runtime by mounting a custom `config.json` or setting environment variables used by the entrypoint.

## Development

### Project Structure
```
src/
├── app/
│   ├── login.component.ts       # Registration/login with password validation
│   ├── artists.component.ts     # Artist management interface  
│   ├── matches.component.ts     # Match results display
│   ├── auth.service.ts          # Authentication API service
│   ├── auth.guard.ts            # Route protection
│   ├── match.service.ts         # Music matching API service
│   ├── password-validators.ts   # Custom password validation logic
│   └── theme.service.ts         # Dark/light theme support
├── environments/               # Environment configurations
└── styles.scss                # Global styling (SCSS root; imports feature partials like styles/_matches.scss)
```

## Customization
- **Max Artists**: Modify `maxArtists` in `artists.component.ts`
- **Password Rules**: Update validation in `password-validators.ts`  
- **Styling**: Global SCSS root + feature partials (e.g., `src/styles/_matches.scss`) replacing earlier monolithic styles
- **Design Tokens** (`src/styles/_tokens.scss`): Central gradients, neutral scale (`$neutral-50..900`), semantic aliases (`$color-control-*`, `$color-danger-*`, `$color-note-*`, `$color-empty-*`), elevation, radii, and `focus-ring` mixin. Imported via `@use 'styles/tokens' as *;`.
- **Themes**: Dark/light theme toggle via `theme.service.ts`

## Build & Deploy

### Development Build
```bash
npm run build
```
Outputs to `dist/kellogg-music-match-ui/`.

### Production Docker Build
```bash
docker build -t kellogg-music-match-ui:latest .
```

### Docker Deployment
```bash
# Run container
docker run -d -p 4200:80 kellogg-music-match-ui:latest

# With custom runtime config
docker run -d -p 4200:80 \
  -e VAPID_PUBLIC_KEY=... \
  -v $(pwd)/config.json:/usr/share/nginx/html/config.json:ro \
  kellogg-music-match-ui:latest
```

## Technical Details
- **Framework**: Angular 17+ with standalone components
- **HTTP Client**: Modern fetch-based HttpClient
- **Routing**: Angular Router with authentication guards
- **Forms**: Reactive forms with custom validators
- **Storage**: localStorage for session persistence
- **Styling**: SCSS tokens + CSS custom properties for theming (tokens feed feature partials; themes provide runtime color surfaces).
- **Build**: Angular CLI with production optimizations

## Future Enhancements
- **Testing**: Broader unit/integration coverage for new range + overlaps controls
- **Artist Autocomplete**: Spotify/MusicBrainz powered suggestions
- **Drag Reorder**: Manual ordering of preferred artists
- **Expanded Accessibility**: Additional ARIA roles & focus management
- **Virtual Scrolling**: For large match result sets (beyond pagination)
- **PWA Features**: Offline shell + install prompt
- **Dual Scoring Display**: Optional legacy PWO comparison badge
- **Analytics**: Interaction metrics for algorithm tuning
