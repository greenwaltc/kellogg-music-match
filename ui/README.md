# Kellogg Music Match UI

Angular standalone (v17+) application for secure user registration, authentication, and music taste matching.

## Features

### 🔐 Authentication & Registration
- **Secure Registration**: Username, email, first name, last name, and password
- **Strong Password Requirements**: 
  - Minimum 8 characters
  - Uppercase, lowercase, number, and special character (!@#$%^&*(),.?":{}|<>_)
  - Real-time validation with visual feedback
  - Password strength indicator (Weak/Fair/Good/Strong)
- **Password Confirmation**: Real-time matching validation
- **Show/Hide Password**: Toggle visibility for both password fields
- **Login**: Secure username/password authentication

### 🎵 Music Features  
- **Artist Management**: Dynamic list (1–10 artists) with add/remove functionality
- **Music Matching**: Find users with similar taste via POST /findMusicMatches
- **Match Results**: Display top 5 closest matching users with overlap scores

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
```

**POST /login**  
```json
{
  "username": "student123",
  "password": "SecurePass123!"
}
```

### Music Endpoints

**POST /findMusicMatches**
```json
{ "artists": ["Taylor Swift", "The Beatles", "Radiohead"] }
```

Response: Array of up to 5 matches with overlap scores:
```json
[
  { "name": "Alice Johnson", "overlap": 2, "score": 0.75 },
  { "name": "Bob Smith", "overlap": 1, "score": 0.45 }
]
```

## Configuration

### Environment Settings
Configure API base URL in `src/environments/environment.ts`:

```typescript
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:8080'  // Backend API URL
};
```

### Runtime Configuration  
For Docker deployments, the app loads configuration from `/config.json`:

```json
{
  "apiBaseUrl": "https://api.yourdomain.com"
}
```

This allows runtime API URL changes without rebuilding the image.

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
└── styles.css                 # Global styling
```

## Customization
- **Max Artists**: Modify `maxArtists` in `artists.component.ts`
- **Password Rules**: Update validation in `password-validators.ts`  
- **Styling**: Shared styles in component CSS files and `src/styles.css`
- **Themes**: Dark/light theme toggle via `theme.service.ts`

## Build & Deploy

### Development Build
```bash
npm run build
```
Outputs to `dist/kellogg-music-match-ui/`.

### Production Docker Build
```bash
docker build --build-arg API_BASE_URL=http://localhost:8080 -t kellogg-music-match-ui:latest .
```

### Docker Deployment
```bash
# Run container
docker run -d -p 4200:80 kellogg-music-match-ui:latest

# Or with custom API URL
docker run -d -p 4200:80 \
  -v $(pwd)/config.json:/usr/share/nginx/html/config.json:ro \
  kellogg-music-match-ui:latest
```

## Technical Details
- **Framework**: Angular 17+ with standalone components
- **HTTP Client**: Modern fetch-based HttpClient
- **Routing**: Angular Router with authentication guards
- **Forms**: Reactive forms with custom validators
- **Storage**: localStorage for session persistence
- **Styling**: CSS custom properties for theming
- **Build**: Angular CLI with production optimizations

## Future Enhancements
- **Testing**: Add comprehensive unit tests (Jest/Jasmine) for components and services
- **Loading States**: Implement skeleton screens and loading indicators
- **Form Enhancements**: Auto-complete for artist names, drag-and-drop reordering
- **Social Features**: User profiles, friend connections, playlist sharing
- **Accessibility**: WCAG compliance, screen reader support, keyboard navigation
- **Performance**: Lazy loading, virtual scrolling for large lists
- **PWA Features**: Offline support, push notifications, app installation
- **Analytics**: User interaction tracking and usage metrics
