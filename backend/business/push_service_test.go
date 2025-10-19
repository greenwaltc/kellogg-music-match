package business

import (
    "context"
    "testing"

    "github.com/google/uuid"
    sqlc "github.com/greenwaltc/kellogg-music-match/backend/db/sqlc"
    "github.com/greenwaltc/kellogg-music-match/backend/config"
)

type fakePushRepo struct{
    subs map[uuid.UUID][]sqlc.PushSubscription
    deleted []string
}

func (f *fakePushRepo) GetPushSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]sqlc.PushSubscription, error) {
    if f.subs == nil { return nil, nil }
    return f.subs[userID], nil
}
func (f *fakePushRepo) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
    f.deleted = append(f.deleted, endpoint)
    return nil
}

func TestWebPushService_DisabledConfig(t *testing.T) {
    cfg := &config.Config{}
    repo := &fakePushRepo{}
    svc := NewWebPushService(cfg, repo)
    err := svc.SendToUser(context.Background(), uuid.New(), WebPushNotification{Title: "t"})
    if err == nil {
        t.Fatalf("expected error when push disabled")
    }
}

func TestWebPushService_NoSubscriptions_NoError(t *testing.T) {
    cfg := &config.Config{Push: config.PushConfig{Enabled: true, VAPIDPublic: "pub", VAPIDPrivate: "priv", Subject: "mailto:x@y"}}
    repo := &fakePushRepo{subs: map[uuid.UUID][]sqlc.PushSubscription{}}
    svc := NewWebPushService(cfg, repo)
    if err := svc.SendToUser(context.Background(), uuid.New(), WebPushNotification{Title: "x"}); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
}

// Note: Full send path uses webpush-go which we don't hit here; further tests would mock SendNotification if we wrap it.
