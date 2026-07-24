package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("expected default config to be disabled")
	}
	if cfg.Exporter != ExporterNone {
		t.Errorf("expected exporter %q, got %q", ExporterNone, cfg.Exporter)
	}
	if cfg.ServiceName != "platform-operator" {
		t.Errorf("expected service name %q, got %q", "platform-operator", cfg.ServiceName)
	}
}

func TestInit_Disabled(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig() // disabled

	tp, err := Init(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
	defer tp.Shutdown(ctx)

	// Should be noop provider — calling Tracer() should not panic.
	tracer := Tracer()
	if tracer == nil {
		t.Fatal("expected non-nil tracer from noop provider")
	}
}

func TestInit_StdoutExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Enabled:     true,
		Exporter:    ExporterStdout,
		ServiceName: "test-operator",
	}

	tp, err := Init(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
	defer tp.Shutdown(ctx)

	tracer := Tracer()
	if tracer == nil {
		t.Fatal("expected non-nil tracer from stdout provider")
	}
}

func TestInit_UnknownExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Enabled:     true,
		Exporter:    "jaeger", // unsupported
		ServiceName: "test-operator",
	}

	_, err := Init(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for unknown exporter, got nil")
	}
}

func TestInit_DisabledOverridesExporter(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		Enabled:  false,
		Exporter: ExporterOTLP, // would fail without endpoint, but disabled takes precedence
	}

	tp, err := Init(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error when disabled: %v", err)
	}
	defer tp.Shutdown(ctx)
}

func TestStartSpan_NoopProvider(t *testing.T) {
	// Initialize with noop first.
	ctx := context.Background()
	tp, err := Init(ctx, DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tp.Shutdown(ctx)

	ctx2, span := StartSpan(ctx, "test-span",
		attribute.String("key", "value"),
	)
	if ctx2 == nil {
		t.Fatal("expected non-nil context from StartSpan")
	}
	if span == nil {
		t.Fatal("expected non-nil span from StartSpan")
	}
	span.End()
}

func TestSpanFromContext(t *testing.T) {
	ctx := context.Background()
	tp, err := Init(ctx, DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer tp.Shutdown(ctx)

	ctx2, span := StartSpan(ctx, "parent-span")
	defer span.End()

	got := SpanFromContext(ctx2)
	if got == nil {
		t.Fatal("expected non-nil span from SpanFromContext")
	}
}

func TestReconcileAttributes(t *testing.T) {
	attrs := ReconcileAttributes("my-app", "production", 5)

	if len(attrs) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(attrs))
	}

	expected := map[string]string{
		"reconcile.resource":  "my-app",
		"reconcile.namespace": "production",
	}
	for _, a := range attrs {
		key := string(a.Key)
		if exp, ok := expected[key]; ok {
			if a.Value.AsString() != exp {
				t.Errorf("attribute %q: expected %q, got %q", key, exp, a.Value.AsString())
			}
		}
		if key == "reconcile.generation" {
			if a.Value.AsInt64() != 5 {
				t.Errorf("expected generation 5, got %d", a.Value.AsInt64())
			}
		}
	}
}

func TestSubReconcileAttributes(t *testing.T) {
	attrs := SubReconcileAttributes("Deployment", "my-app", "default")

	if len(attrs) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(attrs))
	}

	expected := map[string]string{
		"subreconcile.resource":  "Deployment",
		"subreconcile.app":       "my-app",
		"subreconcile.namespace": "default",
	}
	for _, a := range attrs {
		key := string(a.Key)
		if exp, ok := expected[key]; ok {
			if a.Value.AsString() != exp {
				t.Errorf("attribute %q: expected %q, got %q", key, exp, a.Value.AsString())
			}
		}
	}
}

func TestShutdown_NilShutdown(t *testing.T) {
	tp := &TracerProvider{}
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Errorf("expected nil error from nil shutdown func, got: %v", err)
	}
}
