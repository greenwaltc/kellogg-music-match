package main

import (
	"log"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

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

	log.Fatal(http.ListenAndServe(":8080", router))
}
