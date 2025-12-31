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

package v2

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestNewPipelineClientListener_existingReg(t *testing.T) {
	reg := monitoring.NewRegistry()

	listener := NewPipelineClientListener(reg)
	require.NotNil(t, listener, "Listener should not be nil")

	assert.NotNilf(t, listener.eventsTotal,
		"%q metric should be created", metricEventsPipelineTotal)
	assert.NotNilf(t, listener.eventsFiltered,
		"%q metric should be created", metricEventsPipelineFiltered)
	assert.NotNilf(t, listener.eventsPublished,
		"%q metric should be created", metricEventsPipelinePublished)
}

func TestNewPipelineClientListener_nilReg(t *testing.T) {
	listener := NewPipelineClientListener(nil)
	require.NotNil(t, listener, "Listener should not be nil")

	assert.NotNilf(t, listener.eventsTotal,
		"%q metric should be created", metricEventsPipelineTotal)
	assert.NotNilf(t, listener.eventsFiltered,
		"%q metric should be created", metricEventsPipelineFiltered)
	assert.NotNilf(t, listener.eventsPublished,
		"%q metric should be created", metricEventsPipelinePublished)
}

func TestPrepareInputMetrics(t *testing.T) {
	log := logptest.NewTestingLogger(t, "TestPrepareInputMetrics")
	inputID := "test_input_id"
	inputType := "test_input_type"

	tcs := []struct {
		name     string
		cfg      beat.ClientConfig
		assertFn func(t *testing.T, cfg beat.ClientConfig)
	}{
		{
			name: "nil client listener",
			cfg:  beat.ClientConfig{},
			assertFn: func(t *testing.T, cfg beat.ClientConfig) {
				assert.NotNil(t, cfg.ClientListener, "ClientListener should not be nil")
				assert.IsTypef(t, &PipelineClientListener{}, cfg.ClientListener,
					"ClientListener should be of type %T", &PipelineClientListener{})
			},
		},
		{
			name: "existing client listener",
			cfg: beat.ClientConfig{
				ClientListener: &beat.CombinedClientListener{}},
			assertFn: func(t *testing.T, cfg beat.ClientConfig) {
				assert.NotNil(t, cfg.ClientListener, "ClientListener should not be nil")
				combListener, ok := cfg.ClientListener.(*beat.CombinedClientListener)
				assert.True(t, ok, "ClientListener should be of type %T", &beat.CombinedClientListener{})
				_, aOK := combListener.A.(*PipelineClientListener)
				_, bOK := combListener.B.(*PipelineClientListener)
				assert.True(t, aOK || bOK, "Either A or B should be of type %T", &PipelineClientListener{})
			},
		},
	}

	for _, tc := range tcs {
		parent := monitoring.NewRegistry()

		pc := pipetool.WithClientConfigEdit(pipeline.NewNilPipeline(),
			func(orig beat.ClientConfig) (beat.ClientConfig, error) {
				tc.assertFn(t, orig)
				return orig, nil
			})

		reg, wrappedconnector, cancelMetrics :=
			PrepareInputMetrics(inputID, inputType, parent, pc, log)
		t.Cleanup(cancelMetrics)

		require.NotNil(t, reg, "input metrics registry should not be nil")
		require.NotNil(t, wrappedconnector, "wrapped connector should not be nil")

		c, err := wrappedconnector.ConnectWith(tc.cfg)
		assert.NoError(t, err, "ConnectWith should not return an error")
		assert.NotNil(t, c, "client should not be nil")
	}
}

