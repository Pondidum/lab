package lab

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var truthTable = `
BaseResult,BaseError,TrialResult,TrialError,TrialPass,ResultMatch,ErrorMatch
same,same,,,true,true,true
one,two,,,false,false,false
same,same,base,,false,true,false
same,same,,trial,false,true,false
same,same,match,match,true,true,true
one,two,match,match,false,false,true
`

var errs = map[string]error{
	"":      nil,
	"base":  errors.New("baseline error"),
	"trial": errors.New("trial error"),
	"match": errors.New("matching error"),
}

func TestDefaultComparer(t *testing.T) {

	t.Run("comparing value types", func(t *testing.T) {

		r := csv.NewReader(strings.NewReader(truthTable))

		for {
			record, err := r.Read()
			if err == io.EOF {
				return
			}
			require.NoError(t, err)

			t.Run(strings.Join(record, ","), func(t *testing.T) {

				baseline := &Observation[string]{
					Result: record[0],
					Err:    errs[record[2]],
				}
				trial := &Observation[string]{
					Result: record[1],
					Err:    errs[record[3]],
				}

				trialPasses, _ := strconv.ParseBool(record[4])
				resultsMatch, _ := strconv.ParseBool(record[5])
				errorsMatch, _ := strconv.ParseBool(record[6])

				log := &InMemoryHandler{}
				slog.SetDefault(slog.New(log))
				defaultComparer(t.Context(), baseline, trial)

				attrs := asMap(log.Records[0])

				assert.Equal(t, slog.BoolValue(trialPasses), attrs["trial.passed"], "trial.passed")
				assert.Equal(t, slog.BoolValue(resultsMatch), attrs["trial.results_match"], "trial.results_match")

				if baseline.Err != nil || trial.Err != nil {
					assert.Equal(t, slog.BoolValue(errorsMatch), attrs["trial.errors_match"], "trial.errors_match")
				}
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
