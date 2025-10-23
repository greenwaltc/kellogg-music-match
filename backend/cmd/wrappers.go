package main

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/spotify"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
	"github.com/jackc/pgx/v5"
)

// AuthAPIServiceWrapper wraps business logic to implement OpenAPI service interface
type AuthAPIServiceWrapper struct {
	authService          *business.AuthService
	passwordResetService *business.PasswordResetService
}

// NewAuthAPIServiceWrapper creates a new wrapper
func NewAuthAPIServiceWrapper(authService *business.AuthService, passwordResetService *business.PasswordResetService) generated.AuthenticationAPIServicer {
	return &AuthAPIServiceWrapper{
		authService:          authService,
		passwordResetService: passwordResetService,
	}
}

// RegisterUser delegates to business logic
func (w *AuthAPIServiceWrapper) RegisterUser(ctx context.Context, registerRequest generated.RegisterRequest) (generated.ImplResponse, error) {
	return w.authService.RegisterUser(ctx, registerRequest)
}

// LoginUser delegates to business logic
func (w *AuthAPIServiceWrapper) LoginUser(ctx context.Context, loginRequest generated.LoginRequest) (generated.ImplResponse, error) {
	return w.authService.LoginUser(ctx, loginRequest)
}

// ForgotPassword delegates to business logic
func (w *AuthAPIServiceWrapper) ForgotPassword(ctx context.Context, forgotPasswordRequest generated.ForgotPasswordRequest) (generated.ImplResponse, error) {
	return w.passwordResetService.ForgotPassword(ctx, forgotPasswordRequest)
}

// ResetPassword delegates to business logic
func (w *AuthAPIServiceWrapper) ResetPassword(ctx context.Context, resetPasswordRequest generated.ResetPasswordRequest) (generated.ImplResponse, error) {
	return w.passwordResetService.ResetPassword(ctx, resetPasswordRequest)
}

// HealthAPIServiceWrapper wraps business logic to implement OpenAPI service interface
type HealthAPIServiceWrapper struct {
	healthService *business.HealthService
}

// NewHealthAPIServiceWrapper creates a new wrapper
func NewHealthAPIServiceWrapper(healthService *business.HealthService) generated.HealthAPIServicer {
	return &HealthAPIServiceWrapper{
		healthService: healthService,
	}
}

// GetHealth delegates to business logic
func (w *HealthAPIServiceWrapper) GetHealth(ctx context.Context) (generated.ImplResponse, error) {
	return w.healthService.GetHealth(ctx)
}

// MatchingAPIServiceWrapper wraps business logic to implement OpenAPI service interface
type matchFinder interface {
	FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string, rng string, includeDetails bool, limit int32, overlapsLimit ...int32) (generated.ImplResponse, error)
}

type MatchingAPIServiceWrapper struct {
	matchingService matchFinder
	spotifyService  *spotify.Service
}

// context key type for matching basis
// (moved) matchBasisCtxKey replaced by business.MatchBasisContextKey in business/context_keys.go

// In-memory per-user rate limiter for FindMusicMatches.
// Simple sliding window allowing up to 3 requests per 10 second window.
// This guards backend from excessive refresh spam while remaining permissive for normal UI interactions.
type matchRateState struct {
	count       int
	windowStart time.Time
}

var (
	matchRateLimiterMu sync.Mutex
	matchRateLimiter   = make(map[string]*matchRateState) // key: username (empty allowed for anonymous)
	// Loosen: allow up to 5 requests within a 10s window
	matchRateLimitMax = 5
	// Keep 10s window; UI also debounces to reduce rapid bursts
	matchRateWindow = 10 * time.Second
)

// typed context key to store rate header info
type matchRateHeadersKey struct{}

type matchRateHeaders struct {
	Limit      string
	Remaining  string
	Window     string
	RetryAfter string
}

// resetMatchRateLimiter is only for tests.
// ResetMatchRateLimiter clears the in-memory rate limiter (test helper)
func ResetMatchRateLimiter() {
	matchRateLimiterMu.Lock()
	defer matchRateLimiterMu.Unlock()
	matchRateLimiter = make(map[string]*matchRateState)
}

// NewMatchingAPIServiceWrapper creates a new wrapper
func NewMatchingAPIServiceWrapper(matchingService *business.MatchingService, spotifyService *spotify.Service) generated.MatchingAPIServicer {
	return &MatchingAPIServiceWrapper{
		matchingService: matchingService,
		spotifyService:  spotifyService,
	}
}

