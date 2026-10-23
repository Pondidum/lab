package lab

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Reporter[T any] func(ctx context.Context, trials Analysis[T])

func defaultReporter[T any](ctx context.Context, trials Analysis[T]) {
	TraceReporter(ctx, trials)
}

func TraceReporter[T any](ctx context.Context, analysis Analysis[T]) {
	tr := otel.Tracer("lab")

	attrs := AttributesFromAnalysis[T](analysis)
	_, s := tr.Start(ctx, "trial", trace.WithAttributes(attrs...))
	s.End()

}

func AttributesFromAnalysis[T any](analysis Analysis[T]) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("trial.name", analysis.Name),
		attribute.Bool("trial.passed", analysis.Passed),
		attribute.Bool("trial.results_match", analysis.ResultsMatch),
	}

	if analysis.BaselineError != "" || analysis.TrialError != "" {
		attrs = append(attrs, attribute.Bool("trial.errors_match", analysis.ErrorsMatch))
	}

	if analysis.BaselineError != "" {
		attrs = append(attrs, attribute.String("trial.error", analysis.BaselineError))
	}
	if analysis.TrialError != "" {
		attrs = append(attrs, attribute.String("trial.error", analysis.TrialError))
	}
	if analysis.TrialPanic != nil {
		attrs = append(attrs, attribute.String("trial.panic", fmt.Sprint(analysis.TrialPanic)))
	}

	if !analysis.ResultsMatch {
		attrs = append(attrs,
			asOtelAttribute("baseline.result", analysis.BaselineResult),
			asOtelAttribute("trial.result", analysis.TrialResult),
		)
	}

	for k, v := range analysis.Attributes {
		attrs = append(attrs, asOtelAttribute(k, v))
	}

	return attrs
}

func asOtelAttribute(name string, val any) attribute.KeyValue {
	switch v := val.(type) {
	case string:
		return attribute.String(name, v)
	case bool:
		return attribute.Bool(name, v)
	case int:
		return attribute.Int(name, v)
	default:
		return attribute.String(name, fmt.Sprint(v))
	}
}
