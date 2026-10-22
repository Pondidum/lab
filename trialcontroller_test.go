package lab

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrialController(t *testing.T) {

	action := func(result string) (func(ctx context.Context) (string, error), func() bool) {
		ran := false
		check := func() bool { return ran }
		action := func(ctx context.Context) (string, error) {
			ran = true
			return result, nil
		}
		return action, check
	}

	t.Run("no controller specified", func(t *testing.T) {

		base, baseRan := action("base")
		t1, t1Ran := action("trial 1")
		t2, t2Ran := action("trial 2")

		RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(base),
			WithTrial("first", t1),
			WithTrial("second", t2))

		require.True(t, baseRan())
		require.True(t, t1Ran())
		require.True(t, t2Ran())
	})

	t.Run("all trials enabled", func(t *testing.T) {

		base, baseRan := action("base")
		t1, t1Ran := action("trial 1")
		t2, t2Ran := action("trial 2")

		RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(base),
			WithTrial("first", t1),
			WithTrial("second", t2),
			WithTrialController(func(ctx context.Context, trial Trial[string]) bool {
				return true
			}))

		require.True(t, baseRan())
		require.True(t, t1Ran())
		require.True(t, t2Ran())
	})

	t.Run("all trials disabled", func(t *testing.T) {

		base, baseRan := action("base")
		t1, t1Ran := action("trial 1")
		t2, t2Ran := action("trial 2")

		RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(base),
			WithTrial("first", t1),
			WithTrial("second", t2),
			WithTrialController(func(ctx context.Context, trial Trial[string]) bool {
				return false
			}))

		require.True(t, baseRan())
		require.False(t, t1Ran())
		require.False(t, t2Ran())
	})

	t.Run("single trial enabled", func(t *testing.T) {

		base, baseRan := action("base")
		t1, t1Ran := action("trial 1")
		t2, t2Ran := action("trial 2")
		t3, t3Ran := action("trial 3")

		RunExperiment(t.Context(),
			t.Name(),
			WithBaseline(base),
			WithTrial("first", t1),
			WithTrial("second", t2),
			WithTrial("third", t3),
			WithTrialController(func(ctx context.Context, trial Trial[string]) bool {
				return trial.Name() == "second"
			}))

		require.True(t, baseRan())
		require.False(t, t1Ran())
		require.True(t, t2Ran())
		require.False(t, t3Ran())
	})

}
