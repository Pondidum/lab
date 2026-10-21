package lab

import (
	"context"
	"reflect"
)

type Analysis[T any] struct {
	Name   string
	Passed bool

	BaselineError string
	TrialError    string
	ErrorsMatch   bool
	TrialPanic    any

	BaselineResult T
	TrialResult    T
	ResultsMatch   bool
	Attributes     map[string]any
}

type Analyser[T any] func(ctx context.Context, baseline, trial *Observation[T]) Analysis[T]

func DefaultAnalyser[T any]() Analyser[T] {
	compare := func(base, trial T) bool {
		return reflect.DeepEqual(base, trial)
	}

	return func(ctx context.Context, baseline, trial *Observation[T]) Analysis[T] {
		return analyse(ctx, compare, baseline, trial)
	}
}

func CustomCompareAnalyser[T any](compare func(base, trial T) bool) Analyser[T] {
	return func(ctx context.Context, baseline, trial *Observation[T]) Analysis[T] {
		return analyse(ctx, compare, baseline, trial)
	}
}

func analyse[T any](ctx context.Context, compare func(base, trial T) bool, baseline, trial *Observation[T]) Analysis[T] {
	analysis := Analysis[T]{
		Name:         trial.Name,
		ErrorsMatch:  baseline.Err == trial.Err,
		ResultsMatch: compare(baseline.Result, trial.Result),
	}
	analysis.Passed = analysis.ResultsMatch && analysis.ErrorsMatch

	if baseline.Err != nil {
		analysis.BaselineError = baseline.Err.Error()
	}
	if trial.Err != nil {
		analysis.TrialError = trial.Err.Error()
	}
	if trial.Panic != nil {
		analysis.TrialPanic = trial.Panic
	}

	if !analysis.ResultsMatch {
		analysis.BaselineResult = baseline.Result
		analysis.TrialResult = trial.Result
	}

	return analysis

}
