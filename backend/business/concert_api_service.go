package business

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
)

// setMyInterest inspects deprecated user ID buckets to derive the caller's current interest.
// This avoids scanning name arrays (which hold display names only) and keeps logic consistent
// while we maintain deprecated *_UserIds fields for backward compatibility.
func setMyInterest(userID string, evt *concert.Event) {
	if evt == nil || userID == "" {
		return
	}
	for _, id := range evt.InterestedUserIDs {
		if id == userID {
			status := "INTERESTED"
			evt.MyInterest = &status
			return
		}
	}
	for _, id := range evt.GoingUserIDs {
		if id == userID {
			status := "GOING"
			evt.MyInterest = &status
			return
		}
	}
	for _, id := range evt.LookingForGroupUserIDs {
		if id == userID {
			status := "LOOKING_FOR_GROUP"
			evt.MyInterest = &status
			return
		}
	}
	evt.MyInterest = nil
}

// duplicate of jwt middleware user context key/shape (avoid import cycle with main package)
type userContextKeyType string

const userCtxKey userContextKeyType = "user"

type userContextMirror struct {
	UserID   string
	Username string
	Email    string
}

func getUserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// 1. Typed business key variant
	if v := ctx.Value(userCtxKey); v != nil {
		switch val := v.(type) {
		case *userContextMirror:
			if val != nil && val.UserID != "" {
				return val.UserID
			}
		case interface{ GetUserID() string }:
			if id := val.GetUserID(); id != "" {
				return id
			}
		case interface{ GetId() string }:
			if id := val.GetId(); id != "" {
				return id
			}
		}
	}
	// 2. Plain string key (set by middleware bridge)
	if v := ctx.Value("user"); v != nil {
		switch val := v.(type) {
		case *userContextMirror:
			if val != nil && val.UserID != "" {
				return val.UserID
			}
		case interface{ GetUserID() string }:
			if id := val.GetUserID(); id != "" {
				return id
			}
		case interface{ GetId() string }:
			if id := val.GetId(); id != "" {
				return id
			}
		}
	}
	// 3. Logger-attached context key (avoid import cycle by string compare on key name)
	//    We cannot import logger here; rely on known key retrieval at runtime if present.
	//    (If needed we could expose a lightweight identity interface shared across packages.)
	return ""
}

// ConcertAPIService handles concert API operations
type ConcertAPIService struct {
	concertService *ConcertService
}

// NewConcertAPIService creates a new concert API service
func NewConcertAPIService(cfg *config.Config) *ConcertAPIService {
	return &ConcertAPIService{
		concertService: NewConcertService(cfg),
	}
}

// NewConcertAPIServiceWithRepository creates a new concert API service with repository access
func NewConcertAPIServiceWithRepository(provider concert.EventProvider, repository concert.Repository, cfg *config.Config) *ConcertAPIService {
	return &ConcertAPIService{
		concertService: NewConcertServiceWithRepository(provider, repository, cfg),
	}
}

// ValidateConfiguration validates the concert service configuration
func (s *ConcertAPIService) ValidateConfiguration(ctx context.Context) error {
	return s.concertService.ValidateConfiguration(ctx)
}

// SearchConcerts implements the search concerts API endpoint
func (s *ConcertAPIService) SearchConcerts(ctx context.Context, artist string, city string, state string, country string, genre string, startDate string, endDate string, page int32, size int32) (generated.ImplResponse, error) {
	// Build search criteria from parameters
	criteria := concert.SearchCriteria{
		Artist:  artist,
		City:    city,
		State:   state,
		Country: country,
		Genre:   genre,
	}

	// Parse dates if provided
	if startDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", startDate); err == nil {
			criteria.StartDate = parsedDate
		}
	}

	if endDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", endDate); err == nil {
			criteria.EndDate = parsedDate
		}
	}

	// Set pagination
	if page < 0 {
		page = 0
	}
	if size <= 0 || size > 200 {
		size = 20
	}

	// TODO: Add pagination support to the provider interface
	// For now, we'll ignore pagination and return all results

	// Search for events
	result, err := s.concertService.GetUpcomingEvents(ctx, criteria)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: fmt.Sprintf("Failed to search concerts: %v", err),
		}), nil
	}

	// Convert to API response format
	apiResult := s.convertToAPIConcertSearchResult(result)

	return generated.Response(http.StatusOK, apiResult), nil
}

