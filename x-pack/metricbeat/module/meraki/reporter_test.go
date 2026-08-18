// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package meraki

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "Nil",
			input:    nil,
			expected: true,
		},
		{
			name:     "Nil pointer",
			input:    (*int)(nil),
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "Non-empty string",
			input:    "test",
			expected: false,
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: true,
		},
		{
			name:     "Regular value",
			input:    float64(1.2),
			expected: false,
		},
		{
			name:     "Pointer to int",
			input:    func() *int { i := 42; return &i }(),
			expected: false,
		},
		{
			name:     "Pointer to bool",
			input:    func() *bool { b := false; return &b }(),
			expected: false,
		},
		{
			name:     "Boolean false",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("isEmpty(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

type fakeReporter struct {
	events []mb.Event
}

func (f *fakeReporter) Event(e mb.Event) bool {
	f.events = append(f.events, e)
	return true
}
func (f *fakeReporter) Error(e error) bool { return true }

func TestReportMetricsForOrganization(t *testing.T) {
	reporter := &fakeReporter{}
	metrics := []mapstr.M{
		{"foo": "bar"},
		{"empty": ""},
		{"nil": (*float64)(nil)},
		{"@timestamp": "2024-06-17T12:00:00Z", "has_timestamp": true},
		{"@timestamp": "invalid", "invalid_timestamp": true},
	}

	ReportMetricsForOrganization(reporter, "123", metrics)
	assert.Equal(t, 5, len(reporter.events), "expected 5 events")

	for _, e := range reporter.events {
		// Check empty fields are not included
		assert.NotContains(t, e.MetricSetFields, "empty", "expected 'empty' field to be excluded")
		assert.NotContains(t, e.MetricSetFields, "nil", "expected 'nil' field to be excluded")

		// Check that organization_id is present
		assert.Equal(t, "123", e.ModuleFields["organization_id"], "expected organization_id to be '123'")

		// Check that @timestamp is parsed correctly
		if ts, ok := e.MetricSetFields["has_timestamp"]; ok {
			assert.Equal(t, ts, e.ModuleFields["@timestamp"], "expected @timestamp to be '2024-06-17T12:00:00Z'")
		}

		// Check that invalid timestamp is not added
		if _, ok := e.MetricSetFields["invalid_timestamp"]; ok {
			_, ok := e.MetricSetFields["@timestamp"]
			assert.False(t, ok, "expected @timestamp to not be present for invalid timestamp")
			assert.Equal(t, "invalid", e.MetricSetFields["@timestamp"], "expected invalid timestamp to remain in the event")
		}
	}
}
