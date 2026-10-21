package lab

import (
	"context"
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

type Results = map[string]any

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
	reporter   func(ctx context.Context, trials map[string]Results)
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

func WithReporter[T any](reporter func(ctx context.Context, trials map[string]Results)) Setup[T] {
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

	results := make(map[string]Results, len(enabledTrials))

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
		reporter:   defaultReporter,
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

func defaultReporter(ctx context.Context, trials map[string]Results) {
	traceReporter(ctx, trials)
	logReporter(ctx, trials)
}

func traceReporter(ctx context.Context, trials map[string]Results) {
	tr := otel.Tracer("lab")

	ctx, span := tr.Start(ctx, "experiment")
	defer span.End()

	for _, trial := range trials {

		attrs := make([]attribute.KeyValue, 0, len(trial))
		for name, val := range trial {

			switch v := val.(type) {
			case string:
				attrs = append(attrs, attribute.String(name, v))
			case bool:
				attrs = append(attrs, attribute.Bool(name, v))
			case int:
				attrs = append(attrs, attribute.Int(name, v))
			}
		}
		_, s := tr.Start(ctx, "trial", trace.WithAttributes(attrs...))
		s.End()
	}

}

func logReporter(ctx context.Context, trials map[string]Results) {
	for _, trial := range trials {

		attrs := make([]any, 0, len(trial))
		for name, val := range trial {

			switch v := val.(type) {
			case string:
				attrs = append(attrs, slog.String(name, v))
			case bool:
				attrs = append(attrs, slog.Bool(name, v))
			case int:
				attrs = append(attrs, slog.Int(name, v))
			}
		}
		slog.InfoContext(ctx, "trial results", attrs...)
	}
}
