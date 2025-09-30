package main

import (
	"context"
	"log"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
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

	// Initialize JWT service
	jwtService := business.NewJWTService(&cfg.JWT)

	// Initialize business services with config
	authService := business.NewAuthService(userRepo, jwtService)
	healthService := business.NewHealthService()
	matchingService := business.NewMatchingServiceWithConfig(userRepo, matchingEngine, &cfg.Artist)
	feedbackService := business.NewFeedbackService(userRepo)

	// Initialize concert API service (will be enhanced with repository if available)
	var concertAPIService *business.ConcertAPIService

	// Initialize concert synchronization service
	var concertSyncService *concert.SyncService

	// Try to initialize concert service with Ticketmaster API
	tempConcertAPIService := business.NewConcertAPIService(cfg)
	if err := tempConcertAPIService.ValidateConfiguration(context.Background()); err != nil {
		log.Printf("Warning: Concert service configuration invalid: %v", err)
		log.Printf("Concert features will be disabled")
		concertAPIService = tempConcertAPIService // Use basic service without repository
	} else {
		log.Printf("Concert service initialized with Ticketmaster API")

		// Initialize concert repository
		concertRepo, err := concert.NewPostgreSQLRepository(&cfg.Database)
		if err != nil {
			log.Printf("Warning: Failed to initialize concert repository: %v", err)
			log.Printf("Concert sync will be disabled")
			concertAPIService = tempConcertAPIService // Use basic service without repository
		} else {
			log.Printf("Concert repository initialized successfully")

			// Create Ticketmaster event provider
			eventProvider := concert.NewTicketmasterAdapter(&cfg.Ticketmaster)

			// Create concert API service with repository access
			concertAPIService = business.NewConcertAPIServiceWithRepository(eventProvider, concertRepo, cfg)

			// Initialize and start concert sync service
			concertSyncService = concert.NewSyncService(eventProvider, concertRepo, cfg)

			// Start the sync service in a separate goroutine
			go func() {
				if err := concertSyncService.Start(context.Background()); err != nil {
					log.Printf("Error starting concert sync service: %v", err)
				}
			}()

			log.Printf("Concert sync service started - will sync every 24 hours")

			// Ensure graceful shutdown of sync service
			defer func() {
				if concertSyncService != nil {
					log.Printf("Shutting down concert sync service...")
					concertSyncService.Stop()
				}
				if concertRepo != nil {
					log.Printf("Closing concert repository connection...")
					concertRepo.Close()
				}
			}()
		}
	}

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
	ConcertsAPIController := generated.NewConcertsAPIController(concertAPIService)

	router := generated.NewRouter(AuthenticationAPIController, HealthAPIController, MatchingAPIController, FeedbackAPIController, ConcertsAPIController)

	// Initialize JWT middleware
	jwtMiddleware := NewJWTMiddleware(jwtService)

	// Wrap router with middleware layers (innermost to outermost)
	protectedRouter := jwtMiddleware.Middleware(router)
	corsRouter := corsMiddleware(cfg)(protectedRouter)

	serverAddr := ":" + cfg.Server.Port
	log.Printf("Server listening on %s with CORS enabled for origins: %v", serverAddr, cfg.CORS.AllowedOrigins)
	log.Fatal(http.ListenAndServe(serverAddr, corsRouter))
}
