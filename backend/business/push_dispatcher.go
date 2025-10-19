package business

import (
    "context"
    "errors"
    "runtime"
    "sync"

    "github.com/google/uuid"
    "github.com/greenwaltc/kellogg-music-match/backend/logger"
)

// PushJob represents a unit of work to send a notification to a single user.
type PushJob struct {
    UserID       uuid.UUID
    Notification WebPushNotification
}

var (
    // ErrDispatcherClosed indicates the dispatcher is closed and cannot accept more jobs
    ErrDispatcherClosed = errors.New("push dispatcher is closed")
)

// PushDispatcher provides an async channel/worker mechanism to deliver push notifications
// without blocking request handlers.
type PushDispatcher struct {
    svc       PushNotificationService
    queue     chan PushJob
    workers   int
    started   bool
    closed    bool
    mu        sync.RWMutex
    startOnce sync.Once
    stopOnce  sync.Once
    wg        sync.WaitGroup
}

// PushNotifier is the narrow interface other packages should depend on.
// Use this to enqueue push notifications instead of calling a direct sender.
// Internally, the dispatcher uses a PushNotificationService to perform delivery.
type PushNotifier interface {
    // EnqueueToUser submits a single-user notification for async delivery.
    EnqueueToUser(ctx context.Context, userID uuid.UUID, n WebPushNotification) error
    // EnqueueToUsers submits the same notification to multiple users and returns how many jobs were queued.
    EnqueueToUsers(ctx context.Context, userIDs []uuid.UUID, n WebPushNotification) int
}

// PushDispatcherOption configures the dispatcher.
type PushDispatcherOption func(*PushDispatcher)

// WithWorkers overrides the number of worker goroutines (default: GOMAXPROCS or 2)
func WithWorkers(n int) PushDispatcherOption {
    return func(d *PushDispatcher) {
        if n > 0 {
            d.workers = n
        }
    }
}

// WithQueueSize overrides the buffered channel size (default: 1024)
func WithQueueSize(sz int) PushDispatcherOption {
    return func(d *PushDispatcher) {
        if sz > 0 {
            d.queue = make(chan PushJob, sz)
        }
    }
}

// NewPushDispatcher creates a new dispatcher around a PushNotificationService.
func NewPushDispatcher(svc PushNotificationService, opts ...PushDispatcherOption) *PushDispatcher {
    d := &PushDispatcher{svc: svc}
    // Defaults
    if d.workers == 0 {
        if p := runtime.GOMAXPROCS(0); p > 0 {
            d.workers = p
            if d.workers < 2 {
                d.workers = 2
            }
        } else {
            d.workers = 2
        }
    }
    if d.queue == nil {
        d.queue = make(chan PushJob, 1024)
    }
    // Apply options
    for _, opt := range opts {
        opt(d)
    }
    return d
}

// Start launches worker goroutines. Safe to call multiple times; only the first call has effect.
func (d *PushDispatcher) Start(ctx context.Context) {
    d.startOnce.Do(func() {
        d.mu.Lock()
        d.started = true
        d.mu.Unlock()
        log := logger.FromCtx(ctx)
        for i := 0; i < d.workers; i++ {
            d.wg.Add(1)
            go func(workerID int) {
                defer d.wg.Done()
                for job := range d.queue {
                    if job.UserID == uuid.Nil {
                        continue
                    }
                    // Best-effort; errors are logged but don't stop the worker
                    if err := d.svc.SendToUser(ctx, job.UserID, job.Notification); err != nil {
                        log.Warn("push send failed", "userId", job.UserID.String(), "err", err.Error())
                    }
                }
            }(i)
        }
    })
}

// Stop gracefully shuts down the dispatcher by closing the queue and waiting for workers to finish.
// Enqueue after Stop will return ErrDispatcherClosed.
func (d *PushDispatcher) Stop() {
    d.stopOnce.Do(func() {
        d.mu.Lock()
        d.closed = true
        if d.queue != nil {
            // Close channel; workers with `for range` will exit cleanly.
            close(d.queue)
        }
        d.mu.Unlock()
        d.wg.Wait()
    })
}

// Enqueue submits a job for asynchronous delivery. If the queue is full, this call blocks until
// the context is done or capacity is available. Returns ErrDispatcherClosed if dispatcher is stopped.
func (d *PushDispatcher) Enqueue(ctx context.Context, job PushJob) (err error) {
    // Guard against panic if channel is closed concurrently
    defer func() {
        if r := recover(); r != nil {
            // Treat as dispatcher closed when sending to a closed channel
            err = ErrDispatcherClosed
        }
    }()
    d.mu.RLock()
    q := d.queue
    started := d.started
    closed := d.closed
    d.mu.RUnlock()
    if q == nil || closed {
        return ErrDispatcherClosed
    }
    if !started {
        // auto-start in simple setups
        d.Start(ctx)
    }
    select {
    case q <- job:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// EnqueueToUser is a convenience wrapper for submitting a single-user job.
func (d *PushDispatcher) EnqueueToUser(ctx context.Context, userID uuid.UUID, n WebPushNotification) error {
    return d.Enqueue(ctx, PushJob{UserID: userID, Notification: n})
}

// EnqueueToUsers is a helper to enqueue the same notification to many users.
// Returns how many jobs were successfully enqueued.
func (d *PushDispatcher) EnqueueToUsers(ctx context.Context, userIDs []uuid.UUID, n WebPushNotification) int {
    enq := 0
    for _, id := range userIDs {
        if err := d.Enqueue(ctx, PushJob{UserID: id, Notification: n}); err == nil {
            enq++
        }
    }
    return enq
}
