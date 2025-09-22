package main

import (
	"log"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// corsMiddleware handles CORS headers for cross-origin requests
func corsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the origin from the request
			origin := r.Header.Get("Origin")

			// Check if the origin is allowed
			var allowedOrigin string
			for _, allowed := range cfg.CORS.AllowedOrigins {
				if origin == allowed {
					allowedOrigin = origin
					break
				}
			}

			// Set CORS headers
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}
			w.Header().Set("Access-Control-Allow-Methods", cfg.CORS.AllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", cfg.CORS.AllowedHeaders)
			if cfg.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	log.Printf("Server started")

	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize database components
	userRepo, err := business.NewUserRepositoryWithConfig(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize user repository: %v", err)
	}

	matchingEngine := business.NewMatchingEngine()

	// Initialize business services with config
	authService := business.NewAuthService(userRepo)
	healthService := business.NewHealthService()
	matchingService := business.NewMatchingServiceWithConfig(userRepo, matchingEngine, &cfg.Artist)
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
	corsRouter := corsMiddleware(cfg)(router)

	serverAddr := ":" + cfg.Server.Port
	log.Printf("Server listening on %s with CORS enabled for origins: %v", serverAddr, cfg.CORS.AllowedOrigins)
	log.Fatal(http.ListenAndServe(serverAddr, corsRouter))
}
