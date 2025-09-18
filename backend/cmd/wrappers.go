package main

import (
	"context"

	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// AuthAPIServiceWrapper wraps business logic to implement OpenAPI service interface
type AuthAPIServiceWrapper struct {
	authService *business.AuthService
}

// NewAuthAPIServiceWrapper creates a new wrapper
func NewAuthAPIServiceWrapper(authService *business.AuthService) generated.AuthenticationAPIServicer {
	return &AuthAPIServiceWrapper{
		authService: authService,
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
}

// NewMatchingAPIServiceWrapper creates a new wrapper
func NewMatchingAPIServiceWrapper(matchingService *business.MatchingService) generated.MatchingAPIServicer {
	return &MatchingAPIServiceWrapper{
		matchingService: matchingService,
	}
}

// FindMusicMatches delegates to business logic
func (w *MatchingAPIServiceWrapper) FindMusicMatches(ctx context.Context, artistsRequest generated.ArtistsRequest, xUserUsername string) (generated.ImplResponse, error) {
	return w.matchingService.FindMusicMatches(ctx, artistsRequest, xUserUsername)
}

// SearchArtists delegates to business logic
func (w *MatchingAPIServiceWrapper) SearchArtists(ctx context.Context, q string, limit int32) (generated.ImplResponse, error) {
	return w.matchingService.SearchArtists(ctx, q, limit)
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
	feedback, err := w.feedbackService.SubmitFeedback(ctx, xUserUsername, feedbackRequest.Feedback)
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
