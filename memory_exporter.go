package lab

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type memoryExporter struct {
	Spans []sdktrace.ReadOnlySpan
}

func (e *memoryExporter) ByName(name string) sdktrace.ReadOnlySpan {
	for _, s := range e.Spans {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

func (e *memoryExporter) AttributesFor(name string) map[string]attribute.Value {
	span := e.ByName(name)
	if span == nil {
		return map[string]attribute.Value{}
	}

	attrs := make(map[string]attribute.Value, len(span.Attributes()))
	for _, attr := range span.Attributes() {
		attrs[string(attr.Key)] = attr.Value
	}
	return attrs
}

func (e *memoryExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.Spans = append(e.Spans, spans...)

	return nil
}

func (e *memoryExporter) Shutdown(ctx context.Context) error {
	return nil
}

func CreateTracer(t *testing.T) (context.Context, trace.Span, *memoryExporter) {
	exporter := &memoryExporter{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	ctx, span := tp.Tracer("").Start(t.Context(), "testcase")
	require.True(t, span.IsRecording())

	return ctx, span, exporter
}
