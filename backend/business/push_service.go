package business

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/SherClockHolmes/webpush-go"
    "github.com/google/uuid"
    sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
    "github.com/greenwaltc/kellogg-music-match/backend/config"
    "github.com/greenwaltc/kellogg-music-match/backend/logger"
    "net/url"
    "io"
    "encoding/base64"
    "crypto/elliptic"
    "math/big"
)

// WebPushNotification is a portable payload for Web Push
// This should mirror what your Service Worker expects to render a notification
// in the mobile/desktop notification center.
type WebPushNotification struct {
    Title              string                 `json:"title"`
    Body               string                 `json:"body"`
    Icon               string                 `json:"icon,omitempty"`
    Badge              string                 `json:"badge,omitempty"`
    ClickURL           string                 `json:"click_url,omitempty"`
    RequireInteraction bool                   `json:"requireInteraction,omitempty"`
    Data               map[string]any         `json:"data,omitempty"`
    Actions            []map[string]string    `json:"actions,omitempty"` // e.g., {action:"open", title:"View"}
}

// PushNotificationService defines operations for sending web push notifications
// to user device subscriptions stored in the database.
type PushNotificationService interface {
    // SendToUser sends the provided notification to all stored subscriptions for the user.
    // It best-effort delivers to all endpoints; stale subscriptions (404/410) are pruned.
    SendToUser(ctx context.Context, userID uuid.UUID, n WebPushNotification) error

    // SendToUsers sends to many users; returns a per-user error map (nil entry means success)
    // Default implementation loops SendToUser; implementers may optimize/batch.
    SendToUsers(ctx context.Context, userIDs []uuid.UUID, n WebPushNotification) map[uuid.UUID]error
}

// pushQuerier is the narrow database dependency we need; satisfied by sqlc.Queries
type pushQuerier interface {
    GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error)
    DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
}

// webPushService is a concrete PushNotificationService using VAPID keys and webpush-go
type webPushService struct {
    cfg        *config.Config
    db         pushQuerier
    httpClient *http.Client
    ttl        int // seconds
}

// NewWebPushService constructs a new web push sender implementation.
func NewWebPushService(cfg *config.Config, db pushQuerier) PushNotificationService {
    return &webPushService{
        cfg:        cfg,
        db:         db,
        httpClient: &http.Client{Timeout: 15 * time.Second},
        ttl:        300, // widen validity window to reduce risk of BadJwtToken due to small clock skews
    }
}

func (s *webPushService) SendToUser(ctx context.Context, userID uuid.UUID, n WebPushNotification) error {
    log := logger.FromCtx(ctx)
    if err := s.ensureEnabled(); err != nil {
        return err
    }
    if userID == uuid.Nil {
        return errors.New("missing userID")
    }
    // Load subscriptions
    subs, err := s.db.GetPushSubscriptionsByUser(ctx, userID)
    if err != nil {
        return fmt.Errorf("get push subscriptions: %w", err)
    }
    if len(subs) == 0 {
        // No-op
        return nil
    }
    // Build envelope compatible with existing service worker handler: notification+data
    // Provide sensible defaults for icon/badge so platforms that don't fall back to
    // manifest icons still display our app logo in the notification.
    icon := n.Icon
    if icon == "" {
        icon = "/assets/icons/icon-192x192.png"
    }
    badge := n.Badge
    if badge == "" {
        badge = "/assets/icons/badge-72x72.png"
    }
    notif := map[string]any{
        "title":              n.Title,
        "body":               n.Body,
        "icon":               icon,
        "badge":              badge,
        "requireInteraction": n.RequireInteraction,
    }
    // Put click URL in the notification.data.url to simplify click handling in SW
    nd := map[string]any{}
    if n.ClickURL != "" {
        nd["url"] = n.ClickURL
    }
    // Merge any custom data fields into notification.data too (non-destructive)
    for k, v := range n.Data {
        if _, exists := nd[k]; !exists {
            nd[k] = v
        }
    }
    if len(nd) > 0 {
        notif["data"] = nd
    }
    // Top-level data: include timestamp and pass-through any extra fields not already set
    topData := map[string]any{"timestamp": time.Now().UnixMilli()}
    for k, v := range n.Data {
        if _, exists := topData[k]; !exists {
            topData[k] = v
        }
    }
    envelope := map[string]any{
        "notification": notif,
        "data":         topData,
    }
    payload, err := json.Marshal(envelope)
    if err != nil {
        return fmt.Errorf("marshal payload: %w", err)
    }
    opts := &webpush.Options{
        Subscriber:      s.cfg.Push.Subject,
        VAPIDPublicKey:  s.cfg.Push.VAPIDPublic,
        VAPIDPrivateKey: s.cfg.Push.VAPIDPrivate,
        TTL:             s.ttl,
    }
    // If using a custom HTTP client (proxy/timeouts), attach it
    if s.httpClient != nil {
        opts.HTTPClient = s.httpClient
    }

    var firstErr error
    for _, sub := range subs {
        reqSub := &webpush.Subscription{
            Endpoint: sub.Endpoint,
            Keys: webpush.Keys{
                P256dh: sub.P256dh,
                Auth:   sub.Auth,
            },
        }
        resp, sendErr := webpush.SendNotification(payload, reqSub, opts)
        var respBody string
        if resp != nil && resp.Body != nil {
            if b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048)); len(b) > 0 {
                respBody = string(b)
            }
            _ = resp.Body.Close()
        }
        if sendErr != nil {
            // Network or crypto error; keep the subscription, but record error
            if firstErr == nil {
                firstErr = sendErr
            }
            log.Warn("webpush send failed", "endpoint", sub.Endpoint, "err", sendErr.Error())
            continue
        }
        // Diagnostic: log status and provider host for visibility
        func() {
            if resp == nil { return }
            host := ""
            if u, err := url.Parse(sub.Endpoint); err == nil {
                host = u.Host
            }
            if resp.StatusCode >= 400 {
                log.Warn("webpush response error", "status", resp.StatusCode, "host", host, "body", respBody)
            } else {
                log.Info("webpush send", "status", resp.StatusCode, "host", host)
            }
        }()
        // Clean up stale/invalid subscriptions by status code
        if resp != nil && (resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound) {
            if delErr := s.db.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); delErr != nil {
                log.Warn("failed to delete stale push subscription", "endpoint", sub.Endpoint, "err", delErr.Error())
            }
            continue
        }
        // Optionally, we could update last_used_at here if not already handled in the DB query layer.
    }
    return firstErr
}

