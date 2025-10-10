// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
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
		processor, err := createProcessor(nil)
		require.NoError(t, err)
		assert.Nil(t, processor)
	})

	t.Run("empty config returns nil processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{})
		require.NoError(t, err)
		assert.Nil(t, processor)
	})

	t.Run("multiple processor names in config returns error", func(t *testing.T) {
		_, err := createProcessor(map[string]any{
			"add_host_metadata": map[string]any{},
			"another_key":       map[string]any{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected single processor name")
	})

	t.Run("unknown processor returns error", func(t *testing.T) {
		_, err := createProcessor(map[string]any{
			"unknown_processor": map[string]any{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid processor name 'unknown_processor'")
	})

	t.Run("valid add_host_metadata processor config returns processor", func(t *testing.T) {
		processor, err := createProcessor(map[string]any{
			"add_host_metadata": map[string]any{},
		})
		require.NoError(t, err)
		require.NotNil(t, processor)
		assert.Equal(t, "add_host_metadata", processor.String()[:len("add_host_metadata")])
	})
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
