package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	// TracerName is the name used for the operator's tracer.
	TracerName = "github.com/example/platform-operator"

	// ExporterOTLP sends traces to an OTLP-compatible collector (Jaeger, Tempo, etc.)
	ExporterOTLP = "otlp"

	// ExporterStdout writes traces to stdout (useful for debugging).
	ExporterStdout = "stdout"

	// ExporterNone disables tracing entirely.
	ExporterNone = "none"
)

// Config holds the tracing configuration.
type Config struct {
	// Enabled controls whether tracing is active.
	Enabled bool

	// Exporter selects the trace exporter: "otlp", "stdout", or "none".
	Exporter string

	// Endpoint is the OTLP collector endpoint (e.g., "localhost:4317").
	// Only used when Exporter is "otlp".
	Endpoint string

	// ServiceName is the service name reported in traces.
	ServiceName string

	// Insecure disables TLS for the OTLP connection.
	Insecure bool
}

// DefaultConfig returns a disabled tracing configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:     false,
		Exporter:    ExporterNone,
		ServiceName: "platform-operator",
	}
}

// TracerProvider wraps the OpenTelemetry TracerProvider with
// lifecycle management (shutdown).
type TracerProvider struct {
	provider trace.TracerProvider
	shutdown func(context.Context) error
}

// Init initializes the global OpenTelemetry tracer provider.
// Returns a TracerProvider that must be shut down when the operator exits.
//
// When tracing is disabled, a no-op provider is returned with no overhead.
func Init(ctx context.Context, cfg Config) (*TracerProvider, error) {
	if !cfg.Enabled || cfg.Exporter == ExporterNone {
		tp := &TracerProvider{
			provider: noop.NewTracerProvider(),
			shutdown: func(context.Context) error { return nil },
		}
		otel.SetTracerProvider(tp.provider)
		return tp, nil
	}

	// Build resource with service metadata.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTel resource: %w", err)
	}

	var exporter sdktrace.SpanExporter

	switch cfg.Exporter {
	case ExporterOTLP:
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("creating OTLP exporter: %w", err)
		}
		exporter = exp

	case ExporterStdout:
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("creating stdout exporter: %w", err)
		}
		exporter = exp

	default:
		return nil, fmt.Errorf("unknown trace exporter: %s", cfg.Exporter)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return &TracerProvider{
		provider: tp,
		shutdown: tp.Shutdown,
	}, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// Shutdown gracefully flushes and shuts down the tracer provider.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.shutdown != nil {
		return tp.shutdown(ctx)
	}
	return nil
}

// StartSpan creates a new span with standard operator attributes.
// The caller must call span.End() when done.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := Tracer()

	allAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	allAttrs = append(allAttrs, semconv.ServiceName("platform-operator"))
	allAttrs = append(allAttrs, attrs...)

	return tracer.Start(ctx, name, trace.WithAttributes(allAttrs...))
}

// SpanFromContext returns the current span from the context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ReconcileAttributes returns standard attributes for a reconciliation span.
func ReconcileAttributes(name, namespace string, generation int64) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("reconcile.resource", name),
		attribute.String("reconcile.namespace", namespace),
		attribute.Int64("reconcile.generation", generation),
	}
}

// SubReconcileAttributes returns standard attributes for a sub-reconciler span.
func SubReconcileAttributes(resource, appName, namespace string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("subreconcile.resource", resource),
		attribute.String("subreconcile.app", appName),
		attribute.String("subreconcile.namespace", namespace),
	}
}

// CorrelationID generates a unique correlation ID for a reconciliation cycle.
// The ID combines the resource namespace, name, and a generation number to
// correlate spans across a single reconciliation pass.
func CorrelationID(namespace, name string, generation int64) string {
	return fmt.Sprintf("%s/%s@%d", namespace, name, generation)
}

// ReconcileSpan creates a top-level reconciliation span with a correlation ID.
// The caller must call span.End() when done.
func ReconcileSpan(ctx context.Context, name, namespace string, generation int64) (context.Context, trace.Span) {
	corrID := CorrelationID(namespace, name, generation)
	return StartSpan(ctx, "reconcile.PlatformApplication",
		attribute.String("reconcile.name", name),
		attribute.String("reconcile.namespace", namespace),
		attribute.Int64("reconcile.generation", generation),
		attribute.String("reconcile.correlation_id", corrID),
	)
}

// SubReconcileSpan creates a child span for a sub-reconciler operation.
// It inherits the parent span's correlation ID through context propagation.
func SubReconcileSpan(ctx context.Context, resource, appName, namespace string) (context.Context, trace.Span) {
	return StartSpan(ctx, "subreconcile."+resource,
		SubReconcileAttributes(resource, appName, namespace)...)
}

// RecordResourceEvent adds a span event for a resource operation (create, update, delete).
func RecordResourceEvent(span trace.Span, resource, operation string, attrs ...attribute.KeyValue) {
	eventName := fmt.Sprintf("resource.%s.%s", resource, operation)
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}
