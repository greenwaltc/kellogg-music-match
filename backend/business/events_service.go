package business

import (
	"context"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
)

// EventSearchService performs on-demand searches using Ticketmaster and merges local overlays.
// For Phase 1 Step 1, we return Ticketmaster results only (includeAssociated handled later),
// ensuring no regression to legacy flows.
type EventSearchService struct {
	tm *concert.TicketmasterProxy
}

func NewEventSearchService(tm *concert.TicketmasterProxy) *EventSearchService {
	return &EventSearchService{tm: tm}
}

// SearchInput mirrors the OpenAPI query with safe types
type SearchInput struct {
	Keyword            string
	SegmentName        string
	ClassificationName string
	CountryCode        string
	StateCode          string
	City               string
	LatLong            string
	Radius             int
	StartDateTime      *time.Time
	EndDateTime        *time.Time
	Sort               string
	Size               int
	Page               int
	IncludeAssociated  bool
}

// Search executes a Ticketmaster Discovery search and returns the raw TicketmasterProxy response.
func (s *EventSearchService) Search(ctx context.Context, in SearchInput) (*concert.TicketmasterResponse, error) {
	var start, end string
	if in.StartDateTime != nil {
		start = in.StartDateTime.UTC().Format("2006-01-02T15:04:05Z")
	}
	if in.EndDateTime != nil {
		end = in.EndDateTime.UTC().Format("2006-01-02T15:04:05Z")
	}
	opts := concert.DiscoverySearchOptions{
		Keyword:            in.Keyword,
		SegmentName:        in.SegmentName,
		ClassificationName: in.ClassificationName,
		CountryCode:        in.CountryCode,
		StateCode:          in.StateCode,
		City:               in.City,
		LatLong:            in.LatLong,
		Radius:             in.Radius,
		StartDateTime:      start,
		EndDateTime:        end,
		Sort:               in.Sort,
		Size:               in.Size,
		Page:               in.Page,
	}
	return s.tm.SearchDiscovery(ctx, opts)
}
