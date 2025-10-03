package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/greenwaltc/kellogg-music-match/backend/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	once sync.Once
	tp   *sdktrace.TracerProvider
)

// Init sets up a tracer provider based on telemetry config (stdout or OTLP). Safe for multiple calls.
func Init(cfg config.TelemetryConfig) {
	if !cfg.Enabled {
		slog.Info("telemetry disabled")
		return
	}
	once.Do(func() {
		exporter, err := buildExporter(cfg)
		if err != nil {
			slog.Error("telemetry exporter init failed", "error", err)
			return
		}
		res, err := resource.New(context.Background(), resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		))
		if err != nil {
			slog.Error("telemetry resource init failed", "error", err)
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(2*time.Second)),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(tp)
		slog.Info("telemetry initialized", "exporter", cfg.Exporter)
	})
}

func buildExporter(cfg config.TelemetryConfig) (sdktrace.SpanExporter, error) {
	switch cfg.Exporter {
	case "", "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp":
		if cfg.OTLPEndpoint == "" {
			return nil, errors.New("otlp exporter selected but OTLPEndpoint empty")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint))
	default:
		return nil, errors.New("unknown exporter: " + cfg.Exporter)
	}
}

// Shutdown flushes traces.
func Shutdown(ctx context.Context) error {
	if tp != nil {
		return tp.Shutdown(ctx)
	}
	return nil
}
