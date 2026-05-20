// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/eventcache"
	"github.com/elastic/beats/v7/libbeat/publisher"
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

func TestConsumeLogsNativeEvents(t *testing.T) {
	beatInfo := beat.Info{Beat: "testbeat", Version: "1.0.0"}

	makeLogRecordWithToken := func(token int64) plog.LogRecord {
		lr := plog.NewLogRecord()
		lr.Attributes().PutInt(eventcache.TokenAttribute, token)
		return lr
	}

	putEvent := func(event beat.Event) int64 {
		pubEvent := publisher.Event{Content: event}
		return eventcache.Put(&eventcache.Entry{
			Event:    &pubEvent,
			BeatInfo: &beatInfo,
		})
	}

	t.Run("event is fetched from cache, encoded to pcommon body, token removed", func(t *testing.T) {
		ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
		event := beat.Event{
			Timestamp: ts,
			Fields:    mapstr.M{"message": "hello native"},
			Meta:      mapstr.M{},
		}
		token := putEvent(event)

		bp := &beatProcessor{logger: zap.NewNop()}

		logs := plog.NewLogs()
		lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		makeLogRecordWithToken(token).CopyTo(lr)

		out, err := bp.ConsumeLogs(context.Background(), logs)
		require.NoError(t, err)

		result := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

		// Body must be a map (full pcommon encoding).
		body := result.Body().Map().AsRaw()
		assert.Equal(t, "hello native", body["message"])
		assert.Contains(t, body, "@timestamp")

		// Token attribute must be removed.
		_, hasToken := result.Attributes().Get(eventcache.TokenAttribute)
		assert.False(t, hasToken, "cache token attribute must be removed after processing")

		// Cache entry must be gone.
		_, found := eventcache.Take(token)
		assert.False(t, found, "cache entry must be consumed by the processor")
	})

	t.Run("beat processors run on the cached event", func(t *testing.T) {
		event := beat.Event{
			Fields: mapstr.M{"message": "enrich me"},
			Meta:   mapstr.M{},
		}
		token := putEvent(event)

		bp := &beatProcessor{
			logger: zap.NewNop(),
			processors: []beat.Processor{
				mockProcessor{runFunc: func(e *beat.Event) (*beat.Event, error) {
					e.Fields["enriched"] = true
					return e, nil
				}},
			},
		}

		logs := plog.NewLogs()
		lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		makeLogRecordWithToken(token).CopyTo(lr)

		out, err := bp.ConsumeLogs(context.Background(), logs)
		require.NoError(t, err)

		body := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Map().AsRaw()
		assert.Equal(t, true, body["enriched"], "processor must have run on the cached event")
		assert.Equal(t, "enrich me", body["message"])
	})

	t.Run("metadata is included when IncludeMetadata is set", func(t *testing.T) {
		event := beat.Event{
			Fields: mapstr.M{"message": "with meta"},
			Meta:   mapstr.M{"raw_index": "logs-test"},
		}
		pubEvent := publisher.Event{Content: event}
		bi := beat.Info{Beat: "testbeat", Version: "1.0.0", IncludeMetadata: true}
		token := eventcache.Put(&eventcache.Entry{
			Event:    &pubEvent,
			BeatInfo: &bi,
		})

		bp := &beatProcessor{logger: zap.NewNop()}

		logs := plog.NewLogs()
		lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		makeLogRecordWithToken(token).CopyTo(lr)

		out, err := bp.ConsumeLogs(context.Background(), logs)
		require.NoError(t, err)

		body := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Map().AsRaw()
		meta, ok := body["@metadata"].(map[string]any)
		require.True(t, ok, "@metadata must be present in body")
		assert.Equal(t, "testbeat", meta["beat"])
		assert.Equal(t, "logs-test", meta["raw_index"])
	})

	t.Run("cache miss skips encoding and leaves body empty", func(t *testing.T) {
		bp := &beatProcessor{logger: zap.NewNop()}

		logs := plog.NewLogs()
		lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		makeLogRecordWithToken(99999999).CopyTo(lr) // token that was never Put

		out, err := bp.ConsumeLogs(context.Background(), logs)
		require.NoError(t, err)

		result := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		// Body stays empty since there is nothing to encode.
		assert.Equal(t, plog.NewLogRecord().Body().Type(), result.Body().Type())
	})

	t.Run("native and standard log records coexist in the same batch", func(t *testing.T) {
		event := beat.Event{
			Fields: mapstr.M{"source": "native"},
			Meta:   mapstr.M{},
		}
		token := putEvent(event)

		bp := &beatProcessor{
			logger: zap.NewNop(),
			processors: []beat.Processor{
				mockProcessor{runFunc: func(e *beat.Event) (*beat.Event, error) {
					e.Fields["processed"] = true
					return e, nil
				}},
			},
		}

		logs := plog.NewLogs()
		sls := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()

		// First record: native (token).
		nativeLR := sls.LogRecords().AppendEmpty()
		makeLogRecordWithToken(token).CopyTo(nativeLR)

		// Second record: standard pcommon map body.
		stdLR := sls.LogRecords().AppendEmpty()
		stdLR.Body().SetEmptyMap().PutStr("source", "standard")

		out, err := bp.ConsumeLogs(context.Background(), logs)
		require.NoError(t, err)

		records := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
		require.Equal(t, 2, records.Len())

		nativeBody := records.At(0).Body().Map().AsRaw()
		assert.Equal(t, "native", nativeBody["source"])
		assert.Equal(t, true, nativeBody["processed"])

		stdBody := records.At(1).Body().Map().AsRaw()
		assert.Equal(t, "standard", stdBody["source"])
		assert.Equal(t, true, stdBody["processed"])
	})
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
