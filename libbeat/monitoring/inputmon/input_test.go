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

package inputmon

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestNewInputMonitor(t *testing.T) {
	const (
		inputType = "foo-input"
		id        = "my-id"
	)

	testCases := []struct {
		Input          string
		ID             string
		OptionalParent *monitoring.Registry
		PublicMetrics  bool // Are the metrics registered in the global metric namespace making them public?
	}{
		{Input: inputType, ID: id, PublicMetrics: true},
		{Input: "", ID: id, PublicMetrics: false},
		{Input: inputType, ID: "", PublicMetrics: false},
		{Input: "", ID: "", PublicMetrics: false},

		{Input: inputType, ID: id, OptionalParent: globalRegistry(), PublicMetrics: true},
		{Input: "", ID: id, OptionalParent: globalRegistry(), PublicMetrics: false},
		{Input: inputType, ID: "", OptionalParent: globalRegistry(), PublicMetrics: false},
		{Input: "", ID: "", OptionalParent: globalRegistry(), PublicMetrics: false},

		{Input: inputType, ID: id, OptionalParent: monitoring.NewRegistry(), PublicMetrics: false},
		{Input: "", ID: id, OptionalParent: monitoring.NewRegistry(), PublicMetrics: false},
		{Input: inputType, ID: "", OptionalParent: monitoring.NewRegistry(), PublicMetrics: false},
		{Input: "", ID: "", OptionalParent: monitoring.NewRegistry(), PublicMetrics: false},
	}

	for _, tc := range testCases {
		tc := tc
		testName := fmt.Sprintf("with_id=%v/with_input=%v/custom_parent=%v/public_metrics=%v",
			tc.ID != "", tc.Input != "", tc.OptionalParent != nil, tc.PublicMetrics)

		t.Run(testName, func(t *testing.T) {
			reg, unreg := NewInputRegistry(tc.Input, tc.ID, tc.OptionalParent)
			defer unreg()
			assert.NotNil(t, reg)

			// Verify that metrics are registered when a custom parent registry is given.
			if tc.OptionalParent != nil && tc.OptionalParent != globalRegistry() {
				assert.NotNil(t, tc.OptionalParent.Get(tc.ID))
			}

			// Verify whether the metrics are exposed in the global registry which makes the public.
			parent := globalRegistry().GetRegistry(tc.ID)
			if tc.PublicMetrics {
				assert.NotNil(t, parent)
			} else {
				assert.Nil(t, parent)
			}
		})
	}
}

func TestMetricSnapshotJSON(t *testing.T) {
	require.NoError(t, globalRegistry().Clear())
	t.Cleanup(func() {
		require.NoError(t, globalRegistry().Clear())
	})

	// ============== Input using new API and unique namespace ==============
	// Simulates input using the metrics registry from the v2.Context.
	// It cannot use the v2.Context directly because it creates an import cycle.
	inputID := "input-with-pipeline-metrics-new-inputAPI"
	parent := monitoring.GetNamespace("beat-x").GetRegistry()
	reg := parent.
		NewRegistry(inputID)
	monitoring.NewString(reg, "id").Set(inputID)
	monitoring.NewString(reg, "input").Set("test")
	require.NoError(t, RegisterMetrics(inputID+"-test", reg), "could not register metrics")
	monitoring.NewInt(reg, "foo_total").Set(100)
	monitoring.NewInt(reg, "events_pipeline_total").Set(100)
	defer parent.Remove(inputID)

	// =========== Input using new API and legacy, global namespace ===========
	// Simulates input unaware of the metrics registry from the v2.Context. In
	// that case the parent context used by the v2.Context is the legacy
	// globalRegistry() and the input also registers its metrics directly.
	inputID = "input-with-pipeline-metrics-globalRegistry()"
	reg = globalRegistry().NewRegistry(inputID)
	monitoring.NewString(reg, "id").Set(inputID)
	monitoring.NewString(reg, "input").Set("test")
	// Explicitly register those metrics to be published.
	require.NoError(t, RegisterMetrics(inputID+"-test", reg), "could not register metrics")
	monitoring.NewInt(reg, "events_pipeline_total").Set(200)
	defer globalRegistry().Remove(inputID)

	// now the input also register its metrics with the deprecated
	// NewInputRegistry.
	reg, cancel := NewInputRegistry(
		"test", inputID, nil)
	defer cancel()
	monitoring.NewInt(reg, "foo_total").Set(100)

	// ===== An input registering metrics, but not using the v2.Context =====
	// an input registering metrics on the global namespace. This simulates an
	// input which does not use the metrics registry from the v2.Context.
	inputOldAPI := "input-without-pipeline-metrics"
	reg, cancel = NewInputRegistry(
		"test", inputOldAPI, nil)
	defer cancel()
	monitoring.NewInt(reg, "foo_total").Set(100)

	// ==== registries in the global registries which aren't input metrics ===
	// unrelated registry in the global namespace, should be ignored.
	reg = globalRegistry().NewRegistry("another-registry")
	monitoring.NewInt(reg, "foo3_total").Set(100)
	defer globalRegistry().Remove("another-registry")

	// another input registry missing required information.
	reg = globalRegistry().NewRegistry("yet-another-registry")
	monitoring.NewString(reg, "id").Set("some-id")
	monitoring.NewInt(reg, "foo3_total").Set(100)
	defer globalRegistry().Remove("yet-another-registry")

	jsonBytes, err := MetricSnapshotJSON()
	require.NoError(t, err)

	type Resp struct {
		EventsPipelineTotal int    `json:"events_pipeline_total,omitempty"`
		FooTotal            int    `json:"foo_total"`
		ID                  string `json:"id"`
		Input               string `json:"input"`
	}
	var got []Resp

	err = json.Unmarshal(jsonBytes, &got)
	require.NoError(t, err, "failed to unmarshal response")
	want := map[string]Resp{
		"input-with-pipeline-metrics-new-inputAPI": {
			EventsPipelineTotal: 100,
			FooTotal:            100,
			ID:                  "input-with-pipeline-metrics-new-inputAPI",
			Input:               "test",
		},
		"input-with-pipeline-metrics-globalRegistry()": {
			EventsPipelineTotal: 200,
			FooTotal:            100,
			ID:                  "input-with-pipeline-metrics-globalRegistry()",
			Input:               "test",
		},
		"input-without-pipeline-metrics": {
			FooTotal: 100,
			ID:       "input-without-pipeline-metrics",
			Input:    "test",
		},
	}
	found := map[string]bool{}

	assert.Equal(t, len(want), len(got), "got a different number of metrics than wanted")
	for _, m := range got {
		if found[m.ID] {
			t.Error("found duplicate id")
		}

		w, ok := want[m.ID]
		if !assert.True(t, ok, "unexpected input ID in metrics: %s", m.ID) {
			continue
		}

		if assert.Equal(t, w, m) {
			found[m.ID] = true
		}
	}

	// It's easier to understand the failure with the full output.
	if t.Failed() {
		t.Logf("API reponse:\n%s\n", string(jsonBytes))
	}
}
