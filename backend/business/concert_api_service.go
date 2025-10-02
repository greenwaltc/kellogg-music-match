package business

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

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
		Id:      event.ID,
		Name:    event.Name,
		Date:    event.Date,
		Venue:   apiVenue,
		Artists: apiArtists,
		Genres:  event.Genres,
	}

	// Include user interest aggregates if present
	if len(event.InterestedUserIDs) > 0 {
		concert.InterestedUserIds = event.InterestedUserIDs
	}
	if len(event.GoingUserIDs) > 0 {
		concert.GoingUserIds = event.GoingUserIDs
	}
	if len(event.LookingForGroupUserIDs) > 0 {
		concert.LookingForGroupUserIds = event.LookingForGroupUserIDs
	}

	// Only include price range if it has values
	if hasPriceRange {
		concert.PriceRange = apiPriceRange
	}

	// Only include ticket URL if it's not empty
	if event.TicketURL != "" {
		concert.TicketUrl = event.TicketURL
	}

	return concert
}

// GetChicagoEvents retrieves Chicago area events with search and pagination
func (s *ConcertAPIService) GetChicagoEvents(ctx context.Context, artistName string, limit int32, offset int32) (generated.ImplResponse, error) {
	// Convert empty string to nil pointer for optional parameter
	var artistNamePtr *string
	if artistName != "" {
		artistNamePtr = &artistName
	}

	events, totalCount, err := s.concertService.GetChicagoEvents(ctx, artistNamePtr, limit, offset)
	if err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{
			Message: fmt.Sprintf("Failed to get Chicago events: %v", err),
		}), nil
	}

	// Convert events to API format
	apiEvents := make([]generated.Concert, 0, len(events))
	for _, event := range events {
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

// SetEventInterest sets/updates the authenticated user's interest for an event
func (s *ConcertAPIService) SetEventInterest(ctx context.Context, eventId string, xUserUsername string, request generated.SetEventInterestRequest) (generated.ImplResponse, error) {
	if s.concertService == nil || s.concertService.repository == nil {
		return generated.Response(http.StatusServiceUnavailable, generated.ErrorResponse{Message: "concert repository unavailable"}), nil
	}
	if eventId == "" || request.InterestType == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "eventId and interestType required"}), nil
	}
	status := string(request.InterestType)
	// Validate
	switch status {
	case "INTERESTED", "GOING", "LOOKING_FOR_GROUP":
	default:
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "invalid status"}), nil
	}
	if err := s.concertService.repository.UpsertUserInterest(ctx, xUserUsername, eventId, status); err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: fmt.Sprintf("failed to set interest: %v", err)}), nil
	}
	return generated.Response(http.StatusNoContent, nil), nil
}

// RemoveEventInterest removes the authenticated user's interest for an event
func (s *ConcertAPIService) RemoveEventInterest(ctx context.Context, eventId string, xUserUsername string) (generated.ImplResponse, error) {
	if s.concertService == nil || s.concertService.repository == nil {
		return generated.Response(http.StatusServiceUnavailable, generated.ErrorResponse{Message: "concert repository unavailable"}), nil
	}
	if eventId == "" {
		return generated.Response(http.StatusBadRequest, generated.ErrorResponse{Message: "eventId required"}), nil
	}
	if err := s.concertService.repository.RemoveUserInterest(ctx, xUserUsername, eventId); err != nil {
		return generated.Response(http.StatusInternalServerError, generated.ErrorResponse{Message: fmt.Sprintf("failed to remove interest: %v", err)}), nil
	}
	return generated.Response(http.StatusNoContent, nil), nil
}
