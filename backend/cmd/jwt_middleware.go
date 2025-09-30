package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
)

// JWTMiddleware handles JWT authentication for protected routes
type JWTMiddleware struct {
	jwtService *business.JWTService
}

// NewJWTMiddleware creates a new JWT middleware
func NewJWTMiddleware(jwtService *business.JWTService) *JWTMiddleware {
	return &JWTMiddleware{
		jwtService: jwtService,
	}
}

// contextKey is used for storing user info in request context
type contextKey string

const UserContextKey contextKey = "user"

// UserContext represents user information stored in request context
type UserContext struct {
	UserID   string
	Username string
	Email    string
}

// Middleware function that validates JWT tokens
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for public endpoints
		if isPublicEndpoint(r.URL.Path, r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Check for legacy X-User-Username header for backward compatibility
			username := r.Header.Get("X-User-Username")
			if username != "" {
				// For backward compatibility, create a minimal user context
				userCtx := &UserContext{
					Username: username,
				}
				ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if it's a Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Bearer token required", http.StatusUnauthorized)
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString) // Handle extra spaces

		// Validate the token
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user info to request context
		userCtx := &UserContext{
			UserID:   claims.UserID,
			Username: claims.Username,
			Email:    claims.Email,
		}
		ctx := context.WithValue(r.Context(), UserContextKey, userCtx)

		// Continue to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicEndpoint determines if an endpoint should be accessible without authentication
func isPublicEndpoint(path, method string) bool {
	publicEndpoints := map[string][]string{
		"/health":   {"GET"},
		"/login":    {"POST"},
		"/register": {"POST"},
	}

	if methods, exists := publicEndpoints[path]; exists {
		for _, m := range methods {
			if m == method {
				return true
			}
		}
	}

	return false
}

// GetUserFromContext extracts user information from request context
func GetUserFromContext(ctx context.Context) (*UserContext, bool) {
	user, ok := ctx.Value(UserContextKey).(*UserContext)
	return user, ok
}
