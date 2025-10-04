package business

import (
	"context"

	"github.com/greenwaltc/kellogg-music-match/backend/generated"
)

// GeneratedConcertsAdapter adapts the business ConcertAPIService to the generated.ConcertsAPIServicer for
// only the implemented endpoints we support. Unimplemented endpoints fall back to the generated stubs.
type GeneratedConcertsAdapter struct {
	Inner     *ConcertAPIService
	Generated *generated.ConcertsAPIService // fallback for unimplemented endpoints
}

func NewGeneratedConcertsAdapter(inner *ConcertAPIService) *GeneratedConcertsAdapter {
	return &GeneratedConcertsAdapter{Inner: inner, Generated: generated.NewConcertsAPIService()}
}

// Delegate Chicago events to business logic
func (a *GeneratedConcertsAdapter) GetChicagoEvents(ctx context.Context, artistName string, limit int32, offset int32, anyInterest bool) (generated.ImplResponse, error) {
	return a.Inner.GetChicagoEvents(ctx, artistName, limit, offset, anyInterest)
}

// Pass-through to generated stubs for endpoints we have not overridden yet
// Interest mutation endpoints delegate to business logic so they are implemented
func (a *GeneratedConcertsAdapter) SetEventInterest(ctx context.Context, eventId string, xUserUsername string, req generated.SetEventInterestRequest) (generated.ImplResponse, error) {
	return a.Inner.SetEventInterest(ctx, eventId, xUserUsername, req)
}
func (a *GeneratedConcertsAdapter) RemoveEventInterest(ctx context.Context, eventId string, xUserUsername string) (generated.ImplResponse, error) {
	return a.Inner.RemoveEventInterest(ctx, eventId, xUserUsername)
}
func (a *GeneratedConcertsAdapter) SearchConcerts(ctx context.Context, artist, city, state, country, genre, startDate, endDate string, page, size int32) (generated.ImplResponse, error) {
	return a.Generated.SearchConcerts(ctx, artist, city, state, country, genre, startDate, endDate, page, size)
}
func (a *GeneratedConcertsAdapter) GetConcertById(ctx context.Context, eventId string) (generated.ImplResponse, error) {
	return a.Generated.GetConcertById(ctx, eventId)
}
func (a *GeneratedConcertsAdapter) GetChicagoEventById(ctx context.Context, eventId string) (generated.ImplResponse, error) {
	return a.Inner.GetChicagoEventById(ctx, eventId)
}
