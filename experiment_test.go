package lab

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type thingy struct {
	source string
}

func TestExperiment(t *testing.T) {

	baseResult := thingy{source: "baseline"}
	trialResult1 := thingy{source: "trial 1"}
	compareRan := false

	result, err := RunExperiment(
		t.Context(),
		WithBaseline(func(ctx context.Context) (thingy, error) {
			return baseResult, nil
		}),
		WithTrial("trial", func(ctx context.Context) (thingy, error) {
			return trialResult1, nil
		}),
		WithComparer(func(ctx context.Context, baseline, trial *Observation[thingy]) Results {
			compareRan = true
			require.Equal(t, baseResult, baseline.Result)
			require.Equal(t, trialResult1, trial.Result)
			return Results{}
		}),
	)

	require.NoError(t, err)
	require.Equal(t, baseResult, result)
	require.True(t, compareRan)
}

func TestExperimentsRun(t *testing.T) {

	var baseRan, trial1Ran, trial2Ran bool

	result, err := RunExperiment(
		t.Context(),
		WithBaseline(func(ctx context.Context) (string, error) {
			baseRan = true
			return "baseline", nil
		}),
		WithTrial("trial1", func(ctx context.Context) (string, error) {
			trial1Ran = true
			return "trial1", nil
		}),
		WithTrial("trial2", func(ctx context.Context) (string, error) {
			trial2Ran = true
			return "trial2", nil
		}),
	)

	require.NoError(t, err)
	require.Equal(t, "baseline", result)

	require.True(t, baseRan)
	require.True(t, trial1Ran)
	require.True(t, trial2Ran)
}

func returns(val int, err string) func(ctx context.Context) (int, error) {
	return func(ctx context.Context) (int, error) {
		if err == "" {
			return val, nil
		}
		return val, errors.New(err)
	}
}

func TestComparison(t *testing.T) {

	t.Run("a default comparer is used", func(t *testing.T) {
		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithTrial("", returns(1, "")),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("runs for a single trial", func(t *testing.T) {
		compareRan := false
		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithTrial("", returns(1, "")),
			WithComparer(func(ctx context.Context, baseline, trial *Observation[int]) Results {
				require.Equal(t, 0, baseline.Result)
				require.Equal(t, 1, trial.Result)
				compareRan = true
				return Results{}
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
		require.True(t, compareRan)
	})

	t.Run("runs once per trial", func(t *testing.T) {
		values := []int{}

		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithTrial("t1", returns(1, "")),
			WithTrial("t2", returns(2, "")),
			WithComparer(func(ctx context.Context, baseline, trial *Observation[int]) Results {
				values = append(values, trial.Result)
				return Results{}
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
		require.Len(t, values, 2)
		require.Contains(t, values, 1)
		require.Contains(t, values, 2)
	})
}

func TestResults(t *testing.T) {

	t.Run("baseline result is returned", func(t *testing.T) {

		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithTrial("", returns(1, "")),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("baseline error is returned", func(t *testing.T) {
		_, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "err-base")),
			WithTrial("", returns(1, "")),
		)
		require.EqualError(t, err, "err-base")
	})

}

func TestPanics(t *testing.T) {

	t.Run("trial panics are contained", func(t *testing.T) {

		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithTrial("", func(ctx context.Context) (int, error) {
				panic("seen by compare")
			}),
			WithComparer(func(ctx context.Context, baseline, trial *Observation[int]) Results {
				require.Equal(t, "seen by compare", trial.Panic)
				return Results{}
			}),
		)

		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("baseline panics are not contained", func(t *testing.T) {

		compareRan := false
		require.PanicsWithValue(t, "escapes", func() {
			RunExperiment(t.Context(),
				WithBaseline(func(ctx context.Context) (int, error) {
					panic("escapes")
				}),
				WithTrial("", func(ctx context.Context) (int, error) {
					return 1, nil
				}),
				WithComparer(func(ctx context.Context, baseline, trial *Observation[int]) Results {
					compareRan = true
					require.Equal(t, "escapes", baseline.Panic)
					return Results{}
				}),
			)
		})

		require.True(t, compareRan)
	})

	t.Run("comparison panics are not contained", func(t *testing.T) {

		require.PanicsWithValue(t, "escapes", func() {
			RunExperiment(t.Context(),
				WithBaseline(func(ctx context.Context) (int, error) {
					return 0, nil
				}),
				WithTrial("", func(ctx context.Context) (int, error) {
					return 1, nil
				}),
				WithComparer(func(ctx context.Context, baseline, trial *Observation[int]) Results {
					panic("escapes")
				}),
			)
		})
	})
}
