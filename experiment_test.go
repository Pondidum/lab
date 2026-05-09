package experiment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type thingy struct {
	source string
}

func TestExperiment(t *testing.T) {

	// result, err := RunExperiment[thingy](
	// 	t.Context(),
	// 	WithBaseline(func() {}),
	// 	WithExperiment(func() {}),
	// 	WithExperiment(func() {}),
	// 	WithComparer(func(ctx context.Context, baseline, experiment Observation[thingy]) {

	// 	}),
	// )

	baseResult := thingy{source: "baseline"}
	expResult1 := thingy{source: "experiment 1"}
	compareRan := false

	result, err := RunExperiment[thingy](
		t.Context(),
		WithBaseline(func(ctx context.Context) (thingy, error) {
			return baseResult, nil
		}),
		[]Experiment[thingy]{
			{action: func(ctx context.Context) (thingy, error) {
				return expResult1, nil
			}},
		},
		func(ctx context.Context, baseline, experiment *Observation[thingy]) {
			compareRan = true
			require.Equal(t, baseResult, baseline.Result)
			require.Equal(t, expResult1, experiment.Result)
		},
	)

	require.NoError(t, err)
	require.Equal(t, baseResult, result)
	require.True(t, compareRan)
}
