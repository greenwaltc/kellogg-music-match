package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

var (
	once   sync.Once
	global *slog.Logger
)

// Init initializes the global structured logger. Safe to call multiple times.
func Init() {
	once.Do(func() {
		// Use JSON handler; level configurable via env LOG_LEVEL
		level := new(slog.LevelVar)
		switch os.Getenv("LOG_LEVEL") {
		case "DEBUG":
			level.Set(slog.LevelDebug)
		case "INFO", "":
			level.Set(slog.LevelInfo)
		case "WARN":
			level.Set(slog.LevelWarn)
		case "ERROR":
			level.Set(slog.LevelError)
		default:
			level.Set(slog.LevelInfo)
		}
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		global = slog.New(handler)
	})
}

// L returns the global logger (initializing if necessary)
func L() *slog.Logger {
	if global == nil {
		Init()
	}
	return global
}

// With returns a logger with additional context fields.
func With(args ...any) *slog.Logger { return L().With(args...) }

// FromCtx returns a logger with request context correlation if you decide to add trace IDs later.
// internal context keys
type ctxKey string

const (
	requestIDKey ctxKey = "requestID"
	userKey      ctxKey = "user" // generic user context (struct with GetUserID or UserID field)
)

// FromCtx returns a logger decorated with correlation attributes (requestId, userId) if present.
func FromCtx(ctx context.Context) *slog.Logger {
	lg := L()
	if ctx == nil {
		return lg
	}
	if rid, _ := ctx.Value(requestIDKey).(string); rid != "" {
		lg = lg.With("requestId", rid)
	}
	if v := ctx.Value(userKey); v != nil {
		// Attempt interface extraction
		switch u := v.(type) {
		case interface{ GetUserID() string }:
			if id := u.GetUserID(); id != "" {
				lg = lg.With("userId", id)
			}
		case interface{ GetId() string }:
			if id := u.GetId(); id != "" {
				lg = lg.With("userId", id)
			}
		case interface{ UserID() string }:
			if id := u.UserID(); id != "" {
				lg = lg.With("userId", id)
			}
		}
	}
	// Enrich with trace/span if available
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			lg = lg.With("traceId", sc.TraceID().String(), "spanId", sc.SpanID().String())
		}
	}
	return lg
}

// WithRequestID adds a request ID to context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extracts request ID string.
func RequestID(ctx context.Context) string {
	if v, _ := ctx.Value(requestIDKey).(string); v != "" {
		return v
	}
	return ""
}

// ContextUserKey exposes key name for tests when injecting user contexts.
func ContextUserKey() any { return userKey }
