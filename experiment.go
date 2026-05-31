package lab

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
	comparer   func(ctx context.Context, baseline, trial *Observation[T])
}

type Setup[T any] func(s *options[T])

func WithTrial[T any](name string, trial func(ctx context.Context) (T, error)) Setup[T] {
	return func(s *options[T]) {
		s.trials = append(s.trials, Trial[T]{name, trial})
	}
}

func WithComparer[T any](compare func(ctx context.Context, baseline, trial *Observation[T])) Setup[T] {
	return func(s *options[T]) {
		s.comparer = compare
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

	observations := make([]*Observation[T], 1+len(opts.trials))
	wg := sync.WaitGroup{}

	wg.Go(func() {
		observations[0] = &Observation[T]{Name: "baseline"}
		runAction(ctx, observations[0], baseline.action)
	})

	enabledTrials := make(map[int]Trial[T], len(opts.trials))
	for i, trial := range opts.trials {
		if opts.controller(ctx, trial) {
			enabledTrials[i] = trial
		}
	}

	for i, exp := range enabledTrials {
		wg.Go(func() {
			observations[i+1] = &Observation[T]{Name: exp.name}
			runAction(ctx, observations[i+1], exp.action)
		})
	}

	wg.Wait()

	baseObs := observations[0]
	if len(enabledTrials) > 0 {

		for i := range enabledTrials {
			obs := observations[i+1]
			opts.comparer(ctx, baseObs, obs)
		}
	}

	if baseObs.Panic != nil {
		panic(baseObs.Panic)
	}

	return baseObs.Result, baseObs.Err
}

func defaultOptions[T any]() *options[T] {
	return &options[T]{
		comparer:   defaultComparer[T],
		controller: func(ctx context.Context, trial Trial[T]) bool { return true },
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

func defaultComparer[T any](ctx context.Context, baseline, trial *Observation[T]) {

	attrs := []any{}

	if baseline.Err != nil {
		attrs = append(attrs, slog.String("baseline.error", baseline.Err.Error()))
	}
	if trial.Err != nil {
		attrs = append(attrs, slog.String("trial.error", trial.Err.Error()))
	}
	if baseline.Err != nil || trial.Err != nil {
		attrs = append(attrs, slog.Bool("trial.errors_match", baseline.Err == trial.Err))
	}

	if trial.Panic != nil {
		attrs = append(attrs, slog.Any("trial.panic", trial.Panic))
	}

	resultsMatch := cmp.Equal(baseline.Result, trial.Result, cmpopts.IgnoreUnexported())
	attrs = append(attrs, slog.Bool("trial.results_match", resultsMatch))

	if !resultsMatch {
		attrs = append(attrs,
			slog.Any("baseline.result", baseline.Result),
			slog.Any("trial.result", trial.Result),
		)
	}

	errorsMatch := errors.Is(baseline.Err, trial.Err)
	attrs = append(attrs, slog.Bool("trial.passed", resultsMatch && errorsMatch))
	attrs = append(attrs, slog.String("trial.name", trial.Name))

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(asOtelAttributes(attrs)...)

	slog.InfoContext(ctx, "trial results", attrs...)
}

func asOtelAttributes(attrs []any) []attribute.KeyValue {
	otel := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		attr, ok := a.(slog.Attr)
		if !ok {
			continue
		}

		switch attr.Value.Kind() {
		case slog.KindString:
			otel = append(otel, attribute.String(attr.Key, attr.Value.Resolve().String()))
		case slog.KindBool:
			otel = append(otel, attribute.Bool(attr.Key, attr.Value.Resolve().Bool()))
		}
	}
	return otel
}
