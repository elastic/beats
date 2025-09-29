// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcpbigquery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestConfigure(t *testing.T) {
	logger := logp.NewLogger("test")

	t.Run("valid config with single query", func(t *testing.T) {
		configMap := map[string]interface{}{
			"period":     "1m",
			"project_id": "test-project",
			"queries": []map[string]interface{}{
				{
					"query": "SELECT * FROM test_table",
					"cursor": map[string]interface{}{
						"field":         "id",
						"initial_value": "123",
					},
					"timestamp_field": "created_at",
				},
			},
		}

		cfg, err := conf.NewConfigFrom(configMap)
		require.NoError(t, err)

		sources, input, err := configure(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, input)
		require.Len(t, sources, 1)

		// Check source properties
		source := sources[0].(*bigQuerySource)
		assert.Equal(t, "test-project", source.ProjectID)
		assert.Equal(t, "SELECT * FROM test_table", source.Query)
		assert.Equal(t, "id", source.CursorField)
		assert.Equal(t, "123", source.CursorInitialValue)
		assert.Equal(t, "created_at", source.TimestampField)
		assert.True(t, source.ExpandJson) // should default to true

		// Check input properties
		bqInput := input.(*bigQueryInput)
		assert.Equal(t, "test-project", bqInput.config.ProjectID)
		assert.Equal(t, time.Minute, bqInput.config.Period)
	})

	t.Run("valid config with multiple queries", func(t *testing.T) {
		configMap := map[string]interface{}{
			"period":     "5m",
			"project_id": "test-project",
			"queries": []map[string]interface{}{
				{
					"query": "SELECT * FROM table1",
					"cursor": map[string]interface{}{
						"field": "id1",
					},
				},
				{
					"query":               "SELECT * FROM table2",
					"expand_json_strings": false,
				},
			},
		}

		cfg, err := conf.NewConfigFrom(configMap)
		require.NoError(t, err)

		sources, _, err := configure(cfg, logger)
		require.NoError(t, err)
		require.Len(t, sources, 2)

		// Check first source
		source1 := sources[0].(*bigQuerySource)
		assert.Equal(t, "SELECT * FROM table1", source1.Query)
		assert.Equal(t, "id1", source1.CursorField)
		assert.True(t, source1.ExpandJson) // defaults to true

		// Check second source
		source2 := sources[1].(*bigQuerySource)
		assert.Equal(t, "SELECT * FROM table2", source2.Query)
		assert.Equal(t, "", source2.CursorField)
		assert.False(t, source2.ExpandJson) // explicitly set to false
	})

	t.Run("expand_json_strings explicit true", func(t *testing.T) {
		configMap := map[string]interface{}{
			"period":     "1m",
			"project_id": "test-project",
			"queries": []map[string]interface{}{
				{
					"query":               "SELECT * FROM test_table",
					"expand_json_strings": true,
				},
			},
		}

		cfg, err := conf.NewConfigFrom(configMap)
		require.NoError(t, err)

		sources, _, err := configure(cfg, logger)
		require.NoError(t, err)

		source := sources[0].(*bigQuerySource)
		assert.True(t, source.ExpandJson)
	})

	t.Run("invalid config - missing required fields", func(t *testing.T) {
		configMap := map[string]interface{}{
			"period": "1m",
			// missing project_id and queries
		}

		cfg, err := conf.NewConfigFrom(configMap)
		require.NoError(t, err)

		_, _, err = configure(cfg, logger)
		assert.Error(t, err)
	})

	t.Run("invalid config - empty queries", func(t *testing.T) {
		configMap := map[string]interface{}{
			"period":     "1m",
			"project_id": "test-project",
			"queries":    []map[string]interface{}{},
		}

		cfg, err := conf.NewConfigFrom(configMap)
		require.NoError(t, err)

		_, _, err = configure(cfg, logger)
		assert.Error(t, err)
	})
}

type mockStatusReporter struct {
	mock.Mock
}

func (m *mockStatusReporter) UpdateStatus(st status.Status, msg string) {
	m.Called(st, msg)
}

func TestUpdateStatus(t *testing.T) {
	t.Run("nil reporter", func(t *testing.T) {
		ctx := v2.Context{
			StatusReporter: nil,
		}

		assert.NotPanics(t, func() {
			updateStatus(ctx, status.Running, "test message")
		})
	})

	t.Run("with reporter", func(t *testing.T) {
		mockReporter := &mockStatusReporter{}
		mockReporter.On("UpdateStatus", status.Degraded, "test message").Once()

		ctx := v2.Context{
			StatusReporter: mockReporter,
		}

		updateStatus(ctx, status.Degraded, "test message")
		mockReporter.AssertExpectations(t)
	})
}

