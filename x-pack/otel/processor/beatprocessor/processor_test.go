// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestConsumeLogs(t *testing.T) {
	// Arrange
	beatProcessor := &beatProcessor{
		logger: zap.NewNop(),
		processors: []beat.Processor{
			mockProcessor{
				runFunc: func(event *beat.Event) (*beat.Event, error) {
					event.Fields["host"] = mapstr.M{"name": "test-host"}
					return event, nil
				},
			},
		},
	}

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	for i := range 2 {
		logRecord := scopeLogs.LogRecords().AppendEmpty()
		logRecord.Body().SetEmptyMap()
		logRecord.Body().Map().PutStr("message", fmt.Sprintf("test log message %v", i))
	}

	// Act
	processedLogs, err := beatProcessor.ConsumeLogs(context.Background(), logs)
	require.NoError(t, err)

	// Assert
	for _, resourceLogs := range processedLogs.ResourceLogs().All() {
		for _, scopeLogs := range resourceLogs.ScopeLogs().All() {
			for i, logRecord := range scopeLogs.LogRecords().All() {
				// Verify that the original contents of the log is unchanged.
				messageAttribute, found := logRecord.Body().Map().Get("message")
				assert.True(t, found, "'message' not found in log record")
				assert.Equal(t, fmt.Sprintf("test log message %v", i), messageAttribute.Str())

				// Verify that the host attribute is added.
				hostAttribute, found := logRecord.Body().Map().Get("host")
				assert.True(t, found, "'host' not found in log record")
				nameAttribute, found := hostAttribute.Map().Get("name")
				assert.True(t, found, "'name' not found in 'host' attribute")
				assert.Equal(t, "test-host", nameAttribute.Str())
			}
		}
	}
}

func TestCreateProcessor(t *testing.T) {
	t.Run("nil config returns nil processor", func(t *testing.T) {
		processor, err := createProcessor(nil, testLogger())
		require.NoError(t, err)
		assert.Nil(t, processor)
	})

	t.Run("empty config returns nil processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{}, testLogger())
		require.NoError(t, err)
		assert.Nil(t, processor)
	})

	t.Run("multiple processor names in config returns error", func(t *testing.T) {
		_, err := createProcessor(map[string]any{
			"add_host_metadata": map[string]any{},
			"another_key":       map[string]any{},
		}, testLogger())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected single processor name")
	})

	t.Run("unknown processor returns error", func(t *testing.T) {
		_, err := createProcessor(map[string]any{
			"unknown_processor": map[string]any{},
		}, testLogger())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid processor name 'unknown_processor'")
	})

	t.Run("valid add_cloud_metadata processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_cloud_metadata": map[string]any{},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "add_cloud_metadata", processor.String()[:len("add_cloud_metadata")])
	})

	t.Run("valid add_docker_metadata processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_docker_metadata": map[string]any{},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "add_docker_metadata", processor.String()[:len("add_docker_metadata")])
	})

	t.Run("valid add_fields processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_fields": map[string]any{
				"fields": map[string]any{
					"env": "staging",
				},
			},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "add_fields", processor.String()[:len("add_fields")])
	})

	t.Run("valid add_host_metadata processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_host_metadata": map[string]any{},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "add_host_metadata", processor.String()[:len("add_host_metadata")])
	})

	t.Run("valid add_kubernetes_metadata processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_kubernetes_metadata": map[string]any{},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "add_kubernetes_metadata", processor.String()[:len("add_kubernetes_metadata")])
	})

	t.Run("when condition is honored and processor is skipped when condition is false", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_fields": map[string]any{
				"target": "",
				"fields": map[string]any{
					"enriched": "yes",
				},
				"when": map[string]any{
					"contains": map[string]any{
						"tags": "forwarded",
					},
				},
			},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Contains(t, processor.String(), "condition=", "expected processor to be wrapped with a condition")

		event := &beat.Event{Fields: mapstr.M{"message": "hello"}}
		out, err := processor.Run(event)
		require.NoError(t, err)
		_, lookupErr := out.Fields.GetValue("enriched")
		assert.Error(t, lookupErr, "expected 'enriched' field to be absent when condition is not met")
	})

	t.Run("when condition is honored and processor runs when condition is true", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_fields": map[string]any{
				"target": "",
				"fields": map[string]any{
					"enriched": "yes",
				},
				"when": map[string]any{
					"contains": map[string]any{
						"tags": "forwarded",
					},
				},
			},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)

		event := &beat.Event{Fields: mapstr.M{"message": "hello", "tags": []string{"forwarded"}}}
		out, err := processor.Run(event)
		require.NoError(t, err)
		val, err := out.Fields.GetValue("enriched")
		require.NoError(t, err, "expected 'enriched' field to be added when condition is met")
		assert.Equal(t, "yes", val)
	})

	t.Run("when.not.contains skips processor when matching tag is present", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_fields": map[string]any{
				"target": "",
				"fields": map[string]any{
					"enriched": "yes",
				},
				"when.not.contains.tags": "forwarded",
			},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)

		event := &beat.Event{Fields: mapstr.M{"message": "hello", "tags": []string{"forwarded"}}}
		out, err := processor.Run(event)
		require.NoError(t, err)
		_, lookupErr := out.Fields.GetValue("enriched")
		assert.Error(t, lookupErr, "expected 'enriched' field to be absent when 'forwarded' tag is present")
	})

	t.Run("invalid when condition returns error", func(t *testing.T) {
		_, err := createProcessor(map[string]any{
			"add_host_metadata": map[string]any{
				"when": map[string]any{
					"not_a_real_condition": map[string]any{},
				},
			},
		}, testLogger())
		require.Error(t, err)
	})
}

