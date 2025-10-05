package concert

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	GetChicagoEvents(ctx context.Context, artistName *string, anyInterest bool, sortByRelevancy bool, limit int32, offset int32) ([]*Event, error)
	GetChicagoEventsCount(ctx context.Context, artistName *string, anyInterest bool) (int64, error)
	GetChicagoEventByID(ctx context.Context, id string) (*Event, error)
	// User interest operations
	UpsertUserInterest(ctx context.Context, userID string, eventID string, status string) error
	RemoveUserInterest(ctx context.Context, userID string, eventID string) error
}

// PostgreSQLRepository implements Repository using SQLC and pgx
type PostgreSQLRepository struct {
	db      *pgxpool.Pool
	queries *database.Queries
	// simple in-memory cache for userID -> full name
	userNameCache   map[string]string
	userNameCacheMu sync.RWMutex
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
		db:            pool,
		queries:       database.New(pool),
		userNameCache: make(map[string]string),
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
	// Pass empty interest status filter (means no filtering)
	count, err := r.queries.GetConcertEventCount(ctx, "")
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

// GetChicagoEvents retrieves Chicago area events with optional artist filtering and pagination
func (r *PostgreSQLRepository) GetChicagoEvents(ctx context.Context, artistName *string, anyInterest bool, sortByRelevancy bool, limit int32, offset int32) ([]*Event, error) {
	artistNameParam := ""
	if artistName != nil {
		artistNameParam = *artistName
	}

	params := database.GetChicagoEventsWithArtistSearchParams{
		ArtistName:      artistNameParam,
		AnyInterest:     anyInterest,
		SortByRelevancy: sortByRelevancy,
		LimitCount:      limit,
		OffsetCount:     offset,
	}

	rows, err := r.queries.GetChicagoEventsWithArtistSearch(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get Chicago events: %w", err)
	}

	events := make([]*Event, 0, len(rows))
	for _, row := range rows {
		event := &Event{
			ID:             row.ID,
			Name:           row.Name,
			Date:           row.EventDate.Time,
			Status:         row.Status,
			TicketURL:      row.TicketUrl.String,
			Description:    row.Description.String,
			AgeRestriction: row.AgeRestriction.String,
			Genres:         row.Genres,
			Relevancy: func() int {
				if row.Relevancy != nil {
					switch v := row.Relevancy.(type) {
					case int32:
						return int(v)
					case int64:
						return int(v)
					case int:
						return v
					case float64:
						return int(v)
					case float32:
						return int(v)
					case []byte:
						if iv, err := strconv.Atoi(string(v)); err == nil {
							return iv
						}
					}
				}
				return 0
			}(),
			Venue: Venue{
				ID:   row.VenueID.String,
				Name: row.VenueName.String,
				Address: Address{
					Street:  row.VenueStreet.String,
					City:    row.VenueCity.String,
					State:   row.VenueState.String,
					Country: row.VenueCountry.String,
					Postal:  row.VenuePostal.String,
				},
				Capacity: int(row.VenueCapacity.Int32),
			},
		}

		// Price range (convert numeric -> float)
		minPrice := numericToFloat(row.PriceMin)
		maxPrice := numericToFloat(row.PriceMax)
		if minPrice > 0 || maxPrice > 0 || row.PriceCurrency.String != "" {
			event.PriceRange = PriceRange{
				Min:      minPrice,
				Max:      maxPrice,
				Currency: row.PriceCurrency.String,
			}
		}

		// User interest buckets (already []string)
		event.InterestedUserIDs = row.InterestedUserIds
		event.GoingUserIDs = row.GoingUserIds
		event.LookingForGroupUserIDs = row.LookingForGroupUserIds

		// Enrich via helper (best-effort)
		_ = r.enrichEventUserNames(ctx, event)

		// Parse aggregated artists JSON
		if row.ArtistsJson != nil {
			parsedArtists, err := parseArtistsJSON(row.ArtistsJson)
			if err != nil {
				return nil, fmt.Errorf("failed to parse artists for event %s: %w", row.ID, err)
			}
			event.Artists = parsedArtists
		}

		events = append(events, event)
	}

	return events, nil
}

// GetChicagoEventsCount returns the total count of Chicago area events with optional artist filtering
func (r *PostgreSQLRepository) GetChicagoEventsCount(ctx context.Context, artistName *string, anyInterest bool) (int64, error) {
	artistNameParam := ""
	if artistName != nil {
		artistNameParam = *artistName
	}

	params := database.GetChicagoEventsCountWithArtistSearchParams{ArtistName: artistNameParam, InterestStatus: "", AnyInterest: anyInterest}
	count, err := r.queries.GetChicagoEventsCountWithArtistSearch(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("failed to get Chicago events count: %w", err)
	}

	return count, nil
}

// GetChicagoEventByID fetches a single Chicago event and enriches interest user names.
func (r *PostgreSQLRepository) GetChicagoEventByID(ctx context.Context, id string) (*Event, error) {
	if id == "" {
		return nil, fmt.Errorf("empty id")
	}
	// Reuse existing list query with limit 1 filtering by artistName empty then filter in code (simpler short-term)
	params := database.GetChicagoEventsWithArtistSearchParams{ArtistName: "", LimitCount: 1, SortByRelevancy: true, OffsetCount: 0}
	rows, err := r.queries.GetChicagoEventsWithArtistSearch(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	for _, row := range rows {
		if row.ID != id { // ensure match
			continue
		}
		// Build event
		evt := &Event{
			ID:             row.ID,
			Name:           row.Name,
			Date:           row.EventDate.Time,
			Status:         row.Status,
			TicketURL:      row.TicketUrl.String,
			Description:    row.Description.String,
			AgeRestriction: row.AgeRestriction.String,
			Genres:         row.Genres,
			Relevancy: func() int {
				if row.Relevancy != nil {
					switch v := row.Relevancy.(type) {
					case int32:
						return int(v)
					case int64:
						return int(v)
					case int:
						return v
					case float64:
						return int(v)
					case float32:
						return int(v)
					case []byte:
						if iv, err := strconv.Atoi(string(v)); err == nil {
							return iv
						}
					}
				}
				return 0
			}(),
			Venue: Venue{ID: row.VenueID.String, Name: row.VenueName.String, Address: Address{Street: row.VenueStreet.String, City: row.VenueCity.String, State: row.VenueState.String, Country: row.VenueCountry.String, Postal: row.VenuePostal.String}, Capacity: int(row.VenueCapacity.Int32)},
		}
		minPrice := numericToFloat(row.PriceMin)
		maxPrice := numericToFloat(row.PriceMax)
		if minPrice > 0 || maxPrice > 0 || row.PriceCurrency.String != "" {
			evt.PriceRange = PriceRange{Min: minPrice, Max: maxPrice, Currency: row.PriceCurrency.String}
		}
		evt.InterestedUserIDs = row.InterestedUserIds
		evt.GoingUserIDs = row.GoingUserIds
		evt.LookingForGroupUserIDs = row.LookingForGroupUserIds
		if row.ArtistsJson != nil {
			artists, _ := parseArtistsJSON(row.ArtistsJson)
			evt.Artists = artists
		}
		_ = r.enrichEventUserNames(ctx, evt)
		return evt, nil
	}
	return nil, ErrEventNotFound
}

// enrichEventUserNames populates the *Users slices using cached names; batched lookup for misses.
func (r *PostgreSQLRepository) enrichEventUserNames(ctx context.Context, evt *Event) error {
	if evt == nil {
		return nil
	}
	// Collect unique user IDs
	unique := make(map[string]struct{})
	add := func(ids []string) {
		for _, id := range ids {
			if id != "" {
				unique[id] = struct{}{}
			}
		}
	}
	add(evt.InterestedUserIDs)
	add(evt.GoingUserIDs)
	add(evt.LookingForGroupUserIDs)
	if len(unique) == 0 {
		return nil
	}
	// Determine which IDs are missing from cache
	missing := []uuid.UUID{}
	missingStr := []string{}
	r.userNameCacheMu.RLock()
	for id := range unique {
		if _, ok := r.userNameCache[id]; !ok {
			if parsed, err := uuid.Parse(id); err == nil {
				missing = append(missing, parsed)
				missingStr = append(missingStr, id)
			}
		}
	}
	r.userNameCacheMu.RUnlock()

	// Batch fetch missing (if any)
	if len(missing) > 0 {
		rows, err := r.db.Query(ctx, "SELECT id, first_name, last_name FROM users WHERE id = ANY($1)", missing)
		if err == nil {
			defer rows.Close()
			newCache := make(map[string]string)
			for rows.Next() {
				var id uuid.UUID
				var first, last string
				if scanErr := rows.Scan(&id, &first, &last); scanErr == nil {
					full := strings.TrimSpace(first + " " + last)
					if full != "" {
						newCache[id.String()] = full
					}
				}
			}
			if rows.Err() == nil {
				// Merge into cache
				if len(newCache) > 0 {
					r.userNameCacheMu.Lock()
					for k, v := range newCache {
						r.userNameCache[k] = v
					}
					r.userNameCacheMu.Unlock()
				}
			}
		}
		// Fallback: if batch failed, attempt individual queries (best-effort)
		if err != nil {
			for i, uuidVal := range missing {
				u, qErr := r.queries.GetUserByID(ctx, uuidVal)
				if qErr != nil {
					continue
				}
				full := strings.TrimSpace(u.FirstName + " " + u.LastName)
				if full != "" {
					r.userNameCacheMu.Lock()
					r.userNameCache[missingStr[i]] = full
					r.userNameCacheMu.Unlock()
				}
			}
		}
	}

	// Build name slices from cache
	build := func(ids []string) []string {
		out := []string{}
		r.userNameCacheMu.RLock()
		defer r.userNameCacheMu.RUnlock()
		for _, id := range ids {
			if name, ok := r.userNameCache[id]; ok {
				out = append(out, name)
			}
		}
		return out
	}
	evt.InterestedUsers = build(evt.InterestedUserIDs)
	evt.GoingUsers = build(evt.GoingUserIDs)
	evt.LookingForGroupUsers = build(evt.LookingForGroupUserIDs)
	return nil
}

// Close closes the database connection pool
func (r *PostgreSQLRepository) Close() {
	r.db.Close()
}

// MockRepository is a simple in-memory implementation for testing
type MockRepository struct {
	events    map[string]*Event
	interests map[string]map[string]string // eventID -> userID -> status
}

// NewMockRepository creates a new mock repository
func NewMockRepository() Repository {
	return &MockRepository{
		events:    make(map[string]*Event),
		interests: make(map[string]map[string]string),
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

func (m *MockRepository) GetChicagoEvents(ctx context.Context, artistName *string, anyInterest bool, sortByRelevancy bool, limit int32, offset int32) ([]*Event, error) {
	// Collect and sort Chicago events deterministically by date (asc) then ID
	var all []*Event
	for _, event := range m.events {
		// Simple Chicago filter
		if event.Venue.Address.City != "Chicago" {
			continue
		}

		// anyInterest filter: require at least one interested/going/lfg user
		if anyInterest {
			if len(event.InterestedUserIDs) == 0 && len(event.GoingUserIDs) == 0 && len(event.LookingForGroupUserIDs) == 0 {
				continue
			}
		}

		// Artist name filter (case-insensitive prefix match)
		if artistName != nil && *artistName != "" {
			query := strings.ToLower(*artistName)
			found := false
			for _, artist := range event.Artists {
				nameLower := strings.ToLower(artist.Name)
				if len(nameLower) >= len(query) && nameLower[:len(query)] == query {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		all = append(all, event)
	}
	// Sort
	sort.Slice(all, func(i, j int) bool {
		if all[i].Date.Equal(all[j].Date) {
			return all[i].ID < all[j].ID
		}
		return all[i].Date.Before(all[j].Date)
	})
	// Apply offset & limit
	var window []*Event
	for idx, e := range all {
		if int32(idx) < offset {
			continue
		}
		if int32(len(window)) >= limit {
			break
		}
		window = append(window, e)
	}
	return window, nil
}

func (m *MockRepository) GetChicagoEventsCount(ctx context.Context, artistName *string, anyInterest bool) (int64, error) {
	count := int64(0)
	for _, event := range m.events {
		// Simple Chicago filter
		if event.Venue.Address.City != "Chicago" {
			continue
		}
		if anyInterest {
			if len(event.InterestedUserIDs) == 0 && len(event.GoingUserIDs) == 0 && len(event.LookingForGroupUserIDs) == 0 {
				continue
			}
		}

		// Artist name filter
		if artistName != nil && *artistName != "" {
			found := false
			for _, artist := range event.Artists {
				if len(artist.Name) > 0 && len(*artistName) > 0 &&
					artist.Name[:min(len(artist.Name), len(*artistName))] == (*artistName)[:min(len(artist.Name), len(*artistName))] {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		count++
	}
	return count, nil
}

func (m *MockRepository) GetChicagoEventByID(ctx context.Context, id string) (*Event, error) {
	if evt, ok := m.events[id]; ok {
		return evt, nil
	}
	return nil, ErrEventNotFound
}

// UpsertUserInterest records or updates a user's interest for an event in the mock repo.
func (m *MockRepository) UpsertUserInterest(ctx context.Context, userID string, eventID string, status string) error {
	if _, ok := m.events[eventID]; !ok {
		return ErrEventNotFound
	}
	if m.interests[eventID] == nil {
		m.interests[eventID] = make(map[string]string)
	}
	m.interests[eventID][userID] = status
	// Reflect aggregate buckets in Event struct for retrieval
	evt := m.events[eventID]
	// Simple rebuild of buckets
	interested := []string{}
	going := []string{}
	lfg := []string{}
	for uid, st := range m.interests[eventID] {
		switch st {
		case "INTERESTED":
			interested = append(interested, uid)
		case "GOING":
			going = append(going, uid)
		case "LOOKING_FOR_GROUP":
			lfg = append(lfg, uid)
		}
	}
	evt.InterestedUserIDs = interested
	evt.GoingUserIDs = going
	evt.LookingForGroupUserIDs = lfg
	return nil
}

// RemoveUserInterest removes a user's interest in the mock repo.
func (m *MockRepository) RemoveUserInterest(ctx context.Context, userID string, eventID string) error {
	if m.interests[eventID] != nil {
		delete(m.interests[eventID], userID)
		if len(m.interests[eventID]) == 0 {
			delete(m.interests, eventID)
		}
	}
	// Recompute buckets
	if evt, ok := m.events[eventID]; ok {
		interested := []string{}
		going := []string{}
		lfg := []string{}
		for uid, st := range m.interests[eventID] {
			switch st {
			case "INTERESTED":
				interested = append(interested, uid)
			case "GOING":
				going = append(going, uid)
			case "LOOKING_FOR_GROUP":
				lfg = append(lfg, uid)
			}
		}
		evt.InterestedUserIDs = interested
		evt.GoingUserIDs = going
		evt.LookingForGroupUserIDs = lfg
	}
	return nil
}

// Postgres implementations
func (r *PostgreSQLRepository) UpsertUserInterest(ctx context.Context, userID string, eventID string, status string) error {
	// Basic validation of enum
	switch status {
	case "INTERESTED", "GOING", "LOOKING_FOR_GROUP":
	default:
		return fmt.Errorf("invalid interest status: %s", status)
	}
	params := database.UpsertUserConcertEventInterestParams{EventID: eventID, InterestStatus: status}
	// Accept plain string userID (UUID format expected) – rely on DB to validate; convert to uuid via pgx text -> use google/uuid parse
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid userID uuid: %w", err)
	}
	params.UserID = uid
	if err := r.queries.UpsertUserConcertEventInterest(ctx, params); err != nil {
		return fmt.Errorf("upsert user interest failed: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) RemoveUserInterest(ctx context.Context, userID string, eventID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid userID uuid: %w", err)
	}
	params := database.DeleteUserConcertEventInterestParams{UserID: uid, EventID: eventID}
	if err := r.queries.DeleteUserConcertEventInterest(ctx, params); err != nil {
		return fmt.Errorf("remove user interest failed: %w", err)
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// toStringSlice attempts to coerce various interface{} types returned by sqlc/pgx for text[] into a []string.
// sqlc used interface{} because it couldn't infer the array type for the aggregated user UUIDs.
// parseArtistsJSON converts the aggregated jsonb artists array into []Artist
func parseArtistsJSON(raw interface{}) ([]Artist, error) {
	if raw == nil {
		return nil, nil
	}
	// raw could be []uint8 or string depending on pgx jsonb decoding
	var bytes []byte
	switch v := raw.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		// Fallback: JSON marshal the interface
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		bytes = b
	}
	// Expect array of objects with id, name, genres
	var arr []struct {
		ID     string   `json:"id"`
		Name   string   `json:"name"`
		Genres []string `json:"genres"`
	}
	if err := json.Unmarshal(bytes, &arr); err != nil {
		return nil, err
	}
	artists := make([]Artist, 0, len(arr))
	for _, a := range arr {
		artists = append(artists, Artist{ID: a.ID, Name: a.Name, Genres: a.Genres})
	}
	return artists, nil
}

// numericToFloat converts a pgtype.Numeric to float64 best-effort.
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	// Convert through big.Rat using Scan/Value conversions. pgtype.Numeric implements Scan/Value for database/sql compatibility.
	// We attempt: 1) deserialize to string via fmt, 2) parse as float.
	// fmt.Sprintf("%v", n) will yield something like {NaN} for invalid, but we already checked Valid.
	s := fmt.Sprintf("%v", n)
	if s == "" || s == "<nil>" {
		return 0
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// Fallback: try big.Rat parse
	var rat big.Rat
	if _, ok := rat.SetString(s); ok {
		f, _ := rat.Float64()
		return f
	}
	return 0
}

// Common errors
var (
	ErrEventNotFound     = fmt.Errorf("event not found")
	ErrProviderUnhealthy = fmt.Errorf("provider is unhealthy")
	ErrRepositoryFailure = fmt.Errorf("repository operation failed")
)
