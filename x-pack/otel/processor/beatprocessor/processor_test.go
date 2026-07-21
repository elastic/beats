// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/internal/otel/sharedcomponent"
	"github.com/elastic/beats/v7/libbeat/beat"
	_ "github.com/elastic/beats/v7/libbeat/cmd/instance" // needed for registering processors
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
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

	t.Run("valid detect_mime_type processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"detect_mime_type": map[string]any{
				"field":  "http.request.body.content",
				"target": "http.request.mime_type",
			},
		}, testLogger())
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "detect_mime_type", processor.String()[:len("detect_mime_type")])
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

	t.Run("invalid processor", func(t *testing.T) {
		_, err := createProcessor(map[string]any{
			"unsupported_processor": map[string]any{},
		}, testLogger())
		require.Error(t, err)
	})
}

func TestShutdownClosesProcessors(t *testing.T) {
	t.Run("closes every Closer exactly once and ignores non-Closers", func(t *testing.T) {
		closer1 := &closerProcessor{}
		closer2 := &closerProcessor{}
		plain := mockProcessor{runFunc: func(e *beat.Event) (*beat.Event, error) { return e, nil }}

		bp := &beatProcessor{
			logger:     zap.NewNop(),
			processors: []beat.Processor{closer1, plain, closer2},
		}

		require.NoError(t, bp.Shutdown(context.Background()))
		assert.Equal(t, 1, closer1.closeCalls, "first Closer should be closed exactly once on Shutdown")
		assert.Equal(t, 1, closer2.closeCalls, "second Closer should be closed exactly once on Shutdown")

		// A second Shutdown must not close the processors again: for shared
		// processors a double-close would over-decrement the refcount.
		require.NoError(t, bp.Shutdown(context.Background()))
		assert.Equal(t, 1, closer1.closeCalls, "Closer should not be closed again on a repeated Shutdown")
		assert.Equal(t, 1, closer2.closeCalls, "Closer should not be closed again on a repeated Shutdown")
	})

	t.Run("joins close errors", func(t *testing.T) {
		closer1 := &closerProcessor{err: errors.New("boom-1")}
		closer2 := &closerProcessor{err: errors.New("boom-2")}
		bp := &beatProcessor{
			logger:     zap.NewNop(),
			processors: []beat.Processor{closer1, closer2},
		}

		err := bp.Shutdown(context.Background())
		require.Error(t, err, "Shutdown should surface errors returned by processor Close")
		assert.ErrorContains(t, err, "boom-1")
		assert.ErrorContains(t, err, "boom-2")
	})
}

// TestLifecycleReloadClosesSharedProcessors reproduces a collector pipeline
// reload: it builds the component through the shared-component map (as the
// factory does), starts it, then shuts it down and asserts the constructed
// processors were closed and the component was evicted so the next reload
// rebuilds cleanly. Before the fix, Shutdown was a no-op, so the underlying
// processors leaked their resources on every reload.
func TestLifecycleReloadClosesSharedProcessors(t *testing.T) {
	closer := &closerProcessor{}
	sharedMap := sharedcomponent.NewMap[*Config, *beatProcessor]()
	cfg := &Config{}

	comp, err := sharedMap.LoadOrStore(cfg, func() (*beatProcessor, error) {
		return &beatProcessor{
			logger:     zap.NewNop(),
			processors: []beat.Processor{closer},
		}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, sharedMap.Len(), "component should be registered after LoadOrStore")

	require.NoError(t, comp.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, comp.Shutdown(context.Background()))

	assert.Equal(t, 1, closer.closeCalls, "reload teardown should close the constructed processor")
	assert.Equal(t, 0, sharedMap.Len(), "shared component should be evicted after Shutdown so the next reload rebuilds")
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

// closerProcessor is a beat.Processor that also implements processors.Closer so
// tests can observe that Shutdown closes the processors it holds.
type closerProcessor struct {
	closeCalls int
	err        error
}

func (c *closerProcessor) Run(event *beat.Event) (*beat.Event, error) { return event, nil }
func (c *closerProcessor) String() string                             { return "closerProcessor" }
func (c *closerProcessor) Close() error {
	c.closeCalls++
	return c.err
}
