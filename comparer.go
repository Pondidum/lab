package lab

import (
	"context"
	"errors"
	"reflect"
)

type Comparer[T any] func(ctx context.Context, baseline, trial *Observation[T]) Results

func DefaultComparer[T any](ctx context.Context, baseline, trial *Observation[T]) Results {

	results := Results{}

	if baseline.Err != nil {
		results["baseline.error"] = baseline.Err.Error()
	}
	if trial.Err != nil {
		results["trial.error"] = trial.Err.Error()
	}
	if baseline.Err != nil || trial.Err != nil {
		results["trial.errors_match"] = baseline.Err == trial.Err
	}

	if trial.Panic != nil {
		results["trial.panic"] = trial.Panic
	}

	resultsMatch := reflect.DeepEqual(baseline.Result, trial.Result)
	results["trial.results_match"] = resultsMatch

	if !resultsMatch {
		results["baseline.result"] = baseline.Result
		results["trial.result"] = trial.Result
	}

	errorsMatch := errors.Is(baseline.Err, trial.Err)
	results["trial.passed"] = resultsMatch && errorsMatch
	results["trial.name"] = trial.Name

	return results
}
