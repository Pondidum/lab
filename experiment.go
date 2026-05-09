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

type Experiment[T any] struct {
	action func(ctx context.Context) (T, error)
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

func RunExperiment[T any](ctx context.Context, baseline Baseline[T], experiments []Experiment[T], comparer func(ctx context.Context, baseline, experiment *Observation[T])) (T, error) {

	observations := make([]*Observation[T], 1+len(experiments))

	wg := sync.WaitGroup{}

	wg.Go(func() {
		observations[0] = runAction(ctx, baseline.action)
	})

	for i, exp := range experiments {
		wg.Go(func() {
			observations[i+1] = runAction(ctx, exp.action)
		})
	}

	wg.Wait()

	baseObs := observations[0]
	for _, obs := range observations[1:] {
		comparer(ctx, baseObs, obs)
	}

	return observations[0].Result, observations[0].Err
}
