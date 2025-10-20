package business

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "bytes"
    "encoding/json"
    "io"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/greenwaltc/kellogg-music-match/backend/config"
    "github.com/greenwaltc/kellogg-music-match/backend/logger"
    apns "github.com/sideshow/apns2"
    apnstoken "github.com/sideshow/apns2/token"
    "golang.org/x/oauth2"
    "golang.org/x/oauth2/google"
)

// UnifiedPushService sends notifications via WebPush, APNs, and FCM depending on stored endpoints/tokens.
// Initially, APNs/FCM are optional; if not configured, it will gracefully skip those channels.

type UnifiedPushService struct {
    cfg        *config.Config
    repo       UserRepository
    web        PushNotificationService // existing web push sender
    httpClient *http.Client
    // test hooks / dependency overrides
    apnsPushFunc func(req *apns.Notification) (*apns.Response, error)
    fcmDo        func(req *http.Request) (*http.Response, error)
}

func NewUnifiedPushService(cfg *config.Config, repo UserRepository) *UnifiedPushService {
    return &UnifiedPushService{
        cfg:  cfg,
        repo: repo,
        web:  NewWebPushService(cfg, repo),
        httpClient: &http.Client{Timeout: 15 * time.Second},
    }
}

// UnifiedNotification is a superset payload; for now reuse WebPushNotification for content.
// Platform-specific fields (e.g., APNs sound/category) can be added later.

type UnifiedNotification = WebPushNotification

// SendToUser fan-outs to all endpoints for the user across channels.
func (u *UnifiedPushService) SendToUser(ctx context.Context, userID uuid.UUID, n UnifiedNotification) error {
    if userID == uuid.Nil { return errors.New("missing userID") }
    log := logger.FromCtx(ctx)

    var firstErr error

    // 1) Web Push
    if u.cfg.Push.Enabled {
        if err := u.web.SendToUser(ctx, userID, n); err != nil {
            if firstErr == nil {
                firstErr = err
            }
        }
    }

    // 2) Native push: APNs / FCM if enabled
    if u.cfg.NativePush.Enabled {
        // Use a narrow interface to avoid expanding the public repository interface
        type deviceLister interface {
            ListDeviceTokensByUser(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error)
        }
        type tokenDeleter interface {
            DeleteDeviceToken(ctx context.Context, userID uuid.UUID, platform, token string) error
        }
        if dl, ok := any(u.repo).(deviceLister); ok {
            tokens, err := dl.ListDeviceTokensByUser(ctx, userID)
            if err != nil {
                if firstErr == nil { firstErr = fmt.Errorf("list device tokens: %w", err) }
            } else {
                var td tokenDeleter
                if d, ok := any(u.repo).(tokenDeleter); ok { td = d }
                for _, t := range tokens {
                    switch t.Platform {
                    case "ios":
                        if u.cfg.NativePush.APNs.Enabled {
                            prune, err := u.sendAPNs(ctx, userID, t, n)
                            if prune && td != nil { _ = td.DeleteDeviceToken(ctx, userID, t.Platform, t.Token) }
                            if err != nil {
                                log.Warn("apns send failed", "err", err)
                                if firstErr == nil { firstErr = err }
                            }
                        }
                    case "android":
                        if u.cfg.NativePush.FCM.Enabled {
                            prune, err := u.sendFCM(ctx, userID, t, n)
                            if prune && td != nil { _ = td.DeleteDeviceToken(ctx, userID, t.Platform, t.Token) }
                            if err != nil {
                                log.Warn("fcm send failed", "err", err)
                                if firstErr == nil { firstErr = err }
                            }
                        }
                    }
                }
            }
        }
    }

    return firstErr
}

// sendAPNs returns prune=true if the token should be removed (e.g., Unregistered/BadDeviceToken)
func (u *UnifiedPushService) sendAPNs(ctx context.Context, userID uuid.UUID, t DeviceToken, n UnifiedNotification) (bool, error) {
    cfg := u.cfg.NativePush.APNs
    if !u.cfg.NativePush.Enabled || !cfg.Enabled {
        return false, nil
    }
    // Build payload
    // APNs payload
    p := map[string]any{
        "aps": map[string]any{
            "alert": map[string]any{
                "title": n.Title,
                "body":  n.Body,
            },
            "sound": "default",
        },
    }
    if n.ClickURL != "" {
        p["url"] = n.ClickURL
    }
    if len(n.Data) > 0 {
        // Merge custom fields at top-level
        for k, v := range n.Data { if _, exists := p[k]; !exists { p[k] = v } }
    }
    b, _ := json.Marshal(p)
    req := &apns.Notification{Topic: cfg.BundleID, DeviceToken: t.Token, Payload: b}

    // Test hook: short-circuit push if provided
    if u.apnsPushFunc != nil {
        resp, err := u.apnsPushFunc(req)
        if err != nil { return false, err }
        if resp != nil && resp.Sent() { return false, nil }
        // prune for common invalid reasons
        if resp != nil && (resp.Reason == "BadDeviceToken" || resp.Reason == "Unregistered" || resp.Reason == "DeviceTokenNotForTopic") {
            return true, fmt.Errorf("apns error: %s", resp.Reason)
        }
        if resp != nil { return false, fmt.Errorf("apns error: %s", resp.Reason) }
        return false, fmt.Errorf("apns unknown error")
    }

    if cfg.TeamID == "" || cfg.KeyID == "" || cfg.KeyPEM == "" || cfg.BundleID == "" || t.Token == "" {
        // Missing config or token; skip without failing the whole send
        return false, nil
    }
    authKey, err := apnstoken.AuthKeyFromBytes([]byte(cfg.KeyPEM))
    if err != nil { return false, err }
    token := &apnstoken.Token{AuthKey: authKey, KeyID: cfg.KeyID, TeamID: cfg.TeamID}
    client := apns.NewTokenClient(token)
    if cfg.Env == "production" { client = client.Production() } else { client = client.Development() }
    resp, err := client.Push(req)
    if err != nil { return false, err }
    if resp != nil && resp.Sent() { return false, nil }
    if resp != nil {
        if shouldPruneAPNs(resp.Reason) {
            return true, fmt.Errorf("apns error: %s", resp.Reason)
        }
        return false, fmt.Errorf("apns error: %s", resp.Reason)
    }
    return false, fmt.Errorf("apns unknown error")
}

