package main

import (
	"log"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// corsMiddleware handles CORS headers for cross-origin requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the origin from the request
		origin := r.Header.Get("Origin")

		// List of allowed origins
		allowedOrigins := []string{
			"http://localhost:4200",    // Local development
			"http://kmm-ui.traefik.me", // Kubernetes ingress
		}

		// Check if the origin is allowed
		var allowedOrigin string
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				allowedOrigin = origin
				break
			}
		}

		// Set CORS headers
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-Username")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Printf("Server started")

	// Initialize database components
	userRepo, err := business.NewUserRepository()
	if err != nil {
		log.Fatalf("Failed to initialize user repository: %v", err)
	}

	matchingEngine := business.NewMatchingEngine()

	// Initialize business services
	authService := business.NewAuthService(userRepo)
	healthService := business.NewHealthService()
	matchingService := business.NewMatchingService(userRepo, matchingEngine)
	feedbackService := business.NewFeedbackService(userRepo)

	// Create service wrappers that implement the OpenAPI service interfaces
	authAPIService := NewAuthAPIServiceWrapper(authService)
	healthAPIService := NewHealthAPIServiceWrapper(healthService)
	matchingAPIService := NewMatchingAPIServiceWrapper(matchingService)
	feedbackAPIService := NewFeedbackAPIServiceWrapper(feedbackService)

	// Create controllers with our wrapped services
	AuthenticationAPIController := generated.NewAuthenticationAPIController(authAPIService)
	HealthAPIController := generated.NewHealthAPIController(healthAPIService)
	MatchingAPIController := generated.NewMatchingAPIController(matchingAPIService)
	FeedbackAPIController := generated.NewFeedbackAPIController(feedbackAPIService)

	router := generated.NewRouter(AuthenticationAPIController, HealthAPIController, MatchingAPIController, FeedbackAPIController)

	// Wrap router with CORS middleware
	corsRouter := corsMiddleware(router)

	log.Printf("Server listening on :8080 with CORS enabled for multiple origins")
	log.Fatal(http.ListenAndServe(":8080", corsRouter))
}
