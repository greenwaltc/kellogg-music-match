package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

type fakeRepo struct {
	upserts []struct {
		user                       *uuid.UUID
		endpoint, p256dh, auth, ua string
	}
	subs    []sqlc.PushSubscription
	deleted []string
}

func (f *fakeRepo) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	f.upserts = append(f.upserts, struct {
		user                       *uuid.UUID
		endpoint, p256dh, auth, ua string
	}{userID, endpoint, p256dh, auth, userAgent})
	return nil
}
func (f *fakeRepo) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
	return f.subs, nil
}
func (f *fakeRepo) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) {
	return f.subs, nil
}
func (f *fakeRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	f.deleted = append(f.deleted, endpoint)
	return nil
}
func (f *fakeRepo) GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func TestSubscribeHandler_Unauthorized(t *testing.T) {
	repo := &fakeRepo{}
	h := NewSubscribeHandler(repo)
	r := httptest.NewRequest(http.MethodPost, "/push/subscribe", strings.NewReader(`{"endpoint":"e","keys":{"p256dh":"p","auth":"a"}}`))
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTestHandler_Unauthorized(t *testing.T) {
	repo := &fakeRepo{}
	cfg := &config.Config{Push: config.PushConfig{Enabled: true, VAPIDPublic: "pub", VAPIDPrivate: "priv", Subject: "mailto:test@example.com"}}
	h := NewTestHandler(repo, cfg, func(subJSON []byte, cfg *config.Config) error { return nil })
	r := httptest.NewRequest(http.MethodPost, "/push/test", nil)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestTestHandler_FanoutAndCleanup(t *testing.T) {
	userID := uuid.New().String()
	repo := &fakeRepo{
		subs: []sqlc.PushSubscription{
			{Endpoint: "e1", P256dh: "p1", Auth: "a1"},
			{Endpoint: "e2", P256dh: "p2", Auth: "a2"},
		},
	}
	cfg := &config.Config{Push: config.PushConfig{Enabled: true, VAPIDPublic: "pub", VAPIDPrivate: "priv", Subject: "mailto:test@example.com"}}
	calls := 0
	sender := func(subJSON []byte, cfg *config.Config) error {
		calls++
		if calls == 2 { // simulate stale endpoint on second send
			return errors.New("push status 410: Gone")
		}
		return nil
	}
	h := NewTestHandler(repo, cfg, sender)
	base := httptest.NewRequest(http.MethodPost, "/push/test", nil)
	// attach user to context
	ctx := context.WithValue(base.Context(), UserContextKey, &UserContext{UserID: userID})
	r := base.WithContext(ctx)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if calls != 2 {
		t.Fatalf("expected 2 sends, got %d", calls)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "e2" {
		t.Fatalf("expected stale deletion of e2, got %v", repo.deleted)
	}
}
