// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration
// +build !integration

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
		{
			Input:  "2015-01-24T14:06:05.071Z",
			Output: time.Date(2015, time.January, 24, 14, 06, 05, 71*1e6, time.UTC),
		},
		{
			Input:  "2015-03-01T11:19:05.112Z",
			Output: time.Date(2015, time.March, 1, 11, 19, 05, 112*1e6, time.UTC),
		},
		{
			Input:  "2015-02-28T11:19:05.112Z",
			Output: time.Date(2015, time.February, 28, 11, 19, 05, 112*1e6, time.UTC),
		},
	}

	for _, test := range tests {
		result, err := ParseTime(test.Input)
		assert.NoError(t, err)
		assert.Equal(t, test.Output, time.Time(result))
	}
}

func TestParseTimeNegative(t *testing.T) {
	type inputOutput struct {
		Input string
		Err   string
	}

	tests := []inputOutput{
		{
			Input: "2015-02-29TT14:06:05.071Z",
			Err:   "parsing time \"2015-02-29TT14:06:05.071Z\" as \"2006-01-02T15:04:05.000Z\": cannot parse \"T14:06:05.071Z\" as \"15\"",
		},
	}

	for _, test := range tests {
		_, err := ParseTime(test.Input)
		assert.Error(t, err)
		assert.Equal(t, test.Err, err.Error())
	}
}

func TestTimeMarshal(t *testing.T) {
	type inputOutput struct {
		Input  MapStr
		Output string
	}

	tests := []inputOutput{
		{
			Input: MapStr{
				"@timestamp": Time(time.Date(2015, time.March, 01, 11, 19, 05, 112*1e6, time.UTC)),
			},
			Output: `{"@timestamp":"2015-03-01T11:19:05.112Z"}`,
		},
		{
			Input: MapStr{
				"@timestamp": MustParseTime("2015-03-01T11:19:05.112Z"),
				"another":    MustParseTime("2015-03-01T14:19:05.112Z"),
			},
			Output: `{"@timestamp":"2015-03-01T11:19:05.112Z","another":"2015-03-01T14:19:05.112Z"}`,
		},
	}

	for _, test := range tests {
		result, err := json.Marshal(test.Input)
		assert.NoError(t, err)
		assert.Equal(t, test.Output, string(result))
	}
}
