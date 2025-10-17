package business_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	business "github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/business/concert"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"github.com/greenwaltc/kellogg-music-match/backend/generated"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventsAdapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events Adapter Suite")
}

var _ = Describe("GeneratedEventsAdapter", func() {
	It("returns 404-like result when on-demand is disabled", func() {
		cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{OnDemand: false}}
		// Service can be nil because we won't call it when disabled
		adapter := business.NewGeneratedEventsAdapter(business.NewEventSearchService(concert.NewTicketmasterProxy(&cfg.Ticketmaster)), cfg)
		// Call with minimal params
		resp, err := adapter.SearchEvents(context.Background(), "", "", "", "", "", "", "", 0, time.Time{}, time.Time{}, "", 20, 0, true)
		Expect(err).To(BeNil())
		Expect(resp.Code).To(Equal(http.StatusNotFound))
		// Body should be ErrorResponse
		if errResp, ok := resp.Body.(generated.ErrorResponse); ok {
			Expect(errResp.Message).To(ContainSubstring("on-demand"))
			Expect(errResp.CreatedAt.IsZero()).To(BeFalse())
		} else {
			Fail("expected ErrorResponse body")
		}
	})

	It("maps Ticketmaster Discovery results to EventsPage when on-demand enabled", func() {
		// Fake Ticketmaster Discovery server
		tmJSON := map[string]any{
			"_embedded": map[string]any{
				"events": []any{
					map[string]any{
						"name": "Test Concert",
						"id":   "E1",
						"url":  "https://example.com/e1",
						"dates": map[string]any{
							"start": map[string]any{
								"localDate": "2025-12-25",
								"localTime": "19:30:00",
							},
						},
						"_embedded": map[string]any{
							"venues": []any{
								map[string]any{
									"id":   "V1",
									"name": "Test Arena",
									"address": map[string]any{
										"line1": "123 Main",
									},
									"city": map[string]any{"name": "Chicago"},
								},
							},
						},
						"priceRanges": []any{},
					},
				},
			},
			"page": map[string]any{
				"size":          1,
				"totalElements": 1,
				"totalPages":    1,
				"number":        0,
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Expect /events.json
			Expect(r.URL.Path).To(Equal("/events.json"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tmJSON)
		}))
		defer server.Close()

		cfg := &config.Config{Ticketmaster: config.TicketmasterConfig{
			OnDemand:    true,
			ConsumerKey: "dummy",
			BaseURL:     server.URL,
			Timeout:     5,
		}}
		tm := concert.NewTicketmasterProxy(&cfg.Ticketmaster)
		svc := business.NewEventSearchService(tm)
		adapter := business.NewGeneratedEventsAdapter(svc, cfg)

		resp, err := adapter.SearchEvents(context.Background(), "rock", "Music", "", "US", "IL", "Chicago", "", 0, time.Time{}, time.Time{}, "date,asc", 20, 0, true)
		Expect(err).To(BeNil())
		Expect(resp.Code).To(Equal(http.StatusOK))
		page, ok := resp.Body.(generated.EventsPage)
		Expect(ok).To(BeTrue(), "body should be EventsPage")
		Expect(page.Total).To(Equal(int32(1)))
		Expect(len(page.Items)).To(Equal(1))
		ev := page.Items[0]
		Expect(ev.ExternalId).To(Equal("E1"))
		Expect(ev.Source).To(Equal("ticketmaster"))
		Expect(ev.Name).To(Equal("Test Concert"))
		Expect(ev.Url).To(Equal("https://example.com/e1"))
		Expect(ev.StartUtc.IsZero()).To(BeFalse())
		Expect(ev.Venue.Name).To(Equal("Test Arena"))
		Expect(ev.Location.City).To(Equal("Chicago"))
	})
})
