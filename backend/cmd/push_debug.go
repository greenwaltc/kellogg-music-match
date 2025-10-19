package main

import (
    "crypto/elliptic"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "math/big"
    "net/http"
    "net/url"

    "github.com/google/uuid"
    "github.com/greenwaltc/kellogg-music-match/backend/config"
    sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
)

// minimal interface needed from repo
type pushDebugRepo interface {
    GetPushSubscriptionsByUser(rctx interface{}, userID uuid.UUID) ([]sqlc.PushSubscription, error)
}

// NewVapidDebugHandler returns a handler that reports VAPID diagnostics.
// Protected by JWT middleware in main. Only enabled when Push.Enabled.
func NewVapidDebugHandler(repo PushRepo, cfg *config.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        if !cfg.Push.Enabled {
            http.Error(w, "push disabled", http.StatusPreconditionFailed)
            return
        }
        // Require authenticated user
        uctx, ok := GetUserFromContext(r.Context())
        if !ok || uctx == nil || uctx.UserID == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        userID, err := uuid.Parse(uctx.UserID)
        if err != nil {
            http.Error(w, "invalid user id", http.StatusUnauthorized)
            return
        }

        // derive public from private key
        derived, derr := deriveVapidPublic(cfg.Push.VAPIDPrivate)
        pairMatch := derr == nil && derived == cfg.Push.VAPIDPublic

        // pull one subscription to compute audience for the current user; if none, fall back to any stored subscription
        subs, _ := repo.GetPushSubscriptionsByUser(r.Context(), userID)
        endpoint := ""
        host := ""
        audience := ""
        userSubsCount := len(subs)
        fromAny := false
        if userSubsCount == 0 {
            // best-effort global scan for any subscription to surface provider host/audience
            if any, _ := repo.GetAnyPushSubscriptions(r.Context(), 1); len(any) > 0 {
                subs = any
                fromAny = true
            }
        }
        if len(subs) > 0 {
            endpoint = subs[0].Endpoint
            if u, err := url.Parse(endpoint); err == nil {
                host = u.Host
                audience = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
            }
        }

        resp := map[string]any{
            "pairMatch":            pairMatch,
            "publicKeyPrefix":      prefix(cfg.Push.VAPIDPublic, 12),
            "derivedPublicPrefix":  prefix(derived, 12),
            "subject":              cfg.Push.Subject,
            "endpointHost":         host,
            "audienceComputed":     audience,
            "hasSubscriptionStored": userSubsCount > 0,
            "userSubsCount":        userSubsCount,
            "usedFallbackAny":      fromAny,
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }
}

func prefix(s string, n int) string {
    if len(s) <= n {
        return s
    }
    return s[:n]
}

// deriveVapidPublic computes base64url-encoded uncompressed P-256 public key from a base64url private key
func deriveVapidPublic(privB64Url string) (string, error) {
    dBytes, err := base64.RawURLEncoding.DecodeString(privB64Url)
    if err != nil {
        return "", err
    }
    if len(dBytes) != 32 {
        return "", fmt.Errorf("invalid private key length: %d", len(dBytes))
    }
    curve := elliptic.P256()
    dx := new(big.Int).SetBytes(dBytes)
    x, y := curve.ScalarBaseMult(dx.Bytes())
    xb := x.Bytes()
    yb := y.Bytes()
    if len(xb) < 32 { xb = append(make([]byte, 32-len(xb)), xb...) }
    if len(yb) < 32 { yb = append(make([]byte, 32-len(yb)), yb...) }
    out := make([]byte, 65)
    out[0] = 0x04
    copy(out[1:33], xb)
    copy(out[33:], yb)
    return base64.RawURLEncoding.EncodeToString(out), nil
}
