package business

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// ConcertService handles concert-related business logic using provider abstraction
type ConcertService struct {
	provider   concert.EventProvider
	repository concert.Repository
	config     *config.Config
}

// NewConcertService creates a new concert service with a configurable provider
func NewConcertService(cfg *config.Config) *ConcertService {
	// For now, default to Ticketmaster adapter
	// In the future, this could be configurable via environment variable
	provider := concert.NewTicketmasterAdapter(&cfg.Ticketmaster)

	return &ConcertService{
		provider: provider,
		config:   cfg,
	}
}

// NewConcertServiceWithRepository creates a concert service with both provider and repository
func NewConcertServiceWithRepository(provider concert.EventProvider, repository concert.Repository, cfg *config.Config) *ConcertService {
	return &ConcertService{
		provider:   provider,
		repository: repository,
		config:     cfg,
	}
}

// NewConcertServiceWithProvider creates a concert service with a specific provider
func NewConcertServiceWithProvider(provider concert.EventProvider, cfg *config.Config) *ConcertService {
	return &ConcertService{
		provider: provider,
		config:   cfg,
	}
}

// GetUpcomingEvents retrieves upcoming events based on search criteria
func (s *ConcertService) GetUpcomingEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	// Set default location from config if not specified
	if criteria.City == "" {
		criteria.City = s.config.Ticketmaster.DefaultCity
	}
	if criteria.State == "" {
		criteria.State = s.config.Ticketmaster.DefaultState
	}
	if criteria.Country == "" {
		criteria.Country = s.config.Ticketmaster.DefaultCountry
	}

	return s.provider.SearchEvents(ctx, criteria)
}

// GetEventsForUser retrieves events based on user's favorite artists
func (s *ConcertService) GetEventsForUser(ctx context.Context, userID string) (*concert.SearchResult, error) {
	// TODO: This would integrate with your existing user service to get their favorite artists
	// For now, return an empty result

	// Example implementation would be:
	// 1. Get user's favorite artists from UserService
	// 2. Create search criteria for each artist
	// 3. Aggregate results from multiple searches
	// 4. Deduplicate and sort by date

	return &concert.SearchResult{
		Events:      []concert.Event{},
		TotalCount:  0,
		CurrentPage: 0,
		TotalPages:  0,
		HasMore:     false,
	}, nil
}

// GetEventDetails retrieves detailed information about a specific event
func (s *ConcertService) GetEventDetails(ctx context.Context, eventID string) (*concert.Event, error) {
	return s.provider.GetEventByID(ctx, eventID)
}

// SearchEventsByArtist searches for events by artist name
func (s *ConcertService) SearchEventsByArtist(ctx context.Context, artistName string) (*concert.SearchResult, error) {
	if artistName == "" {
		return nil, fmt.Errorf("artist name cannot be empty")
	}

	criteria := concert.SearchCriteria{
		Artist:  artistName,
		City:    s.config.Ticketmaster.DefaultCity,
		State:   s.config.Ticketmaster.DefaultState,
		Country: s.config.Ticketmaster.DefaultCountry,
	}

	return s.provider.GetEventsForArtist(ctx, artistName, criteria)
}

// GetProviderName returns the name of the current event provider
func (s *ConcertService) GetProviderName() string {
	return s.provider.GetProviderName()
}

// ValidateConfiguration checks if the current provider configuration is valid
func (s *ConcertService) ValidateConfiguration(ctx context.Context) error {
	return s.provider.IsHealthy(ctx)
}

// GetChicagoEvents retrieves Chicago area events from the local database with search and pagination
func (s *ConcertService) GetChicagoEvents(ctx context.Context, artistName *string, anyInterest bool, limit int32, offset int32, onlyMyTopArtists bool, anchorUserID *uuid.UUID, topN *int32) ([]*concert.Event, int64, error) {
	if s.repository == nil {
		return nil, 0, fmt.Errorf("repository not available")
	}

	events, err := s.repository.GetChicagoEvents(ctx, artistName, anyInterest, limit, offset, onlyMyTopArtists, anchorUserID, topN)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get Chicago events: %w", err)
	}

	count, err := s.repository.GetChicagoEventsCount(ctx, artistName, anyInterest, onlyMyTopArtists, anchorUserID, topN)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get Chicago events count: %w", err)
	}

	return events, count, nil
}
