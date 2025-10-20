package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/url"

	"github.com/google/uuid"
	"github.com/greenwaltc/kellogg-music-match/backend/business"
	"github.com/greenwaltc/kellogg-music-match/backend/config"
	sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
	"github.com/greenwaltc/kellogg-music-match/backend/logger"
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

// DeviceTokenRepo is the minimal repository contract for managing native device tokens
type DeviceTokenRepo interface {
	UpsertDeviceToken(ctx context.Context, userID uuid.UUID, platform, token, bundleID, appPackage, deviceModel, osVersion, appVersion string) error
	ListDeviceTokensByUser(ctx context.Context, userID uuid.UUID) ([]business.DeviceToken, error)
	DeleteDeviceToken(ctx context.Context, userID uuid.UUID, platform, token string) error
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
		// Diagnostics: log endpoint host and UA
		if u, _ := url.Parse(sub.Endpoint); u != nil {
			logger.L().Info("push subscription stored", "userId", uidPtr.String(), "host", u.Host, "ua", ua)
		} else {
			logger.L().Info("push subscription stored", "userId", uidPtr.String(), "endpoint", sub.Endpoint, "ua", ua)
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

		// rate limit check (bypass in debug mode to ease testing)
		if !cfg.Debug.Enabled {
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
		} else {
			w.Header().Set("X-RateLimit-Limit", "debug")
			w.Header().Set("X-RateLimit-Window", "0s")
			w.Header().Set("X-RateLimit-Remaining", "debug")
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

// NewEnqueueTestHandler enqueues a basic push notification for the authenticated user via the async dispatcher
func NewEnqueueTestHandler(_ PushRepo, notifier business.PushNotifier, cfg *config.Config) http.HandlerFunc {
	// reuse the same simple limiter shape
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

		// user auth required
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

		// rate limit (bypass in debug mode)
		if !cfg.Debug.Enabled {
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
		} else {
			w.Header().Set("X-RateLimit-Limit", "debug")
			w.Header().Set("X-RateLimit-Window", "0s")
			w.Header().Set("X-RateLimit-Remaining", "debug")
		}

		// Build a tiny test notification; business layer wraps into SW-compatible envelope
		n := business.WebPushNotification{
			Title:              "Kellogg Music Match",
			Body:               "Async test notification queued",
			Icon:               "/assets/icons/icon-192x192.png",
			Badge:              "/assets/icons/badge-72x72.png",
			ClickURL:           "/matches",
			RequireInteraction: false,
			Data:               map[string]any{"source": "enqueue-test"},
		}
		// Enqueue one job for this user. The dispatcher will call SendToUser which fans out
		// to ALL of the user's stored subscriptions (desktop + phone), so each device gets it.
		// Use a short timeout to avoid blocking on a full queue.
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := notifier.EnqueueToUser(ctx, userID, n); err != nil {
			http.Error(w, "failed to enqueue notification: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "enqueued"})
	}
}

// NewRegisterDeviceTokenHandler stores a native push token for APNs/FCM
func NewRegisterDeviceTokenHandler(repo DeviceTokenRepo) http.HandlerFunc {
	type req struct {
		Platform    string `json:"platform"` // ios|android
		Token       string `json:"token"`
		BundleID    string `json:"bundleId,omitempty"`
		AppPackage  string `json:"appPackage,omitempty"`
		DeviceModel string `json:"deviceModel,omitempty"`
		OSVersion   string `json:"osVersion,omitempty"`
		AppVersion  string `json:"appVersion,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || (body.Platform != "ios" && body.Platform != "android") {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		// Require authenticated user
		var userID uuid.UUID
		if uctx, ok := GetUserFromContext(r.Context()); !ok || uctx == nil || uctx.UserID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		} else {
			parsed, err := uuid.Parse(uctx.UserID)
			if err != nil {
				http.Error(w, "invalid user id", http.StatusUnauthorized)
				return
			}
			userID = parsed
		}
		if err := repo.UpsertDeviceToken(r.Context(), userID, body.Platform, body.Token, body.BundleID, body.AppPackage, body.DeviceModel, body.OSVersion, body.AppVersion); err != nil {
			http.Error(w, "failed to store token", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status":"ok"})
	}
}

// NewListDeviceTokensHandler returns device tokens for the authenticated user
func NewListDeviceTokensHandler(repo DeviceTokenRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
		uctx, ok := GetUserFromContext(r.Context()); if !ok || uctx == nil || uctx.UserID == "" { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		userID, err := uuid.Parse(uctx.UserID); if err != nil { http.Error(w, "invalid user id", http.StatusUnauthorized); return }
		toks, err := repo.ListDeviceTokensByUser(r.Context(), userID)
		if err != nil { http.Error(w, "failed to load tokens", http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tokens": toks})
	}
}

// NewDeleteDeviceTokenHandler deletes a token for the authenticated user
func NewDeleteDeviceTokenHandler(repo DeviceTokenRepo) http.HandlerFunc {
	type req struct{ Platform string `json:"platform"`; Token string `json:"token"` }
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete { w.WriteHeader(http.StatusMethodNotAllowed); return }
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || (body.Platform != "ios" && body.Platform != "android") {
			http.Error(w, "invalid payload", http.StatusBadRequest); return
		}
		uctx, ok := GetUserFromContext(r.Context()); if !ok || uctx == nil || uctx.UserID == "" { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		userID, err := uuid.Parse(uctx.UserID); if err != nil { http.Error(w, "invalid user id", http.StatusUnauthorized); return }
		if err := repo.DeleteDeviceToken(r.Context(), userID, body.Platform, body.Token); err != nil {
			http.Error(w, "failed to delete", http.StatusInternalServerError); return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status":"deleted"})
	}
}
