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

package perfmon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOpenSuccessful will open query successfully.
func TestOpenSuccessful(t *testing.T) {
	var q Query
	err := q.Open()
	assert.Nil(t, err)
	defer q.Close()
}

// TestAddCounterInvalidArgWhenQueryClosed will check if addcounter func fails when query is closed.
func TestAddCounterInvalidArgWhenQueryClosed(t *testing.T) {
	var q Query
	counter := CounterConfig{Format: "float", InstanceName: "TestInstanceName"}
	queryPath, err := q.ExpandWildCardPath(validQuery)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddCounter(queryPath[0], counter, false)
	assert.EqualValues(t, err, PDH_INVALID_ARGUMENT)
}

// func TestGetFormattedCounterValuesEmptyCounterList will check if getting the counter values will fail when no counter handles are added.
func TestGetFormattedCounterValuesEmptyCounterList(t *testing.T) {
	var q Query
	list, err := q.GetFormattedCounterValues()
	assert.Nil(t, list)
	assert.EqualValues(t, err.Error(), "no counter list found")
}

// TestExpandWildCardPathWithEmptyString will check for a valid path string.
func TestExpandWildCardPathWithEmptyString(t *testing.T) {
	var q Query
	list, err := q.ExpandWildCardPath("")
	assert.Nil(t, list)
	assert.EqualValues(t, err.Error(), "no query path given")
}

// TestSuccessfulQuery retrieves a per counter successfully.
func TestSuccessfulQuery(t *testing.T) {
	var q Query
	err := q.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	counter := CounterConfig{Format: "float", InstanceName: "TestInstanceName"}
	queryPath, err := q.ExpandWildCardPath(validQuery)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddCounter(queryPath[0], counter, false)
	if err != nil {
		t.Fatal(err)
	}
	//Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	err = q.CollectData()
	err = q.CollectData()
	if err != nil {
		t.Fatal(err)
	}
	list, err := q.GetFormattedCounterValues()
	assert.Nil(t, err)
	assert.NotNil(t, list)
}
