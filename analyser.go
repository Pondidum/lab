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

func DefaultAnalyser[T any](ctx context.Context, baseline, trial *Observation[T]) Analysis[T] {

	analysis := Analysis[T]{
		Name:         trial.Name,
		ErrorsMatch:  baseline.Err == trial.Err,
		ResultsMatch: reflect.DeepEqual(baseline.Result, trial.Result),
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
