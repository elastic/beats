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

var asTransformablesTests = []struct {
	name                   string
	message                string
	allowStringArray       bool
	expectedTransformables int
	expectStatusUpdates    bool
	statusUpdate           status.Status
	expectLogs             bool
}{
	{
		name:                   "string_array_allowed",
		message:                `["123456789abcdefgh8866123","123456789zxcvbnmas8a8q60"]`,
		allowStringArray:       true,
		expectedTransformables: 0,
		expectStatusUpdates:    false,
		expectLogs:             false,
	},
	{
		name:                   "single_string_allowed",
		message:                `["123456789abcdefgh8866123"]`,
		allowStringArray:       true,
		expectedTransformables: 0,
		expectStatusUpdates:    false,
		expectLogs:             false,
	},
	{
		name:                   "number_array_allowed",
		message:                `[1, 2]`,
		allowStringArray:       true,
		expectedTransformables: 0,
		expectStatusUpdates:    false,
		expectLogs:             false,
	},
	{
		name:                   "mixed_strings_and_objects_degrades",
		message:                `["123456789abcdefgh8866123", {"text": "123456789zxcvbnmas8a8q60"}, {"text": "4853489589345y8934"}]`,
		allowStringArray:       true,
		expectedTransformables: 2,
		expectStatusUpdates:    true,
		expectLogs:             true,
		statusUpdate:           status.Degraded,
	},
	{
		name:                   "mixed_objects_and_strings_degrades",
		message:                `[{"text": "123456789zxcvbnmas8a8q60"}, "123456789abcdefgh8866123", {"text": "4853489589345y8934"}]`,
		allowStringArray:       true,
		expectedTransformables: 2,
		expectStatusUpdates:    true,
		expectLogs:             true,
		statusUpdate:           status.Degraded,
	},
	{
		name:                   "string_array_not_allowed_degrades",
		message:                `["123456789abcdefgh8866123","123456789zxcvbnmas8a8q60"]`,
		allowStringArray:       false,
		expectedTransformables: 0,
		expectStatusUpdates:    true,
		expectLogs:             true,
		statusUpdate:           status.Degraded,
	},
	{
		name:                   "single_string_not_allowed_degrades",
		message:                `["123456789abcdefgh8866123"]`,
		allowStringArray:       false,
		expectedTransformables: 0,
		expectStatusUpdates:    true,
		expectLogs:             true,
		statusUpdate:           status.Degraded,
	},
	{
		name:                   "number_array_not_allowed_degrades",
		message:                `[1, 2]`,
		allowStringArray:       false,
		expectedTransformables: 0,
		expectStatusUpdates:    true,
		expectLogs:             true,
		statusUpdate:           status.Degraded,
	},
	{
		name:                   "object_response",
		message:                `{"response":{"empty":[]}}`,
		allowStringArray:       false,
		expectedTransformables: 1,
		expectStatusUpdates:    false,
		expectLogs:             false,
	},
}

func TestAsTransformables(t *testing.T) {
	for _, test := range asTransformablesTests {
		t.Run(test.name, func(t *testing.T) {
			var data interface{}
			err := json.Unmarshal([]byte(test.message), &data)
			if err != nil {
				t.Fatalf("error unmarshalling json: %s", err)
			}

			resp := &response{
				page: 1,
				url:  *(newURL("http://test?p1=v1")),
				header: http.Header{
					"Authorization": []string{"Bearer token"},
				},
				body: data,
			}

			stat := &testStatusReporter{}

			logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
			transformables := resp.asTransformables(stat, logger, test.allowStringArray)

			if test.expectStatusUpdates {
				assert.NotEmpty(t, stat.updates(), "expected status updates but got none")
				if len(stat.updates()) > 0 {
					assert.Equal(t, test.statusUpdate, stat.updates()[0].state,
						"status update does not match: got %v, want %v", stat.updates()[0].state, test.statusUpdate)
				}
			} else {
				assert.Empty(t, stat.updates(), "expected no status updates")
			}

			allLogs := logs.All()
			if test.expectLogs {
				assert.NotEmpty(t, allLogs, "expected logs but got none")
			} else {
				assert.Empty(t, allLogs, "expected no logs")
			}

			assert.Len(t, transformables, test.expectedTransformables, "unexpected number of transformables")
		})
	}
}

type testStatusReporter struct {
	mu      sync.RWMutex
	entries []statusUpdate
}

func (r *testStatusReporter) UpdateStatus(s status.Status, msg string) {
	r.mu.Lock()
	r.entries = append(r.entries, statusUpdate{s, msg})
	r.mu.Unlock()
}

func (r *testStatusReporter) updates() []statusUpdate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]statusUpdate{}, r.entries...)
}

type statusUpdate struct {
	state status.Status
	msg   string
}
