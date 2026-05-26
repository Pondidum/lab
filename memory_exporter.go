package lab

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

type memoryExporter struct {
	Spans []trace.ReadOnlySpan
}

func (e *memoryExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	e.Spans = append(e.Spans, spans...)

	return nil
}

func (e *memoryExporter) Shutdown(ctx context.Context) error {
	return nil
}
