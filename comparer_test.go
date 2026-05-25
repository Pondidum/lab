package lab

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultComparer(t *testing.T) {

	t.Run("comparing value types", func(t *testing.T) {

		commonErr := errors.New("common error")
		cases := map[string]struct {
			BaseResult  string
			BaseErr     error
			TrialResult string
			TrialErr    error
			Expected    map[string]slog.Value
		}{
			"matching": {
				BaseResult:  "same",
				TrialResult: "same",
				Expected: map[string]slog.Value{
					"trial.passed": slog.BoolValue(true),
				},
			},
			"differing": {
				BaseResult:  "same",
				TrialResult: "different",
				Expected: map[string]slog.Value{
					"trial.passed":    slog.BoolValue(false),
					"baseline.result": slog.StringValue("same"),
					"trial.result":    slog.StringValue("different"),
				},
			},
			"baseline error": {
				BaseResult:  "same",
				BaseErr:     errors.New("baseline error"),
				TrialResult: "same",
				Expected: map[string]slog.Value{
					"trial.passed":   slog.BoolValue(false),
					"baseline.error": slog.StringValue("baseline error"),
				},
			},
			"trial error": {
				BaseResult:  "same",
				TrialResult: "same",
				TrialErr:    errors.New("trial error"),
				Expected: map[string]slog.Value{
					"trial.passed": slog.BoolValue(false),
					"trial.error":  slog.StringValue("trial error"),
				},
			},
			"both error, same": {
				BaseResult:  "same",
				TrialResult: "same",
				BaseErr:     commonErr,
				TrialErr:    commonErr,
				Expected: map[string]slog.Value{
					"trial.passed":       slog.BoolValue(true),
					"baseline.error":     slog.StringValue(commonErr.Error()),
					"trial.error":        slog.StringValue(commonErr.Error()),
					"trial.errors_match": slog.BoolValue(true),
				},
			},
			"both error, different": {
				BaseResult:  "same",
				TrialResult: "same",
				BaseErr:     errors.New("baseline error"),
				TrialErr:    errors.New("trial error"),
				Expected: map[string]slog.Value{
					"trial.passed":       slog.BoolValue(false),
					"baseline.error":     slog.StringValue("baseline error"),
					"trial.error":        slog.StringValue("trial error"),
					"trial.errors_match": slog.BoolValue(false),
				},
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				baseline := &Observation[string]{Result: tc.BaseResult, Err: tc.BaseErr}
				trial := &Observation[string]{Result: tc.TrialResult, Err: tc.TrialErr}

				log := &InMemoryHandler{}
				slog.SetDefault(slog.New(log))
				defaultComparer(t.Context(), baseline, trial)

				attrs := asMap(log.Records[0])
				assert.Equal(t, tc.Expected, attrs)
			})
		}
	})

}

func asMap(r slog.Record) map[string]slog.Value {
	m := make(map[string]slog.Value, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value
		return true
	})
	return m
}

var _ slog.Handler = &InMemoryHandler{}

type InMemoryHandler struct {
	Records []slog.Record
}

func (h *InMemoryHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *InMemoryHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *InMemoryHandler) WithGroup(name string) slog.Handler       { return h }
func (h *InMemoryHandler) Handle(ctx context.Context, r slog.Record) error {
	h.Records = append(h.Records, r)
	return nil
}