func (s *webPushService) SendToUsers(ctx context.Context, userIDs []uuid.UUID, n WebPushNotification) map[uuid.UUID]error {
    errs := make(map[uuid.UUID]error, len(userIDs))
    for _, id := range userIDs {
        errs[id] = s.SendToUser(ctx, id, n)
    }
    return errs
}

func (s *webPushService) ensureEnabled() error {
    if s.cfg == nil || !s.cfg.Push.Enabled {
        return errors.New("push disabled by configuration")
    }
    if s.cfg.Push.VAPIDPublic == "" || s.cfg.Push.VAPIDPrivate == "" || s.cfg.Push.Subject == "" {
        return errors.New("missing VAPID config: VAPID_PUBLIC_KEY/VAPID_PRIVATE_KEY/VAPID_SUBJECT")
    }
    // Validate that the configured VAPID public key corresponds to the private key.
    // A mismatch will cause providers (notably Apple) to return 403 BadJwtToken.
    derivedPub, derr := deriveVAPIDPublicFromPrivate(s.cfg.Push.VAPIDPrivate)
    if derr == nil && derivedPub != s.cfg.Push.VAPIDPublic {
        return fmt.Errorf("VAPID key mismatch: VAPID_PUBLIC_KEY does not correspond to VAPID_PRIVATE_KEY")
    }
    return nil
}

// deriveVAPIDPublicFromPrivate computes the uncompressed P-256 public key (base64url, no padding)
// from a base64url-encoded 32-byte private key as used by Web Push VAPID.
func deriveVAPIDPublicFromPrivate(privB64Url string) (string, error) {
    // Decode base64url (no padding) to raw 32-byte scalar
    dBytes, err := base64.RawURLEncoding.DecodeString(privB64Url)
    if err != nil {
        return "", err
    }
    if len(dBytes) != 32 {
        return "", fmt.Errorf("invalid VAPID private key length: got %d, want 32", len(dBytes))
    }
    // Scalar multiply to get public point Q = d * G
    curve := elliptic.P256()
    dx := new(big.Int).SetBytes(dBytes)
    x, y := curve.ScalarBaseMult(dx.Bytes())
    // Encode uncompressed form: 0x04 || X(32) || Y(32)
    xb := x.Bytes()
    yb := y.Bytes()
    // Left-pad to 32 bytes each
    if len(xb) < 32 {
        pad := make([]byte, 32-len(xb))
        xb = append(pad, xb...)
    }
    if len(yb) < 32 {
        pad := make([]byte, 32-len(yb))
        yb = append(pad, yb...)
    }
    out := make([]byte, 65)
    out[0] = 0x04
    copy(out[1:33], xb)
    copy(out[33:], yb)
    // Base64url encode without padding
    return base64.RawURLEncoding.EncodeToString(out), nil
}
