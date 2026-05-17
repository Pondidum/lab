package lab

import (
	"context"
	"sync"
)

type Observation[T any] struct {
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
	action func(ctx context.Context) (T, error)
}

type options[T any] struct {
	trials   []Trial[T]
	comparer func(ctx context.Context, baseline, trial *Observation[T])
}

type Setup[T any] func(s *options[T])

func WithTrial[T any](trial func(ctx context.Context) (T, error)) Setup[T] {
	return func(s *options[T]) {
		s.trials = append(s.trials, Trial[T]{trial})
	}
}

func WithComparer[T any](compare func(ctx context.Context, baseline, trial *Observation[T])) Setup[T] {
	return func(s *options[T]) {
		s.comparer = compare
	}
}

func RunExperiment[T any](ctx context.Context, baseline Baseline[T], setup ...Setup[T]) (T, error) {

	opts := &options[T]{}
	for _, apply := range setup {
		apply(opts)
	}

	observations := make([]*Observation[T], 1+len(opts.trials))
	wg := sync.WaitGroup{}

	wg.Go(func() {
		observations[0] = &Observation[T]{}
		runAction(ctx, observations[0], baseline.action)
	})

	for i, exp := range opts.trials {
		wg.Go(func() {
			observations[i+1] = &Observation[T]{}
			runAction(ctx, observations[i+1], exp.action)
		})
	}

	wg.Wait()

	baseObs := observations[0]
	for _, obs := range observations[1:] {
		opts.comparer(ctx, baseObs, obs)
	}

	if observations[0].Panic != nil {
		panic(observations[0].Panic)
	}

	return observations[0].Result, observations[0].Err
}

func runAction[T any](ctx context.Context, observation *Observation[T], action func(context.Context) (T, error)) {
	defer func() {
		if p := recover(); p != nil {
			observation.Panic = p
		}
	}()

	observation.Result, observation.Err = action(ctx)
}