// FindMusicMatches delegates to business logic
func (w *MatchingAPIServiceWrapper) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string, range_ string, basis string, includeDetails bool, userName string, userUsername string, limit int32, overlapsLimit int32) (generated.ImplResponse, error) {
	// Inject basis into context for business layer (default to artists)
	if basis == "" {
		basis = "artists"
	}
	ctx = context.WithValue(ctx, business.MatchBasisContextKey{}, basis)
	// includeDetails passed directly to service, no longer via context
	// Inject optional fuzzy name filter if provided
	if trimmed := strings.TrimSpace(userName); trimmed != "" {
		ctx = context.WithValue(ctx, business.MatchNameFilterContextKey{}, trimmed)
	}
	// Inject optional exact username filter for matching a specific other user
	if trimmedU := strings.TrimSpace(userUsername); trimmedU != "" {
		ctx = context.WithValue(ctx, business.MatchUsernameFilterContextKey{}, trimmedU)
	}
	username := ""
	if user, ok := GetUserFromContext(ctx); ok && user.Username != "" {
		username = user.Username
	} else if xUserUsername != "" {
		username = xUserUsername
	}

	// Apply rate limiting (counts anonymous separately using empty key)
	matchRateLimiterMu.Lock()
	st := matchRateLimiter[username]
	now := time.Now()
	if st == nil || now.Sub(st.windowStart) > matchRateWindow {
		st = &matchRateState{count: 0, windowStart: now}
		matchRateLimiter[username] = st
	}
	st.count++
	cur := st.count
	remaining := matchRateLimitMax - cur
	if remaining < 0 {
		remaining = 0
	}
	overLimit := cur > matchRateLimitMax
	matchRateLimiterMu.Unlock()

	// Stamp context with rate info struct
	headers := &matchRateHeaders{Limit: strconv.Itoa(matchRateLimitMax), Remaining: strconv.Itoa(remaining), Window: (matchRateWindow / time.Second).String() + "s"}

	if overLimit {
		retryAfter := int(matchRateWindow-now.Sub(st.windowStart)) / int(time.Second)
		if retryAfter < 1 {
			retryAfter = 1
		}
		headers.RetryAfter = strconv.Itoa(retryAfter)
		return generated.Response(429, generated.ErrorResponse{Message: "too many match requests - retry shortly", CreatedAt: time.Now().UTC()}), nil
	}
	return w.matchingService.FindMusicMatches(context.WithValue(ctx, matchRateHeadersKey{}, headers), artistsRequest, username, range_, includeDetails, limit, overlapsLimit)
}

// Add SyncSpotify and GetSpotifySyncStatus to satisfy generated.MatchingAPIServicer
func (w *MatchingAPIServiceWrapper) SyncSpotify(ctx context.Context, body generated.SpotifySyncStartRequest) (generated.ImplResponse, error) {
	user, _ := GetUserFromContext(ctx)
	username := user.Username
	job := w.spotifyService.StartSync(ctx, username, body.Code, body.State)
	// Attempt token exchange (PKCE supported if client provided code_verifier)
	var accessToken, refreshToken string
	var expiresIn int
	var err error
	if strings.TrimSpace(body.RedirectUri) != "" {
		accessToken, refreshToken, expiresIn, err = w.spotifyService.ExchangeCodeForTokensWithRedirect(ctx, body.Code, body.CodeVerifier, body.RedirectUri)
	} else {
		accessToken, refreshToken, expiresIn, err = w.spotifyService.ExchangeCodeForTokens(ctx, body.Code, body.CodeVerifier)
	}
	if err != nil {
		job.Status = spotify.StatusFailed
		job.Message = "Token exchange failed"
		logger.L().Error("spotify.sync.token_exchange.error", "err", err, "user", username)
		return generated.Response(500, generated.ErrorResponse{Message: job.Message, CreatedAt: time.Now().UTC()}), nil
	}
	if user != nil && user.UserID != "" {
		if uid, err := uuid.Parse(user.UserID); err == nil {
			if perr := w.spotifyService.PersistTokens(ctx, uid, accessToken, refreshToken, expiresIn, ""); perr != nil { // scope empty stub
				job.Status = spotify.StatusFailed
				job.Message = "Persist tokens failed"
				return generated.Response(500, generated.ErrorResponse{Message: job.Message, CreatedAt: time.Now().UTC()}), nil
			}
			// Provide token info to background ingestion job
			w.spotifyService.SetJobTokens(username, uid, accessToken, refreshToken, expiresIn)
		}
	}
	// Immediately mark job status as pending/syncing response
	resp := generated.SpotifySyncAcceptedResponse{Status: "syncing", Message: job.Message}
	return generated.Response(202, resp), nil
}

