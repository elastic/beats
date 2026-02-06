// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestTemplateValues(t *testing.T) {
	resp := &response{
		page: 1,
		url:  *(newURL("http://test?p1=v1")),
		header: http.Header{
			"Authorization": []string{"Bearer token"},
		},
		body: mapstr.M{
			"param": "value",
		},
	}

	vals := resp.templateValues()

	assert.Equal(t, resp.page, vals["page"])
	v, _ := vals.GetValue("url.value")
	assert.Equal(t, resp.url.String(), v)
	v, _ = vals.GetValue("url.params")
	assert.EqualValues(t, resp.url.Query(), v)
	assert.EqualValues(t, resp.header, vals["header"])
	assert.EqualValues(t, resp.body, vals["body"])

	resp = nil

	vals = resp.templateValues()

	assert.NotNil(t, vals)
	assert.Equal(t, 0, len(vals))
}

func TestTransformable(t *testing.T) {
	tests := []struct {
		name                   string
		message                string
		expectedTransformables int
		expectStatusUpdates    bool
		statusUpdate           status.Status
		expectLogs             bool
		allowStrings           bool
	}{
		{
			name:                   "array_of_strings_allowed",
			message:                `["123456789abcdefgh8866123","123456789zxcvbnmas8a8q60"]`,
			expectedTransformables: 0,
			expectStatusUpdates:    false,
			expectLogs:             false,
			allowStrings:           true,
		},
		{
			name:                   "array_of_1_string_allowed",
			message:                `["123456789abcdefgh8866123"]`,
			expectedTransformables: 0,
			expectStatusUpdates:    false,
			expectLogs:             false,
			allowStrings:           true,
		},
		{
			name:                   "array_of_mixed_strings_and_json_objects_should_cause_the_status_to_degrade",
			message:                `["123456789abcdefgh8866123", { "text": "123456789zxcvbnmas8a8q60"}, { "text": "4853489589345y8934"}]`,
			expectedTransformables: 2,
			expectStatusUpdates:    true,
			expectLogs:             true,
			statusUpdate:           status.Degraded,
			allowStrings:           true,
		},
		{
			name:                   "array_of_ints_should_cause_the_status_to_degrade",
			message:                `[1, 2]`,
			expectedTransformables: 0,
			expectStatusUpdates:    true,
			expectLogs:             true,
			statusUpdate:           status.Degraded,
			allowStrings:           true,
		},
		{
			name:                   "array_of_mixed_json_objects_and_strings_should_cause_the_status_to_degrade",
			message:                `[ {"text": "123456789zxcvbnmas8a8q60"}, "123456789abcdefgh8866123",{ "text": "4853489589345y8934"}]`,
			expectedTransformables: 2,
			expectStatusUpdates:    true,
			expectLogs:             true,
			statusUpdate:           status.Degraded,
			allowStrings:           true,
		},
		{
			name:                   "empty_array_should_be_ignored",
			message:                `{"response":{"empty":[]}}`,
			expectedTransformables: 1,
			expectStatusUpdates:    false,
			expectLogs:             false,
			allowStrings:           true,
		},
		{
			name:                   "array_of_strings_causes_degrade",
			message:                `["123456789abcdefgh8866123","123456789zxcvbnmas8a8q60"]`,
			expectedTransformables: 0,
			expectStatusUpdates:    true,
			expectLogs:             true,
			statusUpdate:           status.Degraded,
			allowStrings:         false,
		},
		{
			name:                   "array_of_1_string_causes_degrade",
			message:                `["123456789abcdefgh8866123"]`,
			expectedTransformables: 0,
			expectStatusUpdates:    true,
			expectLogs:             true,
			statusUpdate:           status.Degraded,
			allowStrings:           false,
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data interface{}
			err := json.Unmarshal([]byte(tt.message), &data)
			if err != nil {
				t.Fatalf("Error unmarshalling json: %s", err)
			}

			resp := &response{
				page: 1,
				url:  *(newURL("http://test?p1=v1")),
				header: http.Header{
					"Authorization": []string{"Bearer token"},
				},
				body: data,
			}

			stat := &mockStatusReporter{}

			logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
			transformables := resp.asTransformables(stat, logger, tt.allowStrings)

			if tt.expectStatusUpdates {
				assert.NotEmpty(t, stat.GetUpdates(), "expected status updates but got none")
				if len(stat.GetUpdates()) > 0 {
					if tt.statusUpdate != stat.GetUpdates()[0].state {
						t.Errorf("status update does not match the expected status update got %v, want %v", stat.GetUpdates()[0], tt.statusUpdate)
					}
				} else {
					t.Errorf("status update expected want %v ", tt.statusUpdate)
				}
			} else {
				assert.Empty(t, stat.GetUpdates(), "expected no status updates")
			}

			allLogs := logs.All()
			if tt.expectLogs {
				assert.NotEmpty(t, allLogs, "expected logs but got none")
			} else {
				assert.Empty(t, allLogs, "expected no logs")
			}

			assert.Len(t, transformables, tt.expectedTransformables, "unexpected number of transformables")
		})
	}
}

type mockStatusReporter struct {
	mutex   sync.RWMutex
	updates []statusUpdate
}

func (m *mockStatusReporter) UpdateStatus(status status.Status, msg string) {
	m.mutex.Lock()
	m.updates = append(m.updates, statusUpdate{status, msg})
	m.mutex.Unlock()
}

func (m *mockStatusReporter) GetUpdates() []statusUpdate {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return append([]statusUpdate{}, m.updates...)
}

type statusUpdate struct {
	state status.Status
	msg   string
}
