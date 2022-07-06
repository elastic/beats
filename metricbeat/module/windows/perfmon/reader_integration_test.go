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

//go:build integration && windows
// +build integration,windows

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewReaderWhenQueryPathNotProvided will check for invalid/no query.
func TestNewReaderWhenQueryPathNotProvided(t *testing.T) {
	config := Config{
		IgnoreNECounters:  false,
		GroupMeasurements: false,
		Queries:           make([]Query, 1),
	}
	config.Queries[0].Name = "Invalid"
	config.Queries[0].Instance = []string{"*"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
	}
	reader, err := NewReader(config)
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.EqualValues(t, err.Error(), `failed to expand counter (query="\Invalid(*)\% Processor Time"): The specified object was not found on the computer.`)
}

// TestNewReaderWithValidQueryPath should successfully instantiate the reader.
func TestNewReaderWithValidQueryPath(t *testing.T) {

	config := Config{
		IgnoreNECounters:  false,
		GroupMeasurements: false,
		Queries:           make([]Query, 1),
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
	}
	reader, err := NewReader(config)
	defer reader.Close()
	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.NotNil(t, reader.query)
	assert.NotNil(t, reader.query.Handle)
	assert.NotNil(t, reader.query.Counters)
	assert.NotZero(t, len(reader.query.Counters))

}

// TestReadSuccessfully will test the func read when it first retrieves no events (and ignored) and then starts retrieving events.
func TestReadSuccessfully(t *testing.T) {
	config := Config{
		IgnoreNECounters:  false,
		GroupMeasurements: false,
		Queries:           make([]Query, 1),
	}
	config.Queries[0].Name = "Processor Information"
	config.Queries[0].Instance = []string{"_Total"}
	config.Queries[0].Counters = []QueryCounter{
		{
			Name: "% Processor Time",
		},
	}
	reader, err := NewReader(config)
	if err != nil {
		t.Fatal(err)
	}
	//Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we call reader.Read() twice.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	events, err := reader.Read()
	assert.NoError(t, err)
	assert.NotNil(t, events)
	assert.NotZero(t, len(events))
}