func (w *MatchingAPIServiceWrapper) GetSpotifySyncStatus(ctx context.Context) (generated.ImplResponse, error) {
	user, _ := GetUserFromContext(ctx)
	username := user.Username
	job := w.spotifyService.GetStatus(username)
	retryable := false
	if job.Status == spotify.StatusFailed || job.Status == spotify.StatusComplete || job.Status == spotify.StatusCancelled {
		retryable = true
	}
	resp := generated.SpotifySyncStatusResponse{Status: job.Status, Progress: job.Progress, Message: job.Message, Retryable: retryable}
	if job.StartedAt != nil {
		resp.StartedAt = job.StartedAt
	}
	if job.FinishedAt != nil {
		resp.FinishedAt = job.FinishedAt
	}
	// Backward compatibility: if no active job but previously complete, still 200.
	return generated.Response(200, resp), nil
}

// CancelSpotifySync cancels an active job
func (w *MatchingAPIServiceWrapper) CancelSpotifySync(ctx context.Context) (generated.ImplResponse, error) {
	user, _ := GetUserFromContext(ctx)
	username := user.Username
	w.spotifyService.CancelSync(username)
	return generated.Response(204, nil), nil
}

// RetrySpotifySync restarts a sync if previous job finished, failed, or was cancelled
func (w *MatchingAPIServiceWrapper) RetrySpotifySync(ctx context.Context, body generated.SpotifySyncStartRequest) (generated.ImplResponse, error) {
	user, _ := GetUserFromContext(ctx)
	username := user.Username
	job, err := w.spotifyService.RetrySync(ctx, username, body.Code, body.State)
	if err != nil {
		if errors.Is(err, spotify.ErrSyncInProgress) {
			return generated.Response(409, generated.ErrorResponse{Message: "Sync in progress", CreatedAt: time.Now().UTC()}), nil
		}
		return generated.Response(500, generated.ErrorResponse{Message: "Retry failed", CreatedAt: time.Now().UTC()}), nil
	}
	var accessToken, refreshToken string
	var expiresIn int
	var err2 error
	if strings.TrimSpace(body.RedirectUri) != "" {
		accessToken, refreshToken, expiresIn, err2 = w.spotifyService.ExchangeCodeForTokensWithRedirect(ctx, body.Code, body.CodeVerifier, body.RedirectUri)
	} else {
		accessToken, refreshToken, expiresIn, err2 = w.spotifyService.ExchangeCodeForTokens(ctx, body.Code, body.CodeVerifier)
	}
	if err2 == nil && user != nil && user.UserID != "" {
		if uid, perr := uuid.Parse(user.UserID); perr == nil {
			_ = w.spotifyService.PersistTokens(ctx, uid, accessToken, refreshToken, expiresIn, "")
		}
	}
	resp := generated.SpotifySyncAcceptedResponse{Status: "syncing", Message: job.Message}
	return generated.Response(202, resp), nil
}

// RefreshSpotifySync triggers a sync by refreshing access token using stored refresh token (no user interaction).
func (w *MatchingAPIServiceWrapper) RefreshSpotifySync(ctx context.Context) (generated.ImplResponse, error) {
	uctx, ok := GetUserFromContext(ctx)
	if !ok || uctx == nil || uctx.UserID == "" {
		return generated.Response(401, generated.ErrorResponse{Message: "unauthorized", CreatedAt: time.Now().UTC()}), nil
	}
	uid, err := uuid.Parse(uctx.UserID)
	if err != nil {
		return generated.Response(400, generated.ErrorResponse{Message: "invalid userId", CreatedAt: time.Now().UTC()}), nil
	}
	job, rerr := w.spotifyService.RefreshUsingStoredTokens(ctx, uctx.Username, uid)
	if rerr != nil {
		if errors.Is(rerr, spotify.ErrSyncInProgress) {
			return generated.Response(409, generated.ErrorResponse{Message: "Sync in progress", CreatedAt: time.Now().UTC()}), nil
		}
		// Surface not found when no stored tokens
		if strings.Contains(strings.ToLower(rerr.Error()), "no stored tokens") || errors.Is(rerr, pgx.ErrNoRows) {
			return generated.Response(404, generated.ErrorResponse{Message: "no stored tokens", CreatedAt: time.Now().UTC()}), nil
		}
		// Log details server-side for easier diagnosis
		logger.L().Error("spotify.refresh.error", "user", uctx.Username, "err", rerr)
		return generated.Response(500, generated.ErrorResponse{Message: "Refresh failed", CreatedAt: time.Now().UTC()}), nil
	}
	resp := generated.SpotifySyncAcceptedResponse{Status: "syncing", Message: job.Message}
	return generated.Response(202, resp), nil
}