// TestConsumeLogsPdataPath verifies that each supported processor either takes
// the pdata fast path (implements PdataProcessor) or falls back to the legacy
// path, and that ConsumeLogs produces the expected result in both cases.
func TestConsumeLogsPdataPath(t *testing.T) {
	type processorCase struct {
		name        string
		config      map[string]any
		wantPdata   bool
		checkResult func(t *testing.T, body map[string]any)
	}

	cases := []processorCase{
		{
			name: "add_host_metadata",
			config: map[string]any{
				"add_host_metadata": map[string]any{},
			},
			wantPdata: true,
			checkResult: func(t *testing.T, body map[string]any) {
				host, ok := body["host"]
				require.True(t, ok, "expected 'host' key in body after add_host_metadata")
				hostMap, ok := host.(map[string]any)
				require.True(t, ok, "expected 'host' to be a map")
				assert.NotEmpty(t, hostMap["name"], "expected host.name to be set")
			},
		},
		{
			name: "add_cloud_metadata",
			config: map[string]any{
				"add_cloud_metadata": map[string]any{},
			},
			wantPdata: true,
			checkResult: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "test message", body["message"], "original message must be preserved")
			},
		},
		{
			name: "add_docker_metadata",
			config: map[string]any{
				"add_docker_metadata": map[string]any{},
			},
			wantPdata: true,
			checkResult: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "test message", body["message"], "original message must be preserved")
			},
		},
		{
			name: "add_kubernetes_metadata",
			config: map[string]any{
				"add_kubernetes_metadata": map[string]any{},
			},
			wantPdata: true,
			checkResult: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "test message", body["message"], "original message must be preserved")
			},
		},
		{
			name: "add_fields",
			config: map[string]any{
				"add_fields": map[string]any{
					"target": "",
					"fields": map[string]any{"env": "test"},
				},
			},
			wantPdata: true,
			checkResult: func(t *testing.T, body map[string]any) {
				assert.Equal(t, "test", body["env"], "add_fields should add the 'env' field")
			},
		},
		{
			name: "add_host_metadata with when condition (pdata inner)",
			config: map[string]any{
				"add_host_metadata": map[string]any{
					"when": map[string]any{
						"equals": map[string]any{"tagged": "yes"},
					},
				},
			},
			wantPdata: true,
			checkResult: func(t *testing.T, body map[string]any) {
				// condition not met — host should not be enriched
				_, hasHost := body["host"]
				assert.False(t, hasHost, "host should not be added when condition is not met")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proc, err := createProcessor(tc.config, testLogger())
			require.NoError(t, err)
			require.NotNil(t, proc)

			_, isPdata := proc.(PdataProcessor)
			assert.Equal(t, tc.wantPdata, isPdata, "PdataProcessor implementation mismatch")

			logs := plog.NewLogs()
			lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			lr.Body().SetEmptyMap()
			lr.Body().Map().PutStr("message", "test message")

			bp := &beatProcessor{logger: zap.NewNop(), processors: []beat.Processor{proc}}
			out, err := bp.ConsumeLogs(context.Background(), logs)
			require.NoError(t, err)

			body := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Map().AsRaw()
			tc.checkResult(t, body)
		})
	}
}

