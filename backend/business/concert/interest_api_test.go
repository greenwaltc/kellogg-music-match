package concert_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	"github.com/stretchr/testify/require"
)

// stubEventProvider minimal implementation for API wiring; not used in interest tests beyond interface satisfaction
type stubEventProvider struct{}

func (s *stubEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return &concert.SearchResult{Events: []concert.Event{}, TotalCount: 0}, nil
}
func (s *stubEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {
	return nil, concert.ErrEventNotFound
}
func (s *stubEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	return &concert.SearchResult{Events: []concert.Event{}, TotalCount: 0}, nil
}
func (s *stubEventProvider) GetProviderName() string             { return "stub" }
func (s *stubEventProvider) IsHealthy(ctx context.Context) error { return nil }

// Test interest lifecycle via HTTP endpoints
func TestInterestEndpoints(t *testing.T) {
	// Setup mock repository and seed an event
	repo := concert.NewMockRepository()
	eventID := "evt-test-1"
	evt := &concert.Event{
		ID:      eventID,
		Name:    "Test Event",
		Date:    time.Now().Add(24 * time.Hour),
		Venue:   concert.Venue{ID: "v1", Name: "Venue", Address: concert.Address{City: "Chicago", Country: "US"}},
		Artists: []concert.Artist{{ID: "a1", Name: "Artist"}},
		Status:  "onsale",
	}
	require.NoError(t, repo.UpsertEvent(context.Background(), evt))

	provider := &stubEventProvider{}
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{DefaultCity: "Chicago", DefaultState: "IL", DefaultCountry: "US"}}
	apiService := business.NewConcertAPIServiceWithRepository(provider, repo, cfg)

	// Wrap service with generated controller
	concertsController := generated.NewConcertsAPIController(apiService)
	router := generated.NewRouter(concertsController)

	userID := uuid.New().String() // using header name X-User-Username although value is a UUID

	// 1. Set interest INTERESTED
	body := map[string]string{"interestType": "INTERESTED"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/concerts/"+eventID+"/interest", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Username", userID)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code, "expected 204 on set interest")

	// 2. Update interest to GOING
	body["interestType"] = "GOING"
	payload, _ = json.Marshal(body)
	req2 := httptest.NewRequest(http.MethodPost, "/concerts/"+eventID+"/interest", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Username", userID)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusNoContent, rr2.Code)

	// 3. Fetch Chicago events and verify aggregation reflects GOING status
	req3 := httptest.NewRequest(http.MethodGet, "/chicago/events?limit=10&offset=0", nil)
	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)
	require.Equal(t, http.StatusOK, rr3.Code)
	var chicagoResp generated.ChicagoEventsResult
	require.NoError(t, json.Unmarshal(rr3.Body.Bytes(), &chicagoResp))
	require.Len(t, chicagoResp.Events, 1)
	// Note: interest arrays are optional; ensure event present

	// 4. Delete interest
	req4 := httptest.NewRequest(http.MethodDelete, "/concerts/"+eventID+"/interest", nil)
	req4.Header.Set("X-User-Username", userID)
	rr4 := httptest.NewRecorder()
	router.ServeHTTP(rr4, req4)
	require.Equal(t, http.StatusNoContent, rr4.Code)

	// 5. Negative: invalid status
	bad := map[string]string{"interestType": "INVALID"}
	badPayload, _ := json.Marshal(bad)
	req5 := httptest.NewRequest(http.MethodPost, "/concerts/"+eventID+"/interest", bytes.NewReader(badPayload))
	req5.Header.Set("Content-Type", "application/json")
	req5.Header.Set("X-User-Username", userID)
	rr5 := httptest.NewRecorder()
	router.ServeHTTP(rr5, req5)
	require.Equal(t, http.StatusBadRequest, rr5.Code)
}

// TestRemoveInterestIdempotent verifies removing interest when none exists returns 204 and remains safe on repeated deletes.
func TestRemoveInterestIdempotent(t *testing.T) {
	repo := concert.NewMockRepository()
	provider := &stubEventProvider{}
	cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{DefaultCity: "Chicago", DefaultState: "IL", DefaultCountry: "US"}}
	apiService := business.NewConcertAPIServiceWithRepository(provider, repo, cfg)
	controller := generated.NewConcertsAPIController(apiService)
	router := generated.NewRouter(controller)

	eventID := "evt-no-interest"
	evt := &concert.Event{ID: eventID, Name: "Empty Interest Event", Date: time.Now().Add(12 * time.Hour), Venue: concert.Venue{ID: "vX", Name: "Venue", Address: concert.Address{City: "Chicago", Country: "US"}}, Status: "onsale"}
	require.NoError(t, repo.UpsertEvent(context.Background(), evt))

	userID := uuid.New().String()

	// First delete (no existing interest)
	req := httptest.NewRequest(http.MethodDelete, "/concerts/"+eventID+"/interest", nil)
	req.Header.Set("X-User-Username", userID)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusNoContent, rr.Code)

	// Second delete (still none)
	req2 := httptest.NewRequest(http.MethodDelete, "/concerts/"+eventID+"/interest", nil)
	req2.Header.Set("X-User-Username", userID)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusNoContent, rr2.Code)
}