func TestPrepareInputMetrics_reusedReg_deprecatedDeprecatedMetricsRegistry(t *testing.T) {
	log := logptest.NewTestingLogger(t, "TestPrepareInputMetrics")
	inputID := "test_input_id"
	inputType := "test_input_type"

	parent := monitoring.NewRegistry()

	connector := pipeline.NewNilPipeline()
	reg, wrappedconnector, cancelMetrics :=
		PrepareInputMetrics(inputID, inputType, parent, connector, log)
	defer cancelMetrics()

	require.NotNil(t, reg, "input metrics registry should not be nil")
	require.NotNil(t, wrappedconnector, "wrapped connector should not be nil")

	got, cancel := inputmon.NewDeprecatedMetricsRegistry(inputType, inputID, parent)
	defer cancel()
	assert.Equal(t, reg, got, "metrics registry should be the same")
}

// TestPrepareInputMetrics_safeConcurrentPipelineClientCreation verifies that
// the PrepareInputMetrics returns a thread-safe PipelineClientListener.
//
// This test reproduces a scenario similar to what happens in inputs like AWS
// S3/SQS that create multiple worker goroutines, each with its own client. See
// https://github.com/elastic/beats/pull/44303 for more details.
//
// The test uses t.Run to execute multiple sub-tests because simply running
// iterations  in a for loop is not sufficient to reliably trigger the race
// condition.
func TestPrepareInputMetrics_safeConcurrentPipelineClientCreation(t *testing.T) {
	for i := range 100 {
		t.Run(fmt.Sprintf("run-%d", i+1), func(t *testing.T) {
			log := logptest.NewTestingLogger(t, "TestPrepareInputMetrics")
			inputID := "test_input_id"
			inputType := "test_input_type"

			parent := monitoring.NewRegistry()
			connector := pipeline.NewNilPipeline()

			reg, wrappedconnector, cancelMetrics :=
				PrepareInputMetrics(inputID, inputType, parent, connector, log)
			defer cancelMetrics()

			require.NotNil(t, reg, "input metrics registry should not be nil")
			require.NotNil(t, wrappedconnector, "wrapped connector should not be nil")

			// make sure ConnectWith can be called multiple times concurrently.
			// It simulates an input, like the AWS sqs input, that creates multiple
			// workers, each one with its own client.
			wg := sync.WaitGroup{}
			for range 5 {
				wg.Add(1)
				go func() {
					defer wg.Done()

					c, err := wrappedconnector.ConnectWith(beat.ClientConfig{})
					assert.NoError(t, err, "ConnectWith should not return an error")
					assert.NotNil(t, c, "client should not be nil")
				}()
			}
			wg.Wait()
		})
	}
}

func TestContextMetricsRegistryOverride(t *testing.T) {
	tcs := []struct {
		name       string
		field      string
		overrideFn func(reg *monitoring.Registry, val string)
		value      string
	}{
		{
			name:       "MetricsRegistryOverrideID",
			field:      inputmon.MetricKeyID,
			overrideFn: MetricsRegistryOverrideID,
			value:      "new-id",
		},
		{
			name:       "MetricsRegistryOverrideInput",
			field:      inputmon.MetricKeyInput,
			overrideFn: MetricsRegistryOverrideInput,
			value:      "new-input-name",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reg := monitoring.NewRegistry()
			monitoring.NewString(reg, tc.field).Set("old-" + tc.field)

			tc.overrideFn(reg, tc.value)
			require.NotNil(t, reg)
			assert.Equal(t, reg, reg)

			var got string
			reg.Visit(
				monitoring.Full,
				monitoring.NewKeyValueVisitor(func(key string, value any) {
					if key == tc.field {
						if s, ok := value.(string); ok {
							got = s
						}
					}
				}))
			assert.Equal(t, tc.value, got,
				"The %q variable in MetricsRegistry was not set correctly", tc.field)
		})
	}
}

// TestContexStatusReporterDoesNotPanic ensures that the UpdateStatus method
// is safe to use with a nil statusReporter
func TestContexStatusReporterDoesNotPanic(t *testing.T) {
	v2Ctx := Context{statusReporter: nil} // explicitly set it to nil
	v2Ctx.UpdateStatus(status.Configuring, "it does not panic")
}