// GetConcertById implements the get concert by ID API endpoint
func (s *ConcertAPIService) GetConcertById(ctx context.Context, eventId string) (generated.ImplResponse, error) {
	if eventId == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{
			Message: "Event ID is required",
		}), nil
	}

	event, err := s.concertService.GetEventDetails(ctx, eventId)
	if err != nil {
		// Check if it's a not found error or a server error
		if err.Error() == "event not found" {
			return generated.Response(http.StatusNotFound, generated.ErrorResponse{
				Message: "Concert not found",
			}), nil
		}

		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: fmt.Sprintf("Failed to get concert details: %v", err),
		}), nil
	}

	// Convert to API response format
	apiConcert := s.convertToAPIConcert(*event)

	return generated.Response(http.StatusOK, apiConcert), nil
}

// convertToAPIConcertSearchResult converts business model to API model
func (s *ConcertAPIService) convertToAPIConcertSearchResult(result *concert.SearchResult) generated.ConcertSearchResult {
	apiEvents := make([]generated.Concert, len(result.Events))
	for i, event := range result.Events {
		apiEvents[i] = s.convertToAPIConcert(event)
	}

	return generated.ConcertSearchResult{
		Events:      apiEvents,
		TotalCount:  int32(result.TotalCount),
		CurrentPage: int32(result.CurrentPage),
		TotalPages:  int32(result.TotalPages),
		HasMore:     result.HasMore,
	}
}

// convertToAPIConcert converts business concert model to API model
func (s *ConcertAPIService) convertToAPIConcert(event concert.Event) generated.Concert {
	// Convert venue
	apiVenue := generated.Venue{
		Name: event.Venue.Name,
		Address: generated.Address{
			Street:     event.Venue.Address.Street,
			City:       event.Venue.Address.City,
			State:      event.Venue.Address.State,
			Country:    event.Venue.Address.Country,
			PostalCode: event.Venue.Address.Postal,
		},
	}

	// Convert artists
	apiArtists := make([]generated.ConcertArtist, len(event.Artists))
	for i, artist := range event.Artists {
		apiArtists[i] = generated.ConcertArtist{
			Name:   artist.Name,
			Genres: artist.Genres,
		}
	}

	// Convert price range if available
	var apiPriceRange generated.PriceRange
	hasPriceRange := false
	if event.PriceRange.Min > 0 || event.PriceRange.Max > 0 {
		apiPriceRange = generated.PriceRange{
			Min:      float32(event.PriceRange.Min),
			Max:      float32(event.PriceRange.Max),
			Currency: event.PriceRange.Currency,
		}
		hasPriceRange = true
	}

	concert := generated.Concert{
		Id:        event.ID,
		Name:      event.Name,
		Date:      event.Date,
		Venue:     apiVenue,
		Artists:   apiArtists,
		Genres:    event.Genres,
		Relevancy: event.Relevancy,
	}

	// Include user interest aggregates if present
	if len(event.InterestedUserIDs) > 0 {
		concert.InterestedUserIds = event.InterestedUserIDs // deprecated retention
	}
	if len(event.GoingUserIDs) > 0 {
		concert.GoingUserIds = event.GoingUserIDs // deprecated retention
	}
	if len(event.LookingForGroupUserIDs) > 0 {
		concert.LookingForGroupUserIds = event.LookingForGroupUserIDs // deprecated retention
	}
	if len(event.InterestedUsers) > 0 {
		concert.InterestedUsers = event.InterestedUsers
	}
	if len(event.GoingUsers) > 0 {
		concert.GoingUsers = event.GoingUsers
	}
	if len(event.LookingForGroupUsers) > 0 {
		concert.LookingForGroupUsers = event.LookingForGroupUsers
	}

	// Only include price range if it has values
	if hasPriceRange {
		concert.PriceRange = apiPriceRange
	}

	// Only include ticket URL if it's not empty
	if event.TicketURL != "" {
		concert.TicketUrl = event.TicketURL
	}
	// Include interest buckets (legacy + new)
	concert.InterestedUserIds = event.InterestedUserIDs
	concert.GoingUserIds = event.GoingUserIDs
	concert.LookingForGroupUserIds = event.LookingForGroupUserIDs
	concert.InterestedUsers = event.InterestedUsers
	concert.GoingUsers = event.GoingUsers
	concert.LookingForGroupUsers = event.LookingForGroupUsers

	if event.MyInterest != nil {
		concert.MyInterest = event.MyInterest
	}
	return concert
}

