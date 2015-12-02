package common

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTime(t *testing.T) {

	type inputOutput struct {
		Input  string
		Output time.Time
	}

	tests := []inputOutput{
		inputOutput{
			Input:  "2015-01-24T14:06:05.071Z",
			Output: time.Date(2015, time.January, 24, 14, 06, 05, 71*1e6, time.UTC),
		},
		inputOutput{
			Input:  "2015-03-01T11:19:05.112Z",
			Output: time.Date(2015, time.March, 1, 11, 19, 05, 112*1e6, time.UTC),
		},
		inputOutput{
			Input:  "2015-02-28T11:19:05.112Z",
			Output: time.Date(2015, time.February, 28, 11, 19, 05, 112*1e6, time.UTC),
		},
		// Golang time pkg happily parses 'wrong' dates like these.
		// Just to have in mind.
		inputOutput{
			Input:  "2015-02-29T11:19:05.112Z",
			Output: time.Date(2015, time.March, 01, 11, 19, 05, 112*1e6, time.UTC),
		},
		inputOutput{
			Input:  "2015-03-31T11:19:05.112Z",
			Output: time.Date(2015, time.March, 31, 11, 19, 05, 112*1e6, time.UTC),
		},
		inputOutput{
			Input:  "2015-04-31T11:19:05.112Z",
			Output: time.Date(2015, time.April, 31, 11, 19, 05, 112*1e6, time.UTC),
		},
	}

	for _, test := range tests {
		result, err := ParseTime(test.Input)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, time.Time(result))
	}
}

func TestParseTimeNegative(t *testing.T) {
	type inputOutput struct {
		Input string
		Err   string
	}

	tests := []inputOutput{
		inputOutput{
			Input: "2015-02-29TT14:06:05.071Z",
			Err:   "parsing time \"2015-02-29TT14:06:05.071Z\" as \"2006-01-02T15:04:05.000Z\": cannot parse \"T14:06:05.071Z\" as \"15\"",
		},
	}

	for _, test := range tests {
		_, err := ParseTime(test.Input)
		assert.NotNil(t, err)
		assert.Equal(t, test.Err, err.Error())
	}
}

func TestTimeMarshal(t *testing.T) {
	type inputOutput struct {
		Input  MapStr
		Output string
	}

	tests := []inputOutput{
		inputOutput{
			Input: MapStr{
				"@timestamp": Time(time.Date(2015, time.March, 01, 11, 19, 05, 112*1e6, time.UTC)),
			},
			Output: `{"@timestamp":"2015-03-01T11:19:05.112Z"}`,
		},
		inputOutput{
			Input: MapStr{
				"@timestamp": MustParseTime("2015-03-01T11:19:05.112Z"),
				"another":    MustParseTime("2015-03-01T14:19:05.112Z"),
			},
			Output: `{"@timestamp":"2015-03-01T11:19:05.112Z","another":"2015-03-01T14:19:05.112Z"}`,
		},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.Input)
		assert.Nil(t, err)
		assert.Equal(t, test.Output, string(result))
	}
}
