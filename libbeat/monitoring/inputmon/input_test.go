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

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
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
	log := logptest.NewTestingLogger(t, "TestMetricSnapshotJSON")

	// ============== Input using new API and unique namespace ==============
	// Simulates input using the metrics registry from the v2.Context.
	// It cannot use the v2.Context directly because it creates an import cycle.
	inputID := "input-with-pipeline-metrics-new-inputAPI"
	inputType := "test"

	parentLocalReg := monitoring.GetNamespace("beat-x").GetRegistry()
	reg := NewMetricsRegistry(inputID, inputType, parentLocalReg, log)
	monitoring.NewInt(reg, "foo_total").Set(10)
	monitoring.NewInt(reg, "events_pipeline_total").Set(10)

	// simulate a duplicated ID in the local and global namespace.
	reg = globalRegistry().NewRegistry(inputID)
	monitoring.NewString(reg, "id").Set(inputID)
	monitoring.NewString(reg, "input").Set(inputType)
	monitoring.NewBool(reg, "should_be_overwritten").Set(true)

	// =========== Input using new API and legacy, global namespace ===========
	// Simulates input unaware of the metrics registry from the v2.Context. In
	// that case the parentLocalReg context used by the v2.Context is the legacy
	// globalRegistry() and the input also registers its metrics directly.
	inputID = "input-with-pipeline-metrics-globalRegistry()"
	inputType = "test"
	reg = NewMetricsRegistry(inputID, inputType, globalRegistry(), log)
	monitoring.NewInt(reg, "events_pipeline_total").Set(20)
	defer globalRegistry().Remove(inputID)

	// now the input also register its metrics with the deprecated
	// NewInputRegistry.
	reg, cancel := NewInputRegistry(
		inputType, inputID, nil)
	defer cancel()
	monitoring.NewInt(reg, "foo_total").Set(20)

	// ===== An input registering metrics, but not using the new API =====
	// an input registering metrics on the global namespace. This simulates an
	// input which does not use the metrics registry from filebeat
	// input/v2.Context.
	inputOldAPI := "input-without-pipeline-metrics"
	reg, cancel = NewInputRegistry(
		inputType, inputOldAPI, nil)
	defer cancel()
	monitoring.NewInt(reg, "foo_total").Set(30)

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

	jsonBytes, err := MetricSnapshotJSON(parentLocalReg)
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
			EventsPipelineTotal: 10,
			FooTotal:            10,
			ID:                  "input-with-pipeline-metrics-new-inputAPI",
			Input:               inputType,
		},
		"input-with-pipeline-metrics-globalRegistry()": {
			EventsPipelineTotal: 20,
			FooTotal:            20,
			ID:                  "input-with-pipeline-metrics-globalRegistry()",
			Input:               inputType,
		},
		"input-without-pipeline-metrics": {
			FooTotal: 30,
			ID:       "input-without-pipeline-metrics",
			Input:    inputType,
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
		t.Logf("API response:\n%s\n", string(jsonBytes))
	}
}

func TestNewMetricsRegistry(t *testing.T) {
	parent := monitoring.NewRegistry()
	inputID := "input-inputID"
	inputType := "input-type"
	got := NewMetricsRegistry(
		inputID,
		inputType,
		parent,
		logptest.NewTestingLogger(t, "test"))

	require.NotNil(t, got, "new metrics registry should not be nil")
	assert.Equal(t, parent.GetRegistry(inputID), got)

	vals := monitoring.CollectFlatSnapshot(got, monitoring.Full, false)
	assert.Equal(t, inputID, vals.Strings["id"])
	assert.Equal(t, inputType, vals.Strings["input"])
}

func TestNewMetricsRegistry_duplicatedInputID(t *testing.T) {
	parent := monitoring.NewRegistry()
	inputID := "input-inputID"
	inputType := "input-type"
	metricName := "foo_total"
	goMetricsRegistryName := "bar_registry"

	// 1st call, create the registry
	got := NewMetricsRegistry(
		inputID,
		inputType,
		parent,
		logptest.NewTestingLogger(t, "test"))

	require.NotNil(t, got, "new metrics registry should not be nil")
	assert.Equal(t, parent.GetRegistry(inputID), got)
	// register a metric to the registry
	monitoring.NewInt(got, metricName)
	adapter.NewGoMetrics(got, goMetricsRegistryName, adapter.Accept)

	// 2nd call, return an unregistered registry
	got = NewMetricsRegistry(
		inputID,
		inputType,
		parent,
		logptest.NewTestingLogger(t, "test"))
	require.NotNil(t, got, "new metrics registry should not be nil")
	assert.NotEqual(t, parent.GetRegistry(inputID), got,
		"should get an unregistered registry, but found the registry on parent")
	assert.NotPanics(t, func() {
		// register the same metric again
		monitoring.NewInt(got, metricName)
		adapter.NewGoMetrics(got, goMetricsRegistryName, adapter.Accept)
	}, "the registry should be a new and empty registry")
}

func TestCancelMetricsRegistry(t *testing.T) {
	parent := monitoring.NewRegistry()
	inputID := "input-ID"
	inputType := "input-type"

	_ = parent.NewRegistry(inputID)
	got := parent.GetRegistry(inputID)
	require.NotNil(t, got, "metrics registry not found on parent")

	CancelMetricsRegistry(inputID, inputType, parent, logptest.NewTestingLogger(t, "test"))

	got = parent.GetRegistry(inputID)
	assert.Nil(t, got, "metrics registry was not removed from parent")
}

func TestMetricSnapshotJSON_regNil(t *testing.T) {
	err := globalRegistry().Clear()
	require.NoError(t, err, "could not clear global registry")

	got, err := MetricSnapshotJSON(nil)

	require.NoError(t, err, "MetricSnapshotJSON should not return an error")
	assert.Equal(t, "[]", string(got))
}