// TestWhenProcessorRunPdataLegacyFallback verifies the round-trip path in
// WhenProcessor.RunPdata when the inner processor does not implement
// PdataProcessor. Fields added by the inner processor must appear and fields it
// removes must be absent after the call.
func TestWhenProcessorRunPdataLegacyFallback(t *testing.T) {
	addField := mockProcessor{
		runFunc: func(event *beat.Event) (*beat.Event, error) {
			event.Fields["injected"] = "yes"
			delete(event.Fields, "remove_me")
			return event, nil
		},
	}

	// Wrap with a when condition that passes when present=="yes".
	cfg, err := config.NewConfigFrom(map[string]any{
		"when": map[string]any{"equals": map[string]any{"present": "yes"}},
	})
	require.NoError(t, err)
	proc, err := processors.NewConditional(func(_ *config.C, _ *logp.Logger) (beat.Processor, error) {
		return addField, nil
	})(cfg, testLogger())
	require.NoError(t, err)

	pp, ok := proc.(PdataProcessor)
	require.True(t, ok, "WhenProcessor must implement PdataProcessor")

	body := plog.NewLogs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().
		LogRecords().AppendEmpty().Body().SetEmptyMap()
	body.PutStr("present", "yes")
	body.PutStr("remove_me", "gone")

	require.NoError(t, pp.RunPdata(body))

	raw := body.AsRaw()
	assert.Equal(t, "yes", raw["injected"], "injected field must be present")
	_, hasRemoved := raw["remove_me"]
	assert.False(t, hasRemoved, "field deleted by inner processor must be absent")
}

// TestAddFieldsRunPdataNoOverwrite verifies that RunPdata with overwrite=false
// does not clobber existing values.
func TestAddFieldsRunPdataNoOverwrite(t *testing.T) {
	proc := addfields.NewAddFields(mapstr.M{"env": "prod", "new": "val"}, false, false)

	pp, ok := proc.(PdataProcessor)
	require.True(t, ok, "addFields must implement PdataProcessor")

	body := plog.NewLogs().ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().
		LogRecords().AppendEmpty().Body().SetEmptyMap()
	body.PutStr("env", "staging") // pre-existing, must not be overwritten
	body.PutStr("message", "hello")

	require.NoError(t, pp.RunPdata(body))

	raw := body.AsRaw()
	assert.Equal(t, "staging", raw["env"], "existing value must not be overwritten")
	assert.Equal(t, "val", raw["new"], "new field must be added")
	assert.Equal(t, "hello", raw["message"], "unrelated field must be preserved")
}

func testLogger() *logp.Logger {
	return logp.NewNopLogger()
}

type mockProcessor struct {
	runFunc func(event *beat.Event) (*beat.Event, error)
}

func (m mockProcessor) Run(event *beat.Event) (*beat.Event, error) {
	return m.runFunc(event)
}

func (m mockProcessor) String() string {
	return "mockProcessor"
}
