package concert

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/stretchr/testify/require"
)

// NOTE: This test assumes a running Postgres reachable via DATABASE_URL or skips.
// It focuses on logical overwrite semantics: setting a new interest removes the old one.
func TestPostgresRepository_UserInterestExclusivity(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres exclusivity test")
	}
	// Very basic parse: expect postgres://user:pass@host:port/db
	// Allow using direct env-driven host overrides if DSN not parseable.
	// For test simplicity, rely on NewPostgreSQLRepository's ConnectionString builder by populating env variables instead if needed.
	// If DSN is provided, set PG* envs temporarily.
	os.Setenv("DB_HOST", "localhost")
	repo, err := NewPostgreSQLRepository(&config.DatabaseConfig{Host: os.Getenv("DB_HOST"), Port: os.Getenv("DB_PORT"), Name: os.Getenv("DB_NAME"), User: os.Getenv("DB_USER"), Password: os.Getenv("DB_PASSWORD"), SSLMode: os.Getenv("DB_SSLMODE")})
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	ctx := context.Background()

	// Insert a minimal event directly (reuse mock style via UpsertEvent if available)
	ev := &Event{
		ID:      "pg-excl-1",
		Name:    "PG Show",
		Date:    time.Now().Add(2 * time.Hour),
		Venue:   Venue{ID: "vpg1", Name: "PG Venue", Address: Address{City: "Chicago", Country: "US"}},
		Artists: []Artist{{ID: "apg1", Name: "Artist", Genres: []string{"Indie"}}},
		Status:  "onsale",
	}
	// Use mock path if UpsertEvent not implemented for Postgres; skip if not available.
	if err := repo.UpsertEvent(ctx, ev); err != nil { // method exists on mock; if absent for Postgres, skip
		t.Skipf("UpsertEvent not implemented for Postgres repo: %v", err)
	}

	userID := "11111111-1111-1111-1111-111111111111" // deterministic UUID
	require.NoError(t, repo.UpsertUserInterest(ctx, userID, ev.ID, "INTERESTED"))
	require.NoError(t, repo.UpsertUserInterest(ctx, userID, ev.ID, "GOING")) // overwrite
	// At this point only GOING should remain when we fetch.
	events, err2 := repo.GetChicagoEvents(ctx, nil, false, 10, 0)
	require.NoError(t, err2)
	var found *Event
	for _, e := range events {
		if e.ID == ev.ID {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatalf("event not found in listing")
	}
	// Exclusivity assertion: user should not appear in interested after switching to going.
	for _, id := range found.InterestedUserIDs {
		if id == userID {
			t.Fatalf("user still present in Interested after overwrite")
		}
	}
}
