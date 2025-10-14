package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/business/spotify"
	"github.com/greenwaltc/kellogg-music-match/backend/config"

	"github.com/google/uuid"
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

	// Initialize Spotify service with real token exchange support (HTTP client + credentials)
	spotifyService := spotify.NewService(
		userRepo,
		cfg.Spotify.RefreshTokenKey,
		spotify.WithSpotifyCredentials(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret, cfg.Spotify.RedirectURI),
	)

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
	matchingAPIService := NewMatchingAPIServiceWrapper(matchingService, spotifyService)
	feedbackAPIService := NewFeedbackAPIServiceWrapper(feedbackService)

	// Create controllers with our wrapped services
	AuthenticationAPIController := generated.NewAuthenticationAPIController(authAPIService)
	HealthAPIController := generated.NewHealthAPIController(healthAPIService)
	MatchingAPIController := generated.NewMatchingAPIController(matchingAPIService)
	FeedbackAPIController := generated.NewFeedbackAPIController(feedbackAPIService)
	// Wrap business concert API service with adapter implementing generated interface
	concertsAdapter := business.NewGeneratedConcertsAdapter(concertAPIService)
	ConcertsAPIController := generated.NewConcertsAPIController(concertsAdapter)

	router := generated.NewRouter(AuthenticationAPIController, HealthAPIController, MatchingAPIController, FeedbackAPIController, ConcertsAPIController)

	// Wrap generated router; override /sync/spotify/status with an extended handler that also reports readiness based on DB state.
	mux := http.NewServeMux()
	mux.Handle("/", router)
	mux.HandleFunc("/sync/spotify/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		uctx, ok := GetUserFromContext(r.Context())
		if !ok || uctx == nil || uctx.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Determine current job status from spotify service
		job := spotifyService.GetStatus(uctx.Username)
		status := job.Status
		// Compute readiness: true if we have any current Spotify snapshot rows (artists or tracks) for this user
		ready := false
		if pr, ok := userRepo.(*business.PostgreSQLUserRepository); ok {
			if parsed, err := uuid.Parse(uctx.UserID); err == nil {
				var hasArtists, hasTracks bool
				// v_current_* views reflect the latest snapshot per range
				q1 := `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_artists WHERE user_id=$1)`
				q2 := `SELECT EXISTS(SELECT 1 FROM v_current_spotify_top_tracks WHERE user_id=$1)`
				_ = pr.Pool().QueryRow(r.Context(), q1, parsed).Scan(&hasArtists)
				_ = pr.Pool().QueryRow(r.Context(), q2, parsed).Scan(&hasTracks)
				ready = hasArtists || hasTracks
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Include a minimal superset of the generated response: status + ready boolean for frontend consumption
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  status,
			"message": job.Message,
			"ready":   ready,
		})
	})

	// Push subscription endpoint
	mux.HandleFunc("/push/subscribe", NewSubscribeHandler(userRepo))

	// Test send endpoint (requires server to have VAPID keys)
	mux.HandleFunc("/push/test", NewTestHandler(userRepo, cfg, sendWebPush))

	// Initialize JWT middleware
	jwtMiddleware := NewJWTMiddleware(jwtService)

	// Wrap router with middleware layers (innermost to outermost)
	// Compose middleware: otelhttp -> jwt -> router
	otelWrapped := otelhttp.NewHandler(mux, "http.server")
	protectedRouter := jwtMiddleware.Middleware(otelWrapped)
	corsRouter := corsMiddleware(cfg)(protectedRouter)

	serverAddr := ":" + cfg.Server.Port
	logger.L().Info("server listening", "addr", serverAddr, "cors.origins", cfg.CORS.AllowedOrigins)
	if err := http.ListenAndServe(serverAddr, corsRouter); err != nil {
		logger.L().Error("server crashed", "error", err)
	}
}

// sendWebPush delivers a simple JSON notification to a subscription using VAPID keys from config.
func sendWebPush(subJSON []byte, cfg *config.Config) error {
	var sub struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(subJSON, &sub); err != nil {
		return err
	}
	s := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{Auth: sub.Keys.Auth, P256dh: sub.Keys.P256dh},
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"notification": map[string]interface{}{
			"title": "Kellogg Music Match",
			"body":  "Hello from KMM!",
			"icon":  "/assets/icons/icon-192x192.png",
			"badge": "/assets/icons/badge-72x72.png",
			"data": map[string]interface{}{
				"url": "/matches", // where to navigate on click
			},
			"requireInteraction": false,
		},
		"data": map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		},
	})
	resp, err := webpush.SendNotification(payload, s, &webpush.Options{
		Subscriber:      cfg.Push.Subject,
		VAPIDPublicKey:  cfg.Push.VAPIDPublic,
		VAPIDPrivateKey: cfg.Push.VAPIDPrivate,
		TTL:             60,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
