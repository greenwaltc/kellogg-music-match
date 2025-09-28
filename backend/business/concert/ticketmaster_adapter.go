package concert

import (
	"context"
	"fmt"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// TicketmasterAdapter implements the EventProvider interface for Ticketmaster API
type TicketmasterAdapter struct {
	proxy  *TicketmasterProxy
	config *config.TicketmasterConfig
}

// NewTicketmasterAdapter creates a new Ticketmaster adapter
func NewTicketmasterAdapter(cfg *config.TicketmasterConfig) *TicketmasterAdapter {
	proxy := NewTicketmasterProxy(cfg)
	return &TicketmasterAdapter{
		proxy:  proxy,
		config: cfg,
	}
}

// SearchEvents implements EventProvider.SearchEvents
func (a *TicketmasterAdapter) SearchEvents(ctx context.Context, criteria SearchCriteria) (*SearchResult, error) {
	// If artist is specified, use the artist-specific search
	if criteria.Artist != "" {
		return a.GetEventsForArtist(ctx, criteria.Artist, criteria)
	}

	// Otherwise, use general concert search
	response, err := a.proxy.FetchConcerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch concerts from Ticketmaster: %w", err)
	}

	return a.convertTicketmasterResponse(response), nil
}

// GetEventByID implements EventProvider.GetEventByID
func (a *TicketmasterAdapter) GetEventByID(ctx context.Context, id string) (*Event, error) {
	// Note: This would require implementing a GetEventByID method in TicketmasterProxy
	// For now, we'll return an error indicating it's not implemented
	return nil, fmt.Errorf("GetEventByID not yet implemented for Ticketmaster provider")
}

// GetEventsForArtist implements EventProvider.GetEventsForArtist
func (a *TicketmasterAdapter) GetEventsForArtist(ctx context.Context, artistName string, criteria SearchCriteria) (*SearchResult, error) {
	response, err := a.proxy.FetchConcertsByArtist(ctx, artistName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch concerts for artist %s: %w", artistName, err)
	}

	return a.convertTicketmasterResponse(response), nil
}

// GetProviderName implements EventProvider.GetProviderName
func (a *TicketmasterAdapter) GetProviderName() string {
	return "Ticketmaster"
}

// IsHealthy implements EventProvider.IsHealthy
func (a *TicketmasterAdapter) IsHealthy(ctx context.Context) error {
	if a.config.ConsumerKey == "" {
		return fmt.Errorf("Ticketmaster consumer key not configured")
	}
	if a.config.ConsumerSecret == "" {
		return fmt.Errorf("Ticketmaster consumer secret not configured")
	}
	if a.config.BaseURL == "" {
		return fmt.Errorf("Ticketmaster base URL not configured")
	}

	// Could add an actual API health check here in the future
	return nil
}

// convertTicketmasterResponse converts Ticketmaster API response to our common Event model
func (a *TicketmasterAdapter) convertTicketmasterResponse(response *TicketmasterResponse) *SearchResult {
	events := make([]Event, 0, len(response.Embedded.Events))

	for _, tmEvent := range response.Embedded.Events {
		event := a.convertTicketmasterEvent(tmEvent)
		events = append(events, event)
	}

	return &SearchResult{
		Events:      events,
		TotalCount:  response.Page.TotalElements,
		CurrentPage: response.Page.Number,
		TotalPages:  response.Page.TotalPages,
		HasMore:     response.Page.Number < response.Page.TotalPages-1,
	}
}

// convertTicketmasterEvent converts a single Ticketmaster event to our Event model
func (a *TicketmasterAdapter) convertTicketmasterEvent(tmEvent TicketmasterEvent) Event {
	event := Event{
		ID:        tmEvent.ID,
		Name:      tmEvent.Name,
		Date:      a.parseEventDate(tmEvent),
		Venue:     a.convertVenue(tmEvent),
		Artists:   a.convertArtists(tmEvent),
		Genres:    a.extractGenres(tmEvent),
		TicketURL: tmEvent.URL,
	}

	// Convert price range if available
	if len(tmEvent.PriceRanges) > 0 {
		event.PriceRange = PriceRange{
			Min:      tmEvent.PriceRanges[0].Min,
			Max:      tmEvent.PriceRanges[0].Max,
			Currency: "USD", // Ticketmaster typically uses USD
		}
	}

	return event
}

// parseEventDate parses the Ticketmaster date format
func (a *TicketmasterAdapter) parseEventDate(tmEvent TicketmasterEvent) time.Time {
	dateStr := tmEvent.Dates.Start.LocalDate
	timeStr := tmEvent.Dates.Start.LocalTime

	// Combine date and time if both are available
	var datetimeStr string
	if timeStr != "" {
		datetimeStr = dateStr + "T" + timeStr
	} else {
		datetimeStr = dateStr + "T00:00:00"
	}

	// Try to parse the datetime
	if parsedTime, err := time.Parse("2006-01-02T15:04:05", datetimeStr); err == nil {
		return parsedTime
	}

	// Fallback to just the date
	if parsedDate, err := time.Parse("2006-01-02", dateStr); err == nil {
		return parsedDate
	}

	// If all parsing fails, return current time
	return time.Now()
}

// convertVenue converts Ticketmaster venue to our Venue model
func (a *TicketmasterAdapter) convertVenue(tmEvent TicketmasterEvent) Venue {
	if len(tmEvent.Embedded.Venues) == 0 {
		return Venue{}
	}

	tmVenue := tmEvent.Embedded.Venues[0]
	return Venue{
		Name: tmVenue.Name,
		Address: Address{
			Street:  tmVenue.Address.Line1,
			City:    tmVenue.City.Name,
			Country: "US", // Ticketmaster typically uses US venues
		},
	}
}

// convertArtists converts Ticketmaster attractions to our Artist model
func (a *TicketmasterAdapter) convertArtists(tmEvent TicketmasterEvent) []Artist {
	artists := make([]Artist, 0, len(tmEvent.Embedded.Attractions))

	for _, attraction := range tmEvent.Embedded.Attractions {
		artist := Artist{
			Name: attraction.Name,
		}

		// Extract genres from classifications
		if len(attraction.Classifications) > 0 {
			classification := attraction.Classifications[0]
			if classification.Genre.Name != "" {
				artist.Genres = []string{classification.Genre.Name}
			}
		}

		artists = append(artists, artist)
	}

	return artists
}

// extractGenres extracts genre information from Ticketmaster event
func (a *TicketmasterAdapter) extractGenres(tmEvent TicketmasterEvent) []string {
	genreSet := make(map[string]bool)

	// Extract genres from attractions
	for _, attraction := range tmEvent.Embedded.Attractions {
		for _, classification := range attraction.Classifications {
			if classification.Genre.Name != "" {
				genreSet[classification.Genre.Name] = true
			}
			if classification.Segment.Name != "" && classification.Segment.Name != "Music" {
				genreSet[classification.Segment.Name] = true
			}
		}
	}

	// Convert map to slice
	genres := make([]string, 0, len(genreSet))
	for genre := range genreSet {
		genres = append(genres, genre)
	}

	return genres
}
