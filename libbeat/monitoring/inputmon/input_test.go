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
	reg := monitoring.GetNamespace("beat-x").GetRegistry().
		NewRegistry(inputID)
	monitoring.NewString(reg, "id").Set(inputID)
	monitoring.NewString(reg, "input").Set("test")
	require.NoError(t, RegisterMetrics(inputID+"-test", reg), "could not register metrics")
	monitoring.NewInt(reg, "foo1_total").Set(100)
	monitoring.NewInt(reg, "events_pipeline_total").Set(100)

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

	// now the input also register its metrics with the deprecated
	// NewInputRegistry.
	reg, cancel := NewInputRegistry(
		"test", inputID, nil)
	defer cancel()
	monitoring.NewInt(reg, "foo2_total").Set(100)

	// ===== An input registering metrics, but not using the v2.Context =====
	// an input registering metrics on the global namespace. This simulates an
	// input which does not use the metrics registry from the v2.Context.
	inputOldAPI := "input-without-pipeline-metrics"
	reg, cancel = NewInputRegistry(
		"test", inputOldAPI, nil)
	defer cancel()
	monitoring.NewInt(reg, "foo2_total").Set(100)

	// ==== registries in the global registries which aren't input metrics ===
	// unrelated registry in the global namespace, should be ignored.
	reg = globalRegistry().NewRegistry("another-registry")
	monitoring.NewInt(reg, "foo3_total").Set(100)

	// another input registry missing required information.
	reg = globalRegistry().NewRegistry("yet-another-registry")
	monitoring.NewString(reg, "id").Set("some-id")
	monitoring.NewInt(reg, "foo3_total").Set(100)

	jsonBytes, err := MetricSnapshotJSON()
	require.NoError(t, err)

	const expected = `[
  {
    "events_pipeline_total": 100,
    "foo1_total": 100,
    "id": "input-with-pipeline-metrics-new-inputAPI",
    "input": "test"
  },
  {
    "events_pipeline_total": 200,
    "foo2_total": 100,
    "id": "input-with-pipeline-metrics-globalRegistry()",
    "input": "test"
  },
  {
    "foo2_total": 100,
    "id": "input-without-pipeline-metrics",
    "input": "test"
  }
]`

	got := string(jsonBytes)
	assert.Equal(t, expected, got)
	// It's easier to understand the failure with the full output.
	if t.Failed() {
		t.Logf("API reponse:\n%s\n", got)
	}
}