// sendFCM returns prune=true if the token should be removed (e.g., NotRegistered/404)
func (u *UnifiedPushService) sendFCM(ctx context.Context, userID uuid.UUID, t DeviceToken, n UnifiedNotification) (bool, error) {
    cfg := u.cfg.NativePush.FCM
    if !u.cfg.NativePush.Enabled || !cfg.Enabled {
        return false, nil
    }
    var httpClient *http.Client
    if u.fcmDo != nil {
        // test hook: use a dummy client that calls fcmDo
        httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) { return u.fcmDo(req) })}
    } else {
        if cfg.ProjectID == "" || cfg.ServiceAccount == "" || t.Token == "" { return false, nil }
        ts, err := google.CredentialsFromJSON(ctx, []byte(cfg.ServiceAccount), "https://www.googleapis.com/auth/firebase.messaging")
        if err != nil { return false, err }
        httpClient = oauth2.NewClient(ctx, ts.TokenSource)
    }
    // Build FCM HTTP v1 message
    msg := map[string]any{
        "message": map[string]any{
            "token": t.Token,
            "notification": map[string]any{"title": n.Title, "body": n.Body},
            "data": mergeData(map[string]string{}, n),
        },
    }
    body, _ := json.Marshal(msg)
    url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", cfg.ProjectID)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil { return false, err }
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    resp, err := httpClient.Do(req)
    if err != nil { return false, err }
    defer resp.Body.Close()
    if resp.StatusCode >= 200 && resp.StatusCode < 300 { return false, nil }
    // classify using body JSON when possible
    var bodyBytes []byte
    if resp.Body != nil {
        bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 4096))
    }
    if shouldPruneFCM(resp.StatusCode, bodyBytes) {
        return true, fmt.Errorf("fcm status %d", resp.StatusCode)
    }
    return false, fmt.Errorf("fcm status %d", resp.StatusCode)
}

func mergeData(dst map[string]string, n UnifiedNotification) map[string]string {
    if n.ClickURL != "" { dst["url"] = n.ClickURL }
    for k, v := range n.Data {
        if s, ok := v.(string); ok { if _, exists := dst[k]; !exists { dst[k] = s } }
    }
    dst["timestamp"] = fmt.Sprintf("%d", time.Now().UnixMilli())
    return dst
}

// shouldPruneAPNs returns true if APNs error reason implies the device token is invalid/stale.
func shouldPruneAPNs(reason string) bool {
    switch reason {
    case "BadDeviceToken", "Unregistered", "DeviceTokenNotForTopic":
        return true
    default:
        // Non-pruning examples: TopicDisallowed, PayloadTooLarge, BadTopic, ExpiredProviderToken, BadPath
        return false
    }
}

// shouldPruneFCM inspects HTTP status and JSON error body to decide whether to prune the token.
// It recognizes v1 error payloads with error.status=="UNREGISTERED" or details[].errorCode=="UNREGISTERED".
func shouldPruneFCM(status int, body []byte) bool {
    if status == 404 || status == 410 {
        return true
    }
    // 400 may be many things; only prune when explicit UNREGISTERED is present
    var j struct{
        Error struct{
            Code int `json:"code"`
            Message string `json:"message"`
            Status string `json:"status"`
            Details []struct{ Type string `json:"@type"`; ErrorCode string `json:"errorCode"` } `json:"details"`
        } `json:"error"`
    }
    if len(body) > 0 && json.Unmarshal(body, &j) == nil {
        if strings.EqualFold(j.Error.Status, "UNREGISTERED") { return true }
        for _, d := range j.Error.Details {
            if strings.EqualFold(d.ErrorCode, "UNREGISTERED") { return true }
        }
    }
    return false
}

// roundTripperFunc allows injecting an http client Do override for tests
type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
    return f(req)
}

// If needed in future: batch send to many users
func (u *UnifiedPushService) SendToUsers(ctx context.Context, userIDs []uuid.UUID, n UnifiedNotification) map[uuid.UUID]error {
    errs := make(map[uuid.UUID]error, len(userIDs))
    for _, id := range userIDs {
        errs[id] = u.SendToUser(ctx, id, n)
    }
    return errs
}
