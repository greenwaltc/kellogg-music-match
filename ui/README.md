# Kellogg Music Match UI

Angular standalone (v17) application for user login/registration and music taste matching.

## Features
- Login/Register form (email + full name). Email placeholder: "Kellogg Student Email".
- Auth persistence (localStorage). Page reload keeps user logged in.
- Angular routing: `/` login, `/artists` submit favorites, `/matches` view results.
- Dynamic artist list (1–10) with plus button to add entries.
- POST /findMusicMatches returns up to 5 closest matching users; names displayed in ordered list.
- Error handling surfaces backend message + retry guidance.

## Quick Start

```bash
cd ui
npm install
npm start
```
Visit http://localhost:4200/ in your browser.

## API Contracts (assumed)
### POST /login
Request: `{ email: string, fullName: string }`
Response: `200 OK` (body ignored) or error `{ message: string }`.

### POST /findMusicMatches
Request: `{ artists: string[] }`
Response: `200 OK` JSON array like: `[ { "name": "Jane Doe" }, ... ]` (up to 5).
Error: `{ message: string }`.

Adjust parsing in `app.component.ts` if backend differs.

## Customization
- Max artists: modify `maxArtists` in `artists.component.ts`.
- Styling: shared styles in `app.component.css` & global `src/styles.css`.

## Build
```bash
npm run build
```
Outputs to `dist/kellogg-music-match-ui/`.

## Notes
- Uses fetch-based HttpClient (withFetch) for modern APIs.
- Standalone Angular components with router & guard for auth.

## Next Ideas
- Persist session (localStorage token) after /login.
- Add loading skeletons.
- Add unit tests (Jasmine/Karma or Jest) for form logic.
