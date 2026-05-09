package experiment

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

type Experiment[T any] struct {
	action func(ctx context.Context) (T, error)
}

type options[T any] struct {
	experiments []Experiment[T]
	comparer    func(ctx context.Context, baseline, experiment *Observation[T])
}

type Setup[T any] func(s *options[T])

func WithExperiment[T any](experiment func(ctx context.Context) (T, error)) Setup[T] {
	return func(s *options[T]) {
		s.experiments = append(s.experiments, Experiment[T]{experiment})
	}
}

func WithComparer[T any](compare func(ctx context.Context, baseline, experiment *Observation[T])) Setup[T] {
	return func(s *options[T]) {
		s.comparer = compare
	}
}

func RunExperiment[T any](ctx context.Context, baseline Baseline[T], setup ...Setup[T]) (T, error) {

	opts := &options[T]{}
	for _, apply := range setup {
		apply(opts)
	}

	observations := make([]*Observation[T], 1+len(opts.experiments))
	wg := sync.WaitGroup{}

	wg.Go(func() {
		observations[0] = runAction(ctx, baseline.action)
	})

	for i, exp := range opts.experiments {
		wg.Go(func() {
			observations[i+1] = runAction(ctx, exp.action)
		})
	}

	wg.Wait()

	baseObs := observations[0]
	for _, obs := range observations[1:] {
		opts.comparer(ctx, baseObs, obs)
	}

	return observations[0].Result, observations[0].Err
}
func runAction[T any](ctx context.Context, action func(context.Context) (T, error)) *Observation[T] {
	obs := &Observation[T]{}

	defer func() {
		if p := recover(); p != nil {
			obs.Panic = p
		}
	}()

	obs.Result, obs.Err = action(ctx)

	return obs
}
