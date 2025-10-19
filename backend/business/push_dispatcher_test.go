package business

import (
    "context"
    "errors"
    "sync/atomic"
    "testing"
    "time"

    "github.com/google/uuid"
)

// fake service to count deliveries and optionally fail
type fakePushService struct{
    calls int32
    fail bool
}

func (f *fakePushService) SendToUser(ctx context.Context, userID uuid.UUID, n WebPushNotification) error {
    atomic.AddInt32(&f.calls, 1)
    if f.fail {
        return errors.New("send failed")
    }
    return nil
}
func (f *fakePushService) SendToUsers(ctx context.Context, userIDs []uuid.UUID, n WebPushNotification) map[uuid.UUID]error {
    out := make(map[uuid.UUID]error)
    for _, id := range userIDs {
        out[id] = f.SendToUser(ctx, id, n)
    }
    return out
}

func TestPushDispatcher_ProcessesJobs(t *testing.T) {
    svc := &fakePushService{}
    d := NewPushDispatcher(svc, WithWorkers(1), WithQueueSize(4))
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    d.Start(ctx)

    uid := uuid.New()
    n := WebPushNotification{Title: "t"}
    // enqueue some jobs
    for i := 0; i < 3; i++ {
        if err := d.Enqueue(ctx, PushJob{UserID: uid, Notification: n}); err != nil {
            t.Fatalf("enqueue failed: %v", err)
        }
    }
    // give workers a moment
    time.Sleep(100 * time.Millisecond)
    if got := atomic.LoadInt32(&svc.calls); got != 3 {
        t.Fatalf("expected 3 deliveries, got %d", got)
    }

    d.Stop()
}

func TestPushDispatcher_StopPreventsEnqueue(t *testing.T) {
    svc := &fakePushService{}
    d := NewPushDispatcher(svc, WithWorkers(1), WithQueueSize(1))
    ctx := context.Background()
    d.Start(ctx)
    d.Stop()
    err := d.Enqueue(ctx, PushJob{UserID: uuid.New(), Notification: WebPushNotification{}})
    if err == nil {
        t.Fatalf("expected error after stop")
    }
}
