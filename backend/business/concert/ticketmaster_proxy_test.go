package concert

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
)

// dummyRoundTripper captures the requested URL for inspection without performing a real network call
type dummyRoundTripper struct {
	lastReq *http.Request
}

func (d *dummyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	d.lastReq = r
	// Return minimal valid JSON body for decoding
	resp := &http.Response{
		StatusCode: 200,
		Body:       nopCloser{Reader: dummyReader(`{"_embedded":{"events":[]},"page":{"size":0,"totalElements":0,"totalPages":0,"number":0}}`)},
		Header:     make(http.Header),
	}
	return resp, nil
}

type dummyReader string

func (d dummyReader) Read(b []byte) (int, error) {
	if len(d) == 0 {
		return 0, http.ErrBodyReadAfterClose
	}
	n := copy(b, d)
	d = d[n:]
	return n, nil
}

type nopCloser struct{ Reader dummyReader }

func (n nopCloser) Read(b []byte) (int, error) { return n.Reader.Read(b) }
func (n nopCloser) Close() error               { return nil }

// helper to build proxy with overridden http client
func newTestProxy(cfg config.TicketmasterConfig, rt http.RoundTripper) *TicketmasterProxy {
	cfg.Timeout = 5
	p := NewTicketmasterProxy(&cfg)
	p.httpClient = &http.Client{Timeout: time.Second * 5, Transport: rt}
	return p
}

func TestFetchConcerts_GeoMode(t *testing.T) {
	rt := &dummyRoundTripper{}
	cfg := config.TicketmasterConfig{
		ConsumerKey:     "dummy",
		BaseURL:         "https://example.com",
		MaxResults:      100,
		DefaultCity:     "Chicago",
		DefaultState:    "IL",
		DefaultCountry:  "US",
		DateRangeMonths: 1,
		GeoLatLong:      "41.8781,-87.6298",
		Radius:          50,
		RadiusUnit:      "miles",
	}
	proxy := newTestProxy(cfg, rt)
	_, err := proxy.FetchConcerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.lastReq == nil {
		t.Fatalf("expected request to be made")
	}
	q := rt.lastReq.URL.Query()
	if q.Get("geoPoint") != cfg.GeoLatLong || q.Get("radius") != "50" || q.Get("unit") != "miles" {
		t.Errorf("expected geo params, got: %v", rt.lastReq.URL.String())
	}
	if q.Get("city") != "" {
		t.Errorf("did not expect city when geo present")
	}
}

func TestFetchConcerts_CityFallbackInvalidGeo(t *testing.T) {
	rt := &dummyRoundTripper{}
	cfg := config.TicketmasterConfig{
		ConsumerKey:     "dummy",
		BaseURL:         "https://example.com",
		MaxResults:      25,
		DefaultCity:     "Chicago",
		DefaultState:    "IL",
		DefaultCountry:  "US",
		DateRangeMonths: 1,
		GeoLatLong:      "bad-value",
		Radius:          30,
	}
	proxy := newTestProxy(cfg, rt)
	_, err := proxy.FetchConcerts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := rt.lastReq.URL.Query()
	if q.Get("city") != "Chicago" || q.Get("stateCode") != "IL" {
		t.Errorf("expected city/state fallback, got URL: %s", rt.lastReq.URL.String())
	}
	if q.Get("geoPoint") != "" {
		t.Errorf("expected no geoPoint when invalid geo provided")
	}
}

func TestFetchConcertsByArtist_Geo(t *testing.T) {
	rt := &dummyRoundTripper{}
	cfg := config.TicketmasterConfig{
		ConsumerKey:     "dummy",
		BaseURL:         "https://example.com",
		MaxResults:      25,
		DefaultCity:     "Chicago",
		DefaultState:    "IL",
		DefaultCountry:  "US",
		DateRangeMonths: 1,
		GeoLatLong:      "41.9,-87.6",
		Radius:          10,
		RadiusUnit:      "km",
	}
	proxy := newTestProxy(cfg, rt)
	_, err := proxy.FetchConcertsByArtist(context.Background(), "Muse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := rt.lastReq.URL.Query()
	if q.Get("geoPoint") == "" || q.Get("radius") != "10" || q.Get("unit") != "km" {
		t.Errorf("expected geo params in artist search, got URL: %s", rt.lastReq.URL.String())
	}
}

func TestFetchConcertsByArtist_CityFallback_NoRadius(t *testing.T) {
	rt := &dummyRoundTripper{}
	cfg := config.TicketmasterConfig{
		ConsumerKey:     "dummy",
		BaseURL:         "https://example.com",
		MaxResults:      25,
		DefaultCity:     "Chicago",
		DefaultState:    "IL",
		DefaultCountry:  "US",
		DateRangeMonths: 1,
		GeoLatLong:      "41.9,-87.6",
		Radius:          0, // radius disabled
	}
	proxy := newTestProxy(cfg, rt)
	_, err := proxy.FetchConcertsByArtist(context.Background(), "Muse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := rt.lastReq.URL.Query()
	if q.Get("city") != "Chicago" || q.Get("geoPoint") != "" {
		t.Errorf("expected city fallback when radius=0, got URL: %s", rt.lastReq.URL.String())
	}
}
