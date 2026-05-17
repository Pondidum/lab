package experiment

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
	expResult1 := thingy{source: "experiment 1"}
	compareRan := false

	result, err := RunExperiment(
		t.Context(),
		WithBaseline(func(ctx context.Context) (thingy, error) {
			return baseResult, nil
		}),
		WithExperiment(func(ctx context.Context) (thingy, error) {
			return expResult1, nil
		}),
		WithComparer(func(ctx context.Context, baseline, experiment *Observation[thingy]) {
			compareRan = true
			require.Equal(t, baseResult, baseline.Result)
			require.Equal(t, expResult1, experiment.Result)
		}),
	)

	require.NoError(t, err)
	require.Equal(t, baseResult, result)
	require.True(t, compareRan)
}

func TestExperimentsRun(t *testing.T) {

	var baseRan, exp1Ran, exp2Ran bool

	result, err := RunExperiment(
		t.Context(),
		WithBaseline(func(ctx context.Context) (string, error) {
			baseRan = true
			return "baseline", nil
		}),
		WithExperiment(func(ctx context.Context) (string, error) {
			exp1Ran = true
			return "exp1", nil
		}),
		WithExperiment(func(ctx context.Context) (string, error) {
			exp2Ran = true
			return "exp2", nil
		}),
		WithComparer(func(ctx context.Context, baseline, experiment *Observation[string]) {
		}),
	)

	require.NoError(t, err)
	require.Equal(t, "baseline", result)

	require.True(t, baseRan)
	require.True(t, exp1Ran)
	require.True(t, exp2Ran)
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

	t.Run("runs for a single experiment", func(t *testing.T) {
		compareRan := false
		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithExperiment(returns(1, "")),
			WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
				require.Equal(t, 0, baseline.Result)
				require.Equal(t, 1, experiment.Result)
				compareRan = true
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
		require.True(t, compareRan)
	})

	t.Run("runs once per experiment", func(t *testing.T) {
		values := []int{}

		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithExperiment(returns(1, "")),
			WithExperiment(returns(2, "")),
			WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
				values = append(values, experiment.Result)
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
		require.Equal(t, []int{1, 2}, values)
	})
}

func TestResults(t *testing.T) {

	t.Run("baseline result is returned", func(t *testing.T) {

		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithExperiment(returns(1, "")),
			WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("baseline error is returned", func(t *testing.T) {
		_, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "err-base")),
			WithExperiment(returns(1, "")),
			WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
			}),
		)
		require.EqualError(t, err, "err-base")
	})

}

func TestPanics(t *testing.T) {

	t.Run("experiment panics are contained", func(t *testing.T) {

		r, err := RunExperiment(t.Context(),
			WithBaseline(returns(0, "")),
			WithExperiment(func(ctx context.Context) (int, error) {
				panic("seen by compare")
			}),
			WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
				require.Equal(t, "seen by compare", experiment.Panic)
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
				WithExperiment(func(ctx context.Context) (int, error) {
					return 1, nil
				}),
				WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
					compareRan = true
					require.Equal(t, "escapes", baseline.Panic)
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
				WithExperiment(func(ctx context.Context) (int, error) {
					return 1, nil
				}),
				WithComparer(func(ctx context.Context, baseline, experiment *Observation[int]) {
					panic("escapes")
				}),
			)
		})
	})
}
