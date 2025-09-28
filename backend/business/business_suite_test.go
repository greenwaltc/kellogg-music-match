package business_test

import (
	"context"
	"testing"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBusiness(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Business Suite")
}

// MockEventProvider implements concert.EventProvider for testing
type MockEventProvider struct {
	events []concert.Event
	errors map[string]error
}

func (m *MockEventProvider) SearchEvents(ctx context.Context, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if err, exists := m.errors["search"]; exists {
		return nil, err
	}

	return &concert.SearchResult{
		Events:      m.events,
		TotalCount:  len(m.events),
		CurrentPage: 0,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) GetEventByID(ctx context.Context, id string) (*concert.Event, error) {
	if err, exists := m.errors["getById"]; exists {
		return nil, err
	}

	for _, event := range m.events {
		if event.ID == id {
			return &event, nil
		}
	}

	return nil, nil
}

func (m *MockEventProvider) GetEventsForArtist(ctx context.Context, artistName string, criteria concert.SearchCriteria) (*concert.SearchResult, error) {
	if err, exists := m.errors["getByArtist"]; exists {
		return nil, err
	}

	var filteredEvents []concert.Event
	for _, event := range m.events {
		for _, artist := range event.Artists {
			if artist.Name == artistName {
				filteredEvents = append(filteredEvents, event)
				break
			}
		}
	}

	return &concert.SearchResult{
		Events:      filteredEvents,
		TotalCount:  len(filteredEvents),
		CurrentPage: 0,
		TotalPages:  1,
		HasMore:     false,
	}, nil
}

func (m *MockEventProvider) IsHealthy(ctx context.Context) error {
	if err, exists := m.errors["health"]; exists {
		return err
	}
	return nil
}

func (m *MockEventProvider) GetProviderName() string {
	return "MockProvider"
}

func (m *MockEventProvider) SetEvents(events []concert.Event) {
	m.events = events
}

func (m *MockEventProvider) SetError(operation string, err error) {
	if m.errors == nil {
		m.errors = make(map[string]error)
	}
	m.errors[operation] = err
}

func (m *MockEventProvider) AddEvent(event concert.Event) {
	m.events = append(m.events, event)
}
