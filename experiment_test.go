package lab

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

type thingy struct {
	source string
}

func TestExperiment(t *testing.T) {
	ctx, span, exporter := CreateTracer(t)

	baseResult := thingy{source: "baseline"}
	trialResult1 := thingy{source: "trial 1"}
	compareRan := false

	result, err := RunExperiment(
		ctx,
		t.Name(),
		WithBaseline(func(ctx context.Context) (thingy, error) {
			return baseResult, nil
		}),
		WithTrial("trial", func(ctx context.Context) (thingy, error) {
			return trialResult1, nil
		}),
		WithAnalyser(func(ctx context.Context, baseline, trial *Observation[thingy]) Analysis[thingy] {
			compareRan = true
			require.Equal(t, baseResult, baseline.Result)
			require.Equal(t, trialResult1, trial.Result)
			return Analysis[thingy]{}
		}),
		WithContext[thingy](attribute.Int("priority", 5)),
	)

	require.NoError(t, err)
	require.Equal(t, baseResult, result)
	require.True(t, compareRan)

	span.End()

	attrs := exporter.AttributesFor("experiment")
	require.Equal(t, attribute.StringValue(t.Name()), attrs["experiment.name"])
	require.Equal(t, attribute.IntValue(5), attrs["priority"])
}

func TestExperimentsRun(t *testing.T) {

	var baseRan, trial1Ran, trial2Ran bool

	result, err := RunExperiment(
		t.Context(),
		t.Name(),
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
			t.Name(),
			WithBaseline(returns(0, "")),
			WithTrial("", returns(1, "")),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("runs for a single trial", func(t *testing.T) {
		compareRan := false
		r, err := RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(returns(0, "")),
			WithTrial("", returns(1, "")),
			WithAnalyser(func(ctx context.Context, baseline, trial *Observation[int]) Analysis[int] {
				require.Equal(t, 0, baseline.Result)
				require.Equal(t, 1, trial.Result)
				compareRan = true
				return Analysis[int]{}
			}),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
		require.True(t, compareRan)
	})

	t.Run("runs once per trial", func(t *testing.T) {
		values := []int{}

		r, err := RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(returns(0, "")),
			WithTrial("t1", returns(1, "")),
			WithTrial("t2", returns(2, "")),
			WithAnalyser(func(ctx context.Context, baseline, trial *Observation[int]) Analysis[int] {
				values = append(values, trial.Result)
				return Analysis[int]{}
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
			t.Name(),
			WithBaseline(returns(0, "")),
			WithTrial("", returns(1, "")),
		)
		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("baseline error is returned", func(t *testing.T) {
		_, err := RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(returns(0, "err-base")),
			WithTrial("", returns(1, "")),
		)
		require.EqualError(t, err, "err-base")
	})

}

func TestPanics(t *testing.T) {

	t.Run("trial panics are contained", func(t *testing.T) {

		r, err := RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(returns(0, "")),
			WithTrial("", func(ctx context.Context) (int, error) {
				panic("seen by compare")
			}),
			WithAnalyser(func(ctx context.Context, baseline, trial *Observation[int]) Analysis[int] {
				require.Equal(t, "seen by compare", trial.Panic)
				return Analysis[int]{}
			}),
		)

		require.NoError(t, err)
		require.Equal(t, 0, r)
	})

	t.Run("baseline panics are not contained", func(t *testing.T) {

		compareRan := false
		require.PanicsWithValue(t, "escapes", func() {
			RunExperiment(t.Context(),
				t.Name(),
				WithBaseline(func(ctx context.Context) (int, error) {
					panic("escapes")
				}),
				WithTrial("", func(ctx context.Context) (int, error) {
					return 1, nil
				}),
				WithAnalyser(func(ctx context.Context, baseline, trial *Observation[int]) Analysis[int] {
					compareRan = true
					require.Equal(t, "escapes", baseline.Panic)
					return Analysis[int]{}
				}),
			)
		})

		require.True(t, compareRan)
	})

	t.Run("comparison panics are not contained", func(t *testing.T) {

		require.PanicsWithValue(t, "escapes", func() {
			RunExperiment(t.Context(),
				t.Name(),
				WithBaseline(func(ctx context.Context) (int, error) {
					return 0, nil
				}),
				WithTrial("", func(ctx context.Context) (int, error) {
					return 1, nil
				}),
				WithAnalyser(func(ctx context.Context, baseline, trial *Observation[int]) Analysis[int] {
					panic("escapes")
				}),
			)
		})
	})
}
