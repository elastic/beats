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

package elastic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/mb"
)

func TestMakeXPackMonitoringIndexName(t *testing.T) {
	tests := []struct {
		Name     string
		Product  Product
		Expected string
	}{
		{
			"Elasticsearch monitoring index",
			Elasticsearch,
			".monitoring-es-7-mb",
		},
		{
			"Kibana monitoring index",
			Kibana,
			".monitoring-kibana-7-mb",
		},
		{
			"Logstash monitoring index",
			Logstash,
			".monitoring-logstash-7-mb",
		},
		{
			"Beats monitoring index",
			Beats,
			".monitoring-beats-7-mb",
		},
	}

	for _, test := range tests {
		name := fmt.Sprintf("Test naming %v", test.Name)
		t.Run(name, func(t *testing.T) {
			indexName := MakeXPackMonitoringIndexName(test.Product)
			assert.Equal(t, test.Expected, indexName)
		})
	}
}

type MockReporterV2 struct {
	mb.ReporterV2
}

func (MockReporterV2) Event(event mb.Event) bool {
	return true
}

var currentErr error // This hack is necessary because the Error method below cannot receive the type *MockReporterV2

func (m MockReporterV2) Error(err error) bool {
	currentErr = err
	return true
}

func TestReportErrorForMissingField(t *testing.T) {
	field := "some.missing.field"
	r := MockReporterV2{}
	err := ReportErrorForMissingField(field, Elasticsearch, r)

	expectedError := fmt.Errorf("Could not find field '%v' in Elasticsearch stats API response", field)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, expectedError, currentErr)
}

func TestFixTimestampField(t *testing.T) {
	tests := []struct {
		Name          string
		OriginalValue map[string]interface{}
		ExpectedValue map[string]interface{}
	}{
		{
			"converts float64s in scientific notation to ints",
			map[string]interface{}{
				"foo": 1.571284349E12,
			},
			map[string]interface{}{
				"foo": 1571284349000,
			},
		},
		{
			"converts regular notation float64s to ints",
			map[string]interface{}{
				"foo": float64(1234),
			},
			map[string]interface{}{
				"foo": 1234,
			},
		},
		{
			"ignores missing fields",
			map[string]interface{}{
				"bar": 12345,
			},
			map[string]interface{}{
				"bar": 12345,
			},
		},
		{
			"leaves strings untouched",
			map[string]interface{}{
				"foo": "bar",
			},
			map[string]interface{}{
				"foo": "bar",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := FixTimestampField(test.OriginalValue, "foo")
			assert.NoError(t, err)
			assert.Equal(t, test.ExpectedValue, test.OriginalValue)
		})
	}
}
