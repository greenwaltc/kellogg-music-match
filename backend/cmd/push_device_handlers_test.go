package main

import (
    "bytes"
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/google/uuid"
    "github.com/greenwaltc/kellogg-music-match/backend/business"
    sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

type fakeTokenRepo struct{ store map[string][]business.DeviceToken }

// Device tokens
func (f *fakeTokenRepo) UpsertDeviceToken(_ context.Context, _ uuid.UUID, _, _, _, _, _, _, _ string) error { return nil }
func (f *fakeTokenRepo) ListDeviceTokensByUser(_ context.Context, _ uuid.UUID) ([]business.DeviceToken, error) { return []business.DeviceToken{}, nil }
func (f *fakeTokenRepo) DeleteDeviceToken(_ context.Context, _ uuid.UUID, _, _ string) error { return nil }

// Unused PushRepo methods (stubs)
func (f *fakeTokenRepo) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error { return nil }
func (f *fakeTokenRepo) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) { return nil, nil }
func (f *fakeTokenRepo) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) { return nil, nil }
func (f *fakeTokenRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error { return nil }

// Implement only the methods required by PushRepo here; unused ones can be no-ops or panics for isolation.

func TestRegisterDeviceToken_BadPayload(t *testing.T) {
    h := NewRegisterDeviceTokenHandler(&fakeTokenRepo{})
    req := httptest.NewRequest(http.MethodPost, "/push/device/register", bytes.NewBufferString(`{"platform":"ios"}`))
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", rr.Code) }
}

func TestListDeviceTokens_Unauthorized(t *testing.T) {
    h := NewListDeviceTokensHandler(&fakeTokenRepo{store: map[string][]business.DeviceToken{}})
    req := httptest.NewRequest(http.MethodGet, "/push/device/list", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusUnauthorized { t.Fatalf("expected 401, got %d", rr.Code) }
}

func TestDeleteDeviceToken_BadPayload(t *testing.T) {
    h := NewDeleteDeviceTokenHandler(&fakeTokenRepo{})
    req := httptest.NewRequest(http.MethodDelete, "/push/device", bytes.NewBufferString(`{"platform":"ios"}`))
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", rr.Code) }
}
