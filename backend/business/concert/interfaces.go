package concert

import (
	"context"
	"time"
)

// Event represents a generic concert/event model
type Event struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Date           time.Time  `json:"date"`
	Venue          Venue      `json:"venue"`
	Artists        []Artist   `json:"artists"`
	Genres         []string   `json:"genres"`
	PriceRange     PriceRange `json:"priceRange,omitempty"`
	TicketURL      string     `json:"ticketUrl,omitempty"`
	Description    string     `json:"description,omitempty"`
	Status         string     `json:"status"` // e.g., "onsale", "offsale", "cancelled"
	AgeRestriction string     `json:"ageRestriction,omitempty"`
	// Aggregated user interest buckets (UUIDs as strings)
	InterestedUserIDs      []string `json:"interestedUserIds,omitempty"`
	GoingUserIDs           []string `json:"goingUserIds,omitempty"`
	LookingForGroupUserIDs []string `json:"lookingForGroupUserIds,omitempty"`
}

// Venue represents a concert venue
type Venue struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Address  Address `json:"address"`
	Capacity int     `json:"capacity,omitempty"`
}

// Address represents a physical address
type Address struct {
	Street  string `json:"street,omitempty"`
	City    string `json:"city"`
	State   string `json:"state,omitempty"`
	Country string `json:"country"`
	Postal  string `json:"postal,omitempty"`
}

// Artist represents an artist/performer
type Artist struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Genres []string `json:"genres,omitempty"`
}

// PriceRange represents ticket price information
type PriceRange struct {
	Min      float64 `json:"min,omitempty"`
	Max      float64 `json:"max,omitempty"`
	Currency string  `json:"currency"`
}

// SearchCriteria represents search parameters for events
type SearchCriteria struct {
	Artist     string    `json:"artist,omitempty"`
	City       string    `json:"city,omitempty"`
	State      string    `json:"state,omitempty"`
	Country    string    `json:"country,omitempty"`
	Genre      string    `json:"genre,omitempty"`
	StartDate  time.Time `json:"startDate,omitempty"`
	EndDate    time.Time `json:"endDate,omitempty"`
	MaxResults int       `json:"maxResults,omitempty"`
	Page       int       `json:"page,omitempty"`
}

// SearchResult represents paginated search results
type SearchResult struct {
	Events      []Event `json:"events"`
	TotalCount  int     `json:"totalCount"`
	CurrentPage int     `json:"currentPage"`
	TotalPages  int     `json:"totalPages"`
	HasMore     bool    `json:"hasMore"`
}

// EventProvider defines the interface that all concert API providers must implement
type EventProvider interface {
	// SearchEvents searches for events based on criteria
	SearchEvents(ctx context.Context, criteria SearchCriteria) (*SearchResult, error)

	// GetEventByID retrieves a specific event by its ID
	GetEventByID(ctx context.Context, id string) (*Event, error)

	// GetEventsForArtist retrieves events for a specific artist
	GetEventsForArtist(ctx context.Context, artistName string, criteria SearchCriteria) (*SearchResult, error)

	// GetProviderName returns the name of the provider (e.g., "Ticketmaster", "Eventbrite")
	GetProviderName() string

	// IsHealthy checks if the provider is available
	IsHealthy(ctx context.Context) error
}

// EventService defines the business logic interface for events
type EventService interface {
	// GetUpcomingEvents retrieves upcoming events based on criteria
	GetUpcomingEvents(ctx context.Context, criteria SearchCriteria) (*SearchResult, error)

	// GetEventsForUser retrieves events based on user's preferences/artists
	GetEventsForUser(ctx context.Context, userID string) (*SearchResult, error)

	// GetEventDetails retrieves detailed information about a specific event
	GetEventDetails(ctx context.Context, eventID string) (*Event, error)

	// SearchEventsByArtist searches for events by artist name
	SearchEventsByArtist(ctx context.Context, artistName string) (*SearchResult, error)
}