func TestBigQuerySource(t *testing.T) {
	t.Run("Name - deterministic", func(t *testing.T) {
		source := &bigQuerySource{
			ProjectID:      "test-project",
			Query:          "SELECT * FROM table",
			CursorField:    "id",
			TimestampField: "created_at",
			ExpandJson:     true,
		}

		name1 := source.Name()
		name2 := source.Name()

		assert.Equal(t, name1, name2, "Name() should return the same value for identical inputs")
		assert.NotEmpty(t, name1, "Name() should not return empty string")
	})

	t.Run("Name - different inputs produce different names", func(t *testing.T) {
		baseSource := &bigQuerySource{
			ProjectID:   "test-project",
			Query:       "SELECT * FROM table",
			CursorField: "id",
		}

		// Different ProjectID
		source1 := &bigQuerySource{
			ProjectID:   "different-project",
			Query:       "SELECT * FROM table",
			CursorField: "id",
		}

		// Different Query
		source2 := &bigQuerySource{
			ProjectID:   "test-project",
			Query:       "SELECT * FROM other_table",
			CursorField: "id",
		}

		// Different CursorField
		source3 := &bigQuerySource{
			ProjectID:   "test-project",
			Query:       "SELECT * FROM table",
			CursorField: "timestamp",
		}

		baseName := baseSource.Name()
		name1 := source1.Name()
		name2 := source2.Name()
		name3 := source3.Name()

		assert.NotEqual(t, baseName, name1, "Different ProjectID should produce different name")
		assert.NotEqual(t, baseName, name2, "Different Query should produce different name")
		assert.NotEqual(t, baseName, name3, "Different CursorField should produce different name")

		// All names should be unique
		allNames := []string{baseName, name1, name2, name3}
		uniqueNames := make(map[string]bool)
		for _, name := range allNames {
			uniqueNames[name] = true
		}
		assert.Len(t, uniqueNames, len(allNames), "All names should be unique")
	})

	t.Run("Name - irrelevant fields do not affect name", func(t *testing.T) {
		source1 := &bigQuerySource{
			ProjectID:          "test-project",
			Query:              "SELECT * FROM table",
			CursorField:        "id",
			CursorInitialValue: "0",
			TimestampField:     "created_at",
			ExpandJson:         true,
		}

		source2 := &bigQuerySource{
			ProjectID:          "test-project",
			Query:              "SELECT * FROM table",
			CursorField:        "id",
			CursorInitialValue: "1000",       // Different
			TimestampField:     "updated_at", // Different
			ExpandJson:         false,        // Different
		}

		name1 := source1.Name()
		name2 := source2.Name()

		assert.Equal(t, name1, name2, "CursorInitialValue, TimestampField, and ExpandJson should not affect the name")
	})
}

func TestBigQueryInput(t *testing.T) {
	t.Run("Name", func(t *testing.T) {
		input := &bigQueryInput{
			config: config{},
			logger: logp.NewLogger("test"),
		}

		assert.Equal(t, "gcpbigquery", input.Name())
	})

	t.Run("Test", func(t *testing.T) {
		input := &bigQueryInput{
			config: config{},
			logger: logp.NewLogger("test"),
		}

		source := &bigQuerySource{
			ProjectID:   "test-project",
			Query:       "SELECT * FROM table",
			CursorField: "id",
		}

		err := input.Test(source, v2.TestContext{})
		assert.NoError(t, err, "Test should always return nil")
	})
}

// // mockPublisher implements cursor.Publisher for testing
// type mockPublisher struct {
// 	mu         sync.Mutex
// 	events     []beat.Event
// 	cursors    []interface{}
// 	publishErr error
// }

// func newMockPublisher() *mockPublisher {
// 	return &mockPublisher{
// 		events:  make([]beat.Event, 0),
// 		cursors: make([]interface{}, 0),
// 	}
// }

// func (m *mockPublisher) Publish(event beat.Event, cursor interface{}) error {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	if m.publishErr != nil {
// 		return m.publishErr
// 	}

// 	m.events = append(m.events, event)
// 	m.cursors = append(m.cursors, cursor)
// 	return nil
// }

// func (m *mockPublisher) Events() []beat.Event {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	events := make([]beat.Event, len(m.events))
// 	copy(events, m.events)
// 	return events
// }

// func (m *mockPublisher) Cursors() []interface{} {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	cursors := make([]interface{}, len(m.cursors))
// 	copy(cursors, m.cursors)
// 	return cursors
// }

// func (m *mockPublisher) SetPublishError(err error) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
// 	m.publishErr = err
// }

// // Helper functions for creating test contexts
// func createTestContext() (v2.Context, context.CancelFunc) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	return v2.Context{
// 		Logger:      logp.NewLogger("test"),
// 		ID:          "test-input",
// 		Cancelation: v2.GoContextFromCanceler(ctx),
// 	}, cancel
// }

// func createTestSource() cursor.Source {
// 	return &bigQuerySource{
// 		ProjectID:   "test-project",
// 		Query:       "SELECT * FROM test_table",
// 		CursorField: "id",
// 	}
// }
