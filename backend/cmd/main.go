package main

import (
	"context"
	"net/http"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"github.com/greenwaltc/kellogg-music-match/backend/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	logger.Init()
	logger.L().Info("server starting")

	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize database components
	userRepo, err := business.NewUserRepositoryWithConfig(&cfg.Database)
	if err != nil {
		logger.L().Error("init user repository failed", "error", err)
		panic(err)
	}

	// Initialize telemetry (tracing & metrics) early
	telemetry.Init(cfg.Telemetry)

	matchingEngine := business.NewMatchingEngine()

	// Initialize JWT service
	jwtService := business.NewJWTService(&cfg.JWT)

	// Initialize email service
	emailService := business.NewEmailService(&cfg.Email)

	// Initialize business services with config
	authService := business.NewAuthService(userRepo, jwtService)
	passwordResetService := business.NewPasswordResetService(userRepo, emailService)
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
		logger.L().Warn("concert service invalid config", "error", err)
		logger.L().Info("concert features disabled")
		concertAPIService = tempConcertAPIService // Use basic service without repository
	} else {
		logger.L().Info("concert service initialized")

		// Initialize concert repository
		concertRepo, err := concert.NewPostgreSQLRepository(&cfg.Database)
		if err != nil {
			logger.L().Warn("concert repository init failed", "error", err)
			logger.L().Info("concert sync disabled")
			concertAPIService = tempConcertAPIService // Use basic service without repository
		} else {
			logger.L().Info("concert repository initialized")

			// Create Ticketmaster event provider
			eventProvider := concert.NewTicketmasterAdapter(&cfg.Ticketmaster)

			// Create concert API service with repository access
			concertAPIService = business.NewConcertAPIServiceWithRepository(eventProvider, concertRepo, cfg)

			// Initialize and start concert sync service
			concertSyncService = concert.NewSyncService(eventProvider, concertRepo, cfg)

			// Start the sync service in a separate goroutine
			go func() {
				if err := concertSyncService.Start(context.Background()); err != nil {
					logger.L().Error("concert sync start failed", "error", err)
				}
			}()

			logger.L().Info("concert sync scheduled", "intervalHours", 24)

			// Ensure graceful shutdown of sync service
			defer func() {
				if concertSyncService != nil {
					logger.L().Info("concert sync shutdown")
					concertSyncService.Stop()
				}
				if concertRepo != nil {
					logger.L().Info("concert repository closing")
					concertRepo.Close()
				}
			}()
		}
	}

	// Create service wrappers that implement the OpenAPI service interfaces
	authAPIService := NewAuthAPIServiceWrapper(authService, passwordResetService)
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
	// Compose middleware: otelhttp -> jwt -> router
	otelWrapped := otelhttp.NewHandler(router, "http.server")
	protectedRouter := jwtMiddleware.Middleware(otelWrapped)
	corsRouter := corsMiddleware(cfg)(protectedRouter)

	serverAddr := ":" + cfg.Server.Port
	logger.L().Info("server listening", "addr", serverAddr, "cors.origins", cfg.CORS.AllowedOrigins)
	if err := http.ListenAndServe(serverAddr, corsRouter); err != nil {
		logger.L().Error("server crashed", "error", err)
	}
}