// GetUserTopArtists enforces self-only access and delegates to business service for paginated results
func (w *MatchingAPIServiceWrapper) GetUserTopArtists(ctx context.Context, userId string, range_ string, limit int32, offset int32) (generated.ImplResponse, error) {
	uctx, ok := GetUserFromContext(ctx)
	if !ok || uctx == nil || uctx.UserID == "" {
		return generated.Response(401, generated.ErrorResponse{Message: "unauthorized", CreatedAt: time.Now().UTC()}), nil
	}
	// Self-only for now (admins not yet implemented)
	if strings.TrimSpace(strings.ToLower(userId)) != strings.TrimSpace(strings.ToLower(uctx.UserID)) {
		return generated.Response(403, generated.ErrorResponse{Message: "forbidden", CreatedAt: time.Now().UTC()}), nil
	}
	uid, err := uuid.Parse(userId)
	if err != nil {
		return generated.Response(400, generated.ErrorResponse{Message: "invalid userId", CreatedAt: time.Now().UTC()}), nil
	}
	// Delegate
	if ms, ok := w.matchingService.(*business.MatchingService); ok {
		return ms.GetUserTopArtistsPage(ctx, uid, range_, limit, offset)
	}
	return generated.Response(500, generated.ErrorResponse{Message: "service unavailable", CreatedAt: time.Now().UTC()}), nil
}

// GetUserTopTracks enforces self-only access and delegates to business service for paginated results
func (w *MatchingAPIServiceWrapper) GetUserTopTracks(ctx context.Context, userId string, range_ string, limit int32, offset int32) (generated.ImplResponse, error) {
	uctx, ok := GetUserFromContext(ctx)
	if !ok || uctx == nil || uctx.UserID == "" {
		return generated.Response(401, generated.ErrorResponse{Message: "unauthorized", CreatedAt: time.Now().UTC()}), nil
	}
	if strings.TrimSpace(strings.ToLower(userId)) != strings.TrimSpace(strings.ToLower(uctx.UserID)) {
		return generated.Response(403, generated.ErrorResponse{Message: "forbidden", CreatedAt: time.Now().UTC()}), nil
	}
	uid, err := uuid.Parse(userId)
	if err != nil {
		return generated.Response(400, generated.ErrorResponse{Message: "invalid userId", CreatedAt: time.Now().UTC()}), nil
	}
	if ms, ok := w.matchingService.(*business.MatchingService); ok {
		return ms.GetUserTopTracksPage(ctx, uid, range_, limit, offset)
	}
	return generated.Response(500, generated.ErrorResponse{Message: "service unavailable", CreatedAt: time.Now().UTC()}), nil
}

// FeedbackAPIServiceWrapper wraps business logic to implement OpenAPI service interface
type FeedbackAPIServiceWrapper struct {
	feedbackService *business.FeedbackService
}

// NewFeedbackAPIServiceWrapper creates a new wrapper
func NewFeedbackAPIServiceWrapper(feedbackService *business.FeedbackService) generated.FeedbackAPIServicer {
	return &FeedbackAPIServiceWrapper{
		feedbackService: feedbackService,
	}
}

// SubmitFeedback delegates to business logic
func (w *FeedbackAPIServiceWrapper) SubmitFeedback(ctx context.Context, xUserUsername string, feedbackRequest generated.FeedbackRequest) (generated.ImplResponse, error) {
	// Try to get user from JWT context first
	username := xUserUsername
	if user, ok := GetUserFromContext(ctx); ok && user.Username != "" {
		username = user.Username
	}

	feedback, err := w.feedbackService.SubmitFeedback(ctx, username, feedbackRequest.Feedback)
	if err != nil {
		return generated.Response(400, generated.ErrorResponse{
			Message: err.Error(),
		}), nil
	}

	createdAt := feedback.CreatedAt.Time
	if !feedback.CreatedAt.Valid {
		createdAt = feedback.CreatedAt.Time
	}

	response := generated.FeedbackResponse{
		Id:        int32(feedback.ID),
		Message:   "Feedback submitted successfully",
		CreatedAt: createdAt,
	}

	return generated.Response(201, response), nil
}
