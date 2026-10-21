package lab

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func defaultReporter[T any](ctx context.Context, trials map[string]Analysis[T]) {
	traceReporter(ctx, trials)
	logReporter(ctx, trials)
}

func traceReporter[T any](ctx context.Context, trials map[string]Analysis[T]) {
	tr := otel.Tracer("lab")

	ctx, span := tr.Start(ctx, "experiment")
	defer span.End()

	for _, analysis := range trials {
		attrs := []attribute.KeyValue{
			attribute.String("trial.name", analysis.Name),
			attribute.Bool("trial.passed", analysis.Passed),
			attribute.Bool("trial.results_match", analysis.ResultsMatch),
			attribute.Bool("trial.errors_match", analysis.ErrorsMatch),
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

		_, s := tr.Start(ctx, "trial", trace.WithAttributes(attrs...))
		s.End()
	}

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

func logReporter[T any](ctx context.Context, trials map[string]Analysis[T]) {

	for _, analysis := range trials {
		attrs := []any{
			slog.String("trial.name", analysis.Name),
			slog.Bool("trial.passed", analysis.Passed),
			slog.Bool("trial.results_match", analysis.ResultsMatch),
			slog.Bool("trial.errors_match", analysis.ErrorsMatch),
		}

		if analysis.BaselineError != "" {
			attrs = append(attrs, slog.String("trial.error", analysis.BaselineError))
		}
		if analysis.TrialError != "" {
			attrs = append(attrs, slog.String("trial.error", analysis.TrialError))
		}
		if analysis.TrialPanic != nil {
			attrs = append(attrs, slog.String("trial.panic", fmt.Sprint(analysis.TrialPanic)))
		}

		if !analysis.ResultsMatch {
			attrs = append(attrs,
				asSlogAttribute("baseline.result", analysis.BaselineResult),
				asSlogAttribute("trial.result", analysis.TrialResult),
			)
		}

		for k, v := range analysis.Attributes {
			attrs = append(attrs, asSlogAttribute(k, v))
		}

		slog.InfoContext(ctx, "trial results", attrs...)
	}
}

func asSlogAttribute(name string, val any) slog.Attr {
	switch v := val.(type) {
	case string:
		return slog.String(name, v)
	case bool:
		return slog.Bool(name, v)
	case int:
		return slog.Int(name, v)
	default:
		return slog.String(name, fmt.Sprint(v))
	}
}
