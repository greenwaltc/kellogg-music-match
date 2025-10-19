package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

type fakeNotifier struct {
	enqueued int
	lastUser uuid.UUID
	lastN    business.WebPushNotification
	err      error
}

func (f *fakeNotifier) EnqueueToUser(ctx context.Context, userID uuid.UUID, n business.WebPushNotification) error {
	f.enqueued++
	f.lastUser = userID
	f.lastN = n
	return f.err
}

func (f *fakeNotifier) EnqueueToUsers(ctx context.Context, userIDs []uuid.UUID, n business.WebPushNotification) int {
	for range userIDs {
		f.enqueued++
	}
	return f.enqueued
}

type fakePushRepo struct {
	ids []uuid.UUID
}

func (f *fakePushRepo) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	return nil
}
func (f *fakePushRepo) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
	return nil, nil
}
func (f *fakePushRepo) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) {
	return nil, nil
}
func (f *fakePushRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return nil
}
func (f *fakePushRepo) GetDistinctPushUserIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	return f.ids, nil
}

func TestEnqueueHandler_Unauthorized(t *testing.T) {
	cfg := &config.Config{Push: config.PushConfig{Enabled: true, VAPIDPublic: "pub", VAPIDPrivate: "priv", Subject: "mailto:x@y"}}
	n := &fakeNotifier{}
	repo := &fakePushRepo{ids: []uuid.UUID{uuid.New()}}
	h := NewEnqueueTestHandler(repo, n, cfg)

	r := httptest.NewRequest(http.MethodPost, "/push/test/enqueue", nil)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestEnqueueHandler_AcceptedAndRateLimit(t *testing.T) {
	cfg := &config.Config{Push: config.PushConfig{Enabled: true, VAPIDPublic: "pub", VAPIDPrivate: "priv", Subject: "mailto:x@y"}}
	n := &fakeNotifier{}
	repo := &fakePushRepo{ids: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}}
	h := NewEnqueueTestHandler(repo, n, cfg)

	uid := uuid.New().String()
	// Perform 3 successful enqueues
	for i := 1; i <= 3; i++ {
		base := httptest.NewRequest(http.MethodPost, "/push/test/enqueue", nil)
		ctx := context.WithValue(base.Context(), UserContextKey, &UserContext{UserID: uid})
		r := base.WithContext(ctx)
		w := httptest.NewRecorder()
		h(w, r)
		if w.Code != http.StatusAccepted {
			t.Fatalf("expected 202 Accepted on attempt %d, got %d", i, w.Code)
		}
		if rem := w.Header().Get("X-RateLimit-Remaining"); rem != strconv.Itoa(3-i) {
			t.Fatalf("remaining wrong: %q", rem)
		}
	}

	// Fourth should be 429
	base := httptest.NewRequest(http.MethodPost, "/push/test/enqueue", nil)
	ctx := context.WithValue(base.Context(), UserContextKey, &UserContext{UserID: uid})
	r := base.WithContext(ctx)
	w := httptest.NewRecorder()
	h(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 4th, got %d", w.Code)
	}
	_ = time.Now()
}