// GetChicagoEvents retrieves Chicago area events with search and pagination.
// Parameter order must match generated.ConcertsAPIServicer: (ctx, artistName, limit, offset, anyInterest, sortByRelevancy)
func (s *ConcertAPIService) GetChicagoEvents(ctx context.Context, artistName string, limit int32, offset int32, anyInterest bool, sortByRelevancy bool) (generated.ImplResponse, error) {
	// Convert empty string to nil pointer for optional parameter
	var artistNamePtr *string
	if artistName != "" {
		artistNamePtr = &artistName
	}

	events, totalCount, err := s.concertService.GetChicagoEvents(ctx, artistNamePtr, anyInterest, sortByRelevancy, limit, offset)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: fmt.Sprintf("Failed to get Chicago events: %v", err),
		}), nil
	}

	// Convert events to API format with per-user interest derivation
	userID := getUserIDFromContext(ctx)
	apiEvents := make([]generated.Concert, 0, len(events))
	for _, event := range events {
		if userID != "" {
			setMyInterest(userID, event)
		}
		apiEvents = append(apiEvents, s.convertToAPIConcert(*event))
	}

	// Calculate hasMore flag
	hasMore := int64(offset+limit) < totalCount

	result := generated.ChicagoEventsResult{
		Events:     apiEvents,
		HasMore:    hasMore,
		TotalCount: int32(totalCount),
	}

	return generated.Response(http.StatusOK, result), nil
}

// GetChicagoEventById retrieves a single Chicago event by ID with interest enrichment
func (s *ConcertAPIService) GetChicagoEventById(ctx context.Context, eventId string) (generated.ImplResponse, error) {
	if eventId == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "eventId required"}), nil
	}
	if s.concertService == nil || s.concertService.repository == nil {
		return generated.Response(http.StatusServiceUnavailable, generated.ErrorResponse{Message: "concert repository unavailable"}), nil
	}
	evt, err := s.concertService.repository.GetChicagoEventByID(ctx, eventId)
	if err != nil {
		if err == concert.ErrEventNotFound {
			return generated.Response(http.StatusNotFound, generated.ErrorResponse{Message: "event not found"}), nil
		}
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: fmt.Sprintf("failed to fetch event: %v", err)}), nil
	}
	userID := getUserIDFromContext(ctx)
	if userID != "" {
		setMyInterest(userID, evt)
	}
	api := s.convertToAPIConcert(*evt)
	return generated.Response(http.StatusOK, api), nil
}

// SetEventInterest sets/updates the authenticated user's interest for an event
func (s *ConcertAPIService) SetEventInterest(ctx context.Context, eventId string, _ string, request generated.SetEventInterestRequest) (generated.ImplResponse, error) {
	// Derive user from JWT middleware context; ignore legacy header param for identity; second param deprecated
	userID := getUserIDFromContext(ctx)
	if userID == "" {
		logger.FromCtx(ctx).Warn("interest unauthorized", "eventId", eventId, "op", "set")
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{Message: "authentication required"}), nil
	}
	if s.concertService == nil || s.concertService.repository == nil {
		return generated.Response(http.StatusServiceUnavailable, generated.ErrorResponse{Message: "concert repository unavailable"}), nil
	}
	if eventId == "" || request.InterestType == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "eventId and interestType required"}), nil
	}
	status := string(request.InterestType)
	switch status {
	case "INTERESTED", "GOING", "LOOKING_FOR_GROUP":
	default:
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "invalid status"}), nil
	}
	if err := s.concertService.repository.UpsertUserInterest(ctx, userID, eventId, status); err != nil {
		logger.FromCtx(ctx).Error("interest set failed", "userId", userID, "eventId", eventId, "status", status, "error", err)
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: fmt.Sprintf("failed to set interest: %v", err)}), nil
	}
	logger.FromCtx(ctx).Info("interest set", "userId", userID, "eventId", eventId, "status", status)
	return generated.Response(http.StatusNoContent, nil), nil
}

// RemoveEventInterest removes the authenticated user's interest for an event
func (s *ConcertAPIService) RemoveEventInterest(ctx context.Context, eventId string, _ string) (generated.ImplResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == "" {
		logger.FromCtx(ctx).Warn("interest unauthorized", "eventId", eventId, "op", "remove")
		return generated.Response(http.StatusUnauthorized, generated.ErrorResponse{Message: "authentication required"}), nil
	}
	if s.concertService == nil || s.concertService.repository == nil {
		return generated.Response(http.StatusServiceUnavailable, generated.ErrorResponse{Message: "concert repository unavailable"}), nil
	}
	if eventId == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "eventId required"}), nil
	}
	if err := s.concertService.repository.RemoveUserInterest(ctx, userID, eventId); err != nil {
		logger.FromCtx(ctx).Error("interest remove failed", "userId", userID, "eventId", eventId, "error", err)
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: fmt.Sprintf("failed to remove interest: %v", err)}), nil
	}
	logger.FromCtx(ctx).Info("interest removed", "userId", userID, "eventId", eventId)
	return generated.Response(http.StatusNoContent, nil), nil
}
