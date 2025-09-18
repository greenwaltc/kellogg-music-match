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
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
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

	// Create service wrappers that implement the OpenAPI service interfaces
	authAPIService := NewAuthAPIServiceWrapper(authService)
	healthAPIService := NewHealthAPIServiceWrapper(healthService)
	matchingAPIService := NewMatchingAPIServiceWrapper(matchingService)

	// Create controllers with our wrapped services
	AuthenticationAPIController := generated.NewAuthenticationAPIController(authAPIService)
	HealthAPIController := generated.NewHealthAPIController(healthAPIService)
	MatchingAPIController := generated.NewMatchingAPIController(matchingAPIService)

	router := generated.NewRouter(AuthenticationAPIController, HealthAPIController, MatchingAPIController)

	// Wrap router with CORS middleware
	corsRouter := corsMiddleware(router)

	log.Printf("Server listening on :8080 with CORS enabled for http://localhost:4200")
	log.Fatal(http.ListenAndServe(":8080", corsRouter))
}
