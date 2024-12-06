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

	r, cancel := NewInputRegistry("test", "my-id", nil)
	defer cancel()
	monitoring.NewInt(r, "foo_total").Set(100)

	jsonBytes, err := MetricSnapshotJSON()
	require.NoError(t, err)

	const expected = `[
  {
    "foo_total": 100,
    "id": "my-id",
    "input": "test"
  }
]`

	assert.Equal(t, expected, string(jsonBytes))
}
