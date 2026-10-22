package lab

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	for _, tc := range cases {

		t.Run(tc.Name, func(t *testing.T) {
			analysis := DefaultAnalyser[string]()(t.Context(), tc.Base, tc.Trial)

			assert.Equal(t, tc.Passes, analysis.Passed)
			assert.Equal(t, tc.ResultsMatch, analysis.ResultsMatch)

			if tc.Base.Err != nil || tc.Trial.Err != nil {
				assert.Equal(t, tc.ErrorsMatch, analysis.ErrorsMatch)
			}
		})
	}
}
