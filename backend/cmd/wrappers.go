package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/spotify"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
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
type MatchingAPIServiceWrapper struct {
	matchingService *business.MatchingService
	spotifyService  *spotify.Service
}

// NewMatchingAPIServiceWrapper creates a new wrapper
func NewMatchingAPIServiceWrapper(matchingService *business.MatchingService, spotifyService *spotify.Service) generated.MatchingAPIServicer {
	return &MatchingAPIServiceWrapper{
		matchingService: matchingService,
		spotifyService:  spotifyService,
	}
}

// FindMusicMatches delegates to business logic
func (w *MatchingAPIServiceWrapper) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string) (generated.ImplResponse, error) {
	// Try to get user from JWT context first
	if user, ok := GetUserFromContext(ctx); ok && user.Username != "" {
		return w.matchingService.FindMusicMatches(ctx, artistsRequest, user.Username)
	}

	// Fall back to header-based auth for backward compatibility
	if xUserUsername != "" {
		return w.matchingService.FindMusicMatches(ctx, artistsRequest, xUserUsername)
	}

	return w.matchingService.FindMusicMatches(ctx, artistsRequest, "")
}

// SearchArtists delegates to business logic
func (w *MatchingAPIServiceWrapper) SearchArtists(ctx context.Context, q string, limit int32) (generated.ImplResponse, error) {
	return w.matchingService.SearchArtists(ctx, q, limit)
}

// Add SyncSpotify and GetSpotifySyncStatus to satisfy generated.MatchingAPIServicer
func (w *MatchingAPIServiceWrapper) SyncSpotify(ctx context.Context, body generated.SpotifySyncStartRequest) (generated.ImplResponse, error) {
	user, _ := GetUserFromContext(ctx)
	username := user.Username
	job := w.spotifyService.StartSync(ctx, username, body.Code, body.State)
	// Perform (stub) token exchange early; errors could mark job failed (future enhancement)
	accessToken, refreshToken, expiresIn, err := w.spotifyService.ExchangeCodeForTokens(ctx, body.Code)
	if err != nil {
		job.Status = spotify.StatusFailed
		job.Message = "Token exchange failed"
		return generated.Response(500, generated.ErrorResponse{Message: job.Message, CreatedAt: time.Now().UTC()}), nil
	}
	if user != nil && user.UserID != "" {
		if uid, err := uuid.Parse(user.UserID); err == nil {
			if perr := w.spotifyService.PersistTokens(ctx, uid, accessToken, refreshToken, expiresIn, ""); perr != nil { // scope empty stub
				job.Status = spotify.StatusFailed
				job.Message = "Persist tokens failed"
				return generated.Response(500, generated.ErrorResponse{Message: job.Message, CreatedAt: time.Now().UTC()}), nil
			}
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
	accessToken, refreshToken, expiresIn, err2 := w.spotifyService.ExchangeCodeForTokens(ctx, body.Code)
	if err2 == nil && user != nil && user.UserID != "" {
		if uid, perr := uuid.Parse(user.UserID); perr == nil {
			_ = w.spotifyService.PersistTokens(ctx, uid, accessToken, refreshToken, expiresIn, "")
		}
	}
	resp := generated.SpotifySyncAcceptedResponse{Status: "syncing", Message: job.Message}
	return generated.Response(202, resp), nil
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
