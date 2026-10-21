package lab

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Observation[T any] struct {
	Name   string
	Result T
	Err    error
	Panic  any
}

type Baseline[T any] struct {
	action func(ctx context.Context) (T, error)
}

func WithBaseline[T any](baseline func(ctx context.Context) (T, error)) Baseline[T] {
	return Baseline[T]{
		action: baseline,
	}
}

type Trial[T any] struct {
	name   string
	action func(ctx context.Context) (T, error)
}

func (t *Trial[T]) Name() string { return t.name }

type options[T any] struct {
	trials     []Trial[T]
	controller func(ctx context.Context, trial Trial[T]) bool
	analyse    Analyser[T]
	reporter   func(ctx context.Context, trials map[string]Analysis[T])
}

type Setup[T any] func(s *options[T])

func WithTrial[T any](name string, trial func(ctx context.Context) (T, error)) Setup[T] {
	return func(s *options[T]) {
		s.trials = append(s.trials, Trial[T]{name, trial})
	}
}

func WithAnalyser[T any](analyse Analyser[T]) Setup[T] {
	return func(s *options[T]) {
		s.analyse = analyse
	}
}

func WithReporter[T any](reporter func(ctx context.Context, trials map[string]Analysis[T])) Setup[T] {
	return func(s *options[T]) {
		s.reporter = reporter
	}
}

func WithTrialController[T any](controller func(ctx context.Context, trial Trial[T]) bool) Setup[T] {
	return func(s *options[T]) {
		s.controller = controller
	}
}

func RunExperiment[T any](ctx context.Context, baseline Baseline[T], setup ...Setup[T]) (T, error) {

	opts := defaultOptions[T]()
	for _, apply := range setup {
		apply(opts)
	}

	baseObs := &Observation[T]{Name: "baseline"}
	observations := make([]*Observation[T], len(opts.trials))
	wg := sync.WaitGroup{}

	wg.Go(func() {
		runAction(ctx, baseObs, baseline.action)
	})

	enabledTrials := make(map[int]Trial[T], len(opts.trials))
	for i, trial := range opts.trials {
		if opts.controller(ctx, trial) {
			enabledTrials[i] = trial
		}
	}

	for i, exp := range enabledTrials {
		wg.Go(func() {
			observations[i] = &Observation[T]{Name: exp.name}
			runAction(ctx, observations[i], exp.action)
		})
	}

	wg.Wait()

	results := make(map[string]Analysis[T], len(enabledTrials))

	for i := range enabledTrials {
		obs := observations[i]
		results[obs.Name] = opts.analyse(ctx, baseObs, obs)
	}

	opts.reporter(ctx, results)

	if baseObs.Panic != nil {
		panic(baseObs.Panic)
	}

	return baseObs.Result, baseObs.Err
}

func defaultOptions[T any]() *options[T] {
	return &options[T]{
		analyse:    DefaultAnalyser[T],
		controller: func(ctx context.Context, trial Trial[T]) bool { return true },
		reporter:   defaultReporter[T],
	}
}

func runAction[T any](ctx context.Context, observation *Observation[T], action func(context.Context) (T, error)) {
	defer func() {
		if p := recover(); p != nil {
			observation.Panic = p
		}
	}()

	observation.Result, observation.Err = action(ctx)
}

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
