package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"strconv"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

// minimal fake repo that returns one dummy subscription for a fixed user
type rlTestRepo struct{}

func (r *rlTestRepo) UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error {
	return nil
}
func (r *rlTestRepo) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
	return []sqlc.PushSubscription{{Endpoint: "https://example/sub", P256dh: "k1", Auth: "a1"}}, nil
}
func (r *rlTestRepo) GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error) {
	return nil, nil
}
func (r *rlTestRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return nil
}

// sender that always succeeds
func okSender(_ []byte, _ *config.Config) error { return nil }

// helper to attach a fake user to context
func reqWithUser(method, target, userID string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(req.Context(), UserContextKey, &UserContext{UserID: userID})
	return req.WithContext(ctx)
}

func TestNewTestHandler_RateLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.Push.Enabled = true
	cfg.Push.VAPIDPublic = "pub"
	cfg.Push.VAPIDPrivate = "priv"
	cfg.Push.Subject = "mailto:test@example.com"

	repo := &rlTestRepo{}
	h := NewTestHandler(repo, cfg, okSender)

	// stable user id
	uid := uuid.New().String()

	// 1..3 should pass (200), X-RateLimit-Remaining should count down 2,1,0
	for i := 1; i <= 3; i++ {
		rr := httptest.NewRecorder()
		req := reqWithUser(http.MethodPost, "/push/test", uid)
		h(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 on attempt %d, got %d", i, rr.Code)
		}
		if rl := rr.Header().Get("X-RateLimit-Limit"); rl != "3" {
			t.Fatalf("expected X-RateLimit-Limit=3, got %q", rl)
		}
		if win := rr.Header().Get("X-RateLimit-Window"); win != "60s" {
			t.Fatalf("expected X-RateLimit-Window=60s, got %q", win)
		}
		// remaining is 3-i
		expRem := 3 - i
		if got := rr.Header().Get("X-RateLimit-Remaining"); got != strconv.Itoa(expRem) {
			t.Fatalf("expected X-RateLimit-Remaining=%d, got %q", expRem, got)
		}
	}

	// 4th within the same window should be 429
	rr := httptest.NewRecorder()
	req := reqWithUser(http.MethodPost, "/push/test", uid)
	h(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 4th attempt, got %d", rr.Code)
	}

	// Advance time by >60s should reset; simulate by sleeping a tiny amount if needed but rely on handler's time.Now
	// Note: Without a clock injection this part is non-deterministic; we skip waiting to keep test fast.
	_ = time.Now()
}
