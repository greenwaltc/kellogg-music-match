package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

// PushSender abstracts sending a Web Push payload to a subscription
type PushSender func(subJSON []byte, cfg *config.Config) error

// PushRepo is the minimal repository contract required by push handlers
type PushRepo interface {
	UpsertPushSubscription(ctx context.Context, userID *uuid.UUID, endpoint, p256dh, auth, userAgent string) error
	GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error)
	GetAnyPushSubscriptions(ctx context.Context, lim int32) ([]sqlc.PushSubscription, error)
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
}

// NewSubscribeHandler returns an http.HandlerFunc to upsert a subscription for the current user
func NewSubscribeHandler(repo PushRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var sub struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				P256dh string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&sub); err != nil || sub.Endpoint == "" || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
			http.Error(w, "invalid subscription", http.StatusBadRequest)
			return
		}
		// Require an authenticated user
		var uidPtr *uuid.UUID
		if u, ok := GetUserFromContext(r.Context()); ok && u != nil && u.UserID != "" {
			if parsed, err := uuid.Parse(u.UserID); err == nil {
				uidPtr = &parsed
			}
		}
		if uidPtr == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ua := r.Header.Get("User-Agent")
		if err := repo.UpsertPushSubscription(r.Context(), uidPtr, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth, ua); err != nil {
			http.Error(w, "failed to store subscription", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
	}
}

// NewTestHandler returns an http.HandlerFunc to send a test push to all of the user's subscriptions
func NewTestHandler(repo PushRepo, cfg *config.Config, sender PushSender) http.HandlerFunc {
	// naive in-memory per-user rate limiter: allow 3 requests per 60s window
	type rlState struct {
		count       int
		windowStart time.Time
	}
	limiter := struct {
		m  map[uuid.UUID]*rlState
		mu sync.Mutex
	}{m: make(map[uuid.UUID]*rlState)}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !cfg.Push.Enabled || cfg.Push.VAPIDPrivate == "" || cfg.Push.VAPIDPublic == "" {
			http.Error(w, "push not configured", http.StatusPreconditionFailed)
			return
		}

		// Require authenticated user and fan-out to all of their subscriptions
		var userID uuid.UUID
		if u, ok := GetUserFromContext(r.Context()); !ok || u == nil || u.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		} else {
			parsed, err := uuid.Parse(u.UserID)
			if err != nil {
				http.Error(w, "invalid user id", http.StatusUnauthorized)
				return
			}
			userID = parsed
		}

		// rate limit check
		limiter.mu.Lock()
		st := limiter.m[userID]
		now := time.Now()
		if st == nil || now.Sub(st.windowStart) > 60*time.Second {
			st = &rlState{count: 0, windowStart: now}
			limiter.m[userID] = st
		}
		st.count++
		cur := st.count
		limiter.mu.Unlock()

		// Set basic rate limit headers for visibility
		w.Header().Set("X-RateLimit-Limit", "3")
		w.Header().Set("X-RateLimit-Window", "60s")
		rem := 3 - cur
		if rem < 0 {
			rem = 0
		}
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rem))
		if cur > 3 {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}

		subs, err := repo.GetPushSubscriptionsByUser(r.Context(), userID)
		if err != nil || len(subs) == 0 {
			http.Error(w, "no subscription available", http.StatusNotFound)
			return
		}

		var sent, failed, deleted int
		for _, s := range subs {
			subJSON, _ := json.Marshal(map[string]interface{}{
				"endpoint": s.Endpoint,
				"keys":     map[string]string{"p256dh": s.P256dh, "auth": s.Auth},
				"ts":       time.Now().Unix(),
			})
			if err := sender(subJSON, cfg); err != nil {
				failed++
				// Best-effort cleanup on 404/410 from push provider
				if strings.Contains(err.Error(), "status 404") || strings.Contains(err.Error(), "status 410") {
					if derr := repo.DeletePushSubscriptionByEndpoint(r.Context(), s.Endpoint); derr == nil {
						deleted++
					}
				}
				continue
			}
			sent++
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "sent", "sent": sent, "failed": failed, "deleted": deleted})
	}
}
