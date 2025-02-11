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

	"github.com/elastic/beats/v7/libbeat/beat"
	libbeatmonitoring "github.com/elastic/beats/v7/libbeat/monitoring"
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

	inputID := "input-with-pipeline-metrics"
	r1, cancel1 := NewInputRegistry("test", inputID, nil)
	defer cancel1()
	monitoring.NewInt(r1, "foo1_total").Set(100)

	r2, cancel2 := NewInputRegistry(
		"test", "input-without-pipeline-metrics", nil)
	defer cancel2()
	monitoring.NewInt(r2, "foo2_total").Set(100)

	// this metric should not be reported
	r3 := globalRegistry().NewRegistry("another-registry")
	monitoring.NewInt(r3, "foo3_total").Set(100)

	// this metric should not be reported
	r4 := globalRegistry().NewRegistry("yet-another-registry")
	monitoring.NewString(r4, "id").Set("some-id")
	monitoring.NewInt(r3, "foo3_total").Set(100)

	bInfo := beat.Info{}
	bInfo.Monitoring.Namespace = monitoring.GetNamespace("TestMetricSnapshotJSON")
	intInputReg := bInfo.Monitoring.Namespace.GetRegistry().
		NewRegistry(libbeatmonitoring.RegistryNameInternalInputs).
		NewRegistry(inputID)
	monitoring.NewInt(intInputReg, "events_pipeline_total").Set(100)

	jsonBytes, err := MetricSnapshotJSON(bInfo)
	require.NoError(t, err)

	const expected = `[
  {
    "events_pipeline_total": 100,
    "foo1_total": 100,
    "id": "input-with-pipeline-metrics",
    "input": "test"
  },
  {
    "foo2_total": 100,
    "id": "input-without-pipeline-metrics",
    "input": "test"
  }
]`

	assert.Equal(t, expected, string(jsonBytes))
}
