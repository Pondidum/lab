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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

type testCase[T any] struct {
	Name string

	Base  *Observation[T]
	Trial *Observation[T]

	Passes       bool
	ResultsMatch bool
	ErrorsMatch  bool
}

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

func parseRow(record []string) testCase[string] {
	tc := testCase[string]{
		Name: strings.Join(record, ","),
		Base: &Observation[string]{
			Result: record[0],
			Err:    errs[record[2]],
		},
		Trial: &Observation[string]{
			Result: record[1],
			Err:    errs[record[3]],
		},
	}

	tc.Passes, _ = strconv.ParseBool(record[4])
	tc.ResultsMatch, _ = strconv.ParseBool(record[5])
	tc.ErrorsMatch, _ = strconv.ParseBool(record[6])

	return tc
}

func testCases(t *testing.T) []testCase[string] {
	t.Helper()

	cases := []testCase[string]{}

	r := csv.NewReader(strings.NewReader(truthTable))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		cases = append(cases, parseRow(record))
	}

	return cases
}

func TestDefaultAnalyser(t *testing.T) {
	cases := testCases(t)

	t.Run("validate log attributes", func(t *testing.T) {

		for _, tc := range cases {

			t.Run(tc.Name, func(t *testing.T) {

				log := &InMemoryHandler{}
				slog.SetDefault(slog.New(log))

				results := DefaultAnalyser[string]()(t.Context(), tc.Base, tc.Trial)
				defaultReporter(t.Context(), []Analysis[string]{results})

				attrs := slogAttributes(log.Records[0])
				assert.Equal(t, slog.BoolValue(tc.Passes), attrs["trial.passed"], "trial.passed")
				assert.Equal(t, slog.BoolValue(tc.ResultsMatch), attrs["trial.results_match"], "trial.results_match")

				if tc.Base.Err != nil || tc.Trial.Err != nil {
					assert.Equal(t, slog.BoolValue(tc.ErrorsMatch), attrs["trial.errors_match"], "trial.errors_match")
				}
			})

		}
	})

	t.Run("validate span attributes", func(t *testing.T) {

		for _, tc := range cases {

			t.Run(tc.Name, func(t *testing.T) {

				exporter := &memoryExporter{}
				tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
				otel.SetTracerProvider(tp)
				ctx, span := tp.Tracer("").Start(t.Context(), "testcase")
				require.True(t, span.IsRecording())

				results := DefaultAnalyser[string]()(ctx, tc.Base, tc.Trial)
				defaultReporter(ctx, []Analysis[string]{results})
				span.End()

				resultSpan := exporter.ByName("trial")
				require.NotNil(t, resultSpan)
				attrs := otelAttributes(*resultSpan)

				assert.Equal(t, attribute.BoolValue(tc.Passes), attrs["trial.passed"], "trial.passed")
				assert.Equal(t, attribute.BoolValue(tc.ResultsMatch), attrs["trial.results_match"], "trial.results_match")

				if tc.Base.Err != nil || tc.Trial.Err != nil {
					assert.Equal(t, attribute.BoolValue(tc.ErrorsMatch), attrs["trial.errors_match"], "trial.errors_match")
				}
			})

		}
	})

}

func slogAttributes(r slog.Record) map[string]slog.Value {
	m := make(map[string]slog.Value, r.NumAttrs())

	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value
		return true
	})
	return m
}

func otelAttributes(span trace.ReadOnlySpan) map[string]attribute.Value {
	attrs := make(map[string]attribute.Value, len(span.Attributes()))

	for _, attr := range span.Attributes() {
		attrs[string(attr.Key)] = attr.Value
	}
	return attrs
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
