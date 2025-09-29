package concert

import (
	"context"
	"fmt"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	database "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines the interface for concert event database operations
type Repository interface {
	UpsertEvent(ctx context.Context, event *Event) error
	DeleteOldEvents(ctx context.Context, cutoffDate time.Time) error
	GetEventCount(ctx context.Context) (int64, error)
	IsHealthy(ctx context.Context) error
}

// PostgreSQLRepository implements Repository using SQLC and pgx
type PostgreSQLRepository struct {
	db      *pgxpool.Pool
	queries *database.Queries
}

// NewPostgreSQLRepository creates a new PostgreSQL repository
func NewPostgreSQLRepository(cfg *config.DatabaseConfig) (*PostgreSQLRepository, error) {
	// Create connection pool
	pool, err := pgxpool.New(context.Background(), cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgreSQLRepository{
		db:      pool,
		queries: database.New(pool),
	}, nil
}

// UpsertEvent inserts or updates a concert event
func (r *PostgreSQLRepository) UpsertEvent(ctx context.Context, event *Event) error {
	// First, upsert the venue
	venueParams := database.UpsertVenueParams{
		ID:       event.Venue.ID,
		Name:     event.Venue.Name,
		Street:   pgtype.Text{String: event.Venue.Address.Street, Valid: event.Venue.Address.Street != ""},
		City:     event.Venue.Address.City,
		State:    pgtype.Text{String: event.Venue.Address.State, Valid: event.Venue.Address.State != ""},
		Country:  event.Venue.Address.Country,
		Postal:   pgtype.Text{String: event.Venue.Address.Postal, Valid: event.Venue.Address.Postal != ""},
		Capacity: pgtype.Int4{Int32: int32(event.Venue.Capacity), Valid: event.Venue.Capacity > 0},
	}

	venue, err := r.queries.UpsertVenue(ctx, venueParams)
	if err != nil {
		return fmt.Errorf("failed to upsert venue: %w", err)
	}

	// Upsert the concert event
	eventParams := database.UpsertConcertEventParams{
		ID:             event.ID,
		Name:           event.Name,
		EventDate:      pgtype.Timestamp{Time: event.Date, Valid: true},
		VenueID:        pgtype.Text{String: venue.ID, Valid: true},
		Genres:         event.Genres,
		PriceMin:       pgtype.Numeric{},
		PriceMax:       pgtype.Numeric{},
		PriceCurrency:  pgtype.Text{String: event.PriceRange.Currency, Valid: event.PriceRange.Currency != ""},
		TicketUrl:      pgtype.Text{String: event.TicketURL, Valid: event.TicketURL != ""},
		Description:    pgtype.Text{String: event.Description, Valid: event.Description != ""},
		Status:         event.Status,
		AgeRestriction: pgtype.Text{String: event.AgeRestriction, Valid: event.AgeRestriction != ""},
		Provider:       "ticketmaster",
		ExternalUrl:    pgtype.Text{String: event.TicketURL, Valid: event.TicketURL != ""},
	}

	// Handle price range if provided (store as strings to avoid big.Int complexity)
	if event.PriceRange.Min > 0 {
		minStr := fmt.Sprintf("%.2f", event.PriceRange.Min)
		if err := eventParams.PriceMin.Scan(minStr); err == nil {
			eventParams.PriceMin.Valid = true
		}
	}
	if event.PriceRange.Max > 0 {
		maxStr := fmt.Sprintf("%.2f", event.PriceRange.Max)
		if err := eventParams.PriceMax.Scan(maxStr); err == nil {
			eventParams.PriceMax.Valid = true
		}
	}

	concertEvent, err := r.queries.UpsertConcertEvent(ctx, eventParams)
	if err != nil {
		return fmt.Errorf("failed to upsert concert event: %w", err)
	}

	// Upsert artists and associate with the event
	for _, artist := range event.Artists {
		// Upsert artist
		artistParams := database.UpsertConcertArtistParams{
			ID:     artist.ID,
			Name:   artist.Name,
			Genres: artist.Genres,
		}

		concertArtist, err := r.queries.UpsertConcertArtist(ctx, artistParams)
		if err != nil {
			return fmt.Errorf("failed to upsert artist %s: %w", artist.Name, err)
		}

		// Associate artist with event
		associateParams := database.UpsertEventArtistParams{
			EventID:  concertEvent.ID,
			ArtistID: concertArtist.ID,
			Role:     pgtype.Text{String: "performer", Valid: true},
		}

		if err := r.queries.UpsertEventArtist(ctx, associateParams); err != nil {
			return fmt.Errorf("failed to associate artist %s with event %s: %w", artist.Name, event.Name, err)
		}
	}

	return nil
}

// DeleteOldEvents removes events older than the cutoff date
func (r *PostgreSQLRepository) DeleteOldEvents(ctx context.Context, cutoffDate time.Time) error {
	cutoff := pgtype.Timestamp{Time: cutoffDate, Valid: true}

	if err := r.queries.DeleteOldConcertEvents(ctx, cutoff); err != nil {
		return fmt.Errorf("failed to delete old concert events: %w", err)
	}

	return nil
}

// GetEventCount returns the total number of events in the database
func (r *PostgreSQLRepository) GetEventCount(ctx context.Context) (int64, error) {
	count, err := r.queries.GetConcertEventCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get concert event count: %w", err)
	}

	return count, nil
}

// IsHealthy checks if the database connection is healthy
func (r *PostgreSQLRepository) IsHealthy(ctx context.Context) error {
	if err := r.db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}

// Close closes the database connection pool
func (r *PostgreSQLRepository) Close() {
	r.db.Close()
}

// MockRepository is a simple in-memory implementation for testing
type MockRepository struct {
	events map[string]*Event
}

// NewMockRepository creates a new mock repository
func NewMockRepository() Repository {
	return &MockRepository{
		events: make(map[string]*Event),
	}
}

func (m *MockRepository) UpsertEvent(ctx context.Context, event *Event) error {
	m.events[event.ID] = event
	return nil
}

func (m *MockRepository) DeleteOldEvents(ctx context.Context, cutoffDate time.Time) error {
	for id, event := range m.events {
		if event.Date.Before(cutoffDate) {
			delete(m.events, id)
		}
	}
	return nil
}

func (m *MockRepository) GetEventCount(ctx context.Context) (int64, error) {
	return int64(len(m.events)), nil
}

func (m *MockRepository) IsHealthy(ctx context.Context) error {
	return nil
}

// Common errors
var (
	ErrEventNotFound = fmt.Errorf("event not found")
)
