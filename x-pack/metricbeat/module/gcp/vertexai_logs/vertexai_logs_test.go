// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vertexai_logs

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func TestGenerateQuery(t *testing.T) {
	m := &MetricSet{
		config: config{
			TableID: "project-123.dataset.table_name",
		},
	}

	query := m.generateQuery()

	// verify that table name quoting is in effect
	assert.Contains(t, query, "`project-123.dataset.table_name`")
	// verify WHERE clause is present
	assert.Contains(t, query, "WHERE")
	assert.Contains(t, query, "logging_time IS NOT NULL")
	// verify ORDER BY is present
	assert.Contains(t, query, "ORDER BY")
	assert.Contains(t, query, "logging_time DESC")
	// verify LIMIT is present
	assert.Contains(t, query, "LIMIT 10000")
	// verify CAST for request_id
	assert.Contains(t, query, "CAST(IFNULL(request_id, 0) AS FLOAT64)")
}

func TestCreateEvent(t *testing.T) {
	assert := assert.New(t)

	testTime := time.Date(2023, 12, 1, 10, 30, 45, 0, time.UTC)
	row := VertexAILogRow{
		Endpoint:        "https://us-central1-aiplatform.googleapis.com",
		DeployedModelID: "model-123456",
		LoggingTime:     testTime,
		RequestID:       12345.67,
		RequestPayload:  []string{"prompt1", "prompt2"},
		ResponsePayload: []string{"response1", "response2"},
		Model:           "gemini-2.5-pro",
		ModelVersion:    "1.0",
		APIMethod:       "generateContent",
		FullRequest:     `{"inputs": ["test"]}`,
		FullResponse:    `{"outputs": ["result"]}`,
		Metadata:        `{"user_id": "user123"}`,
		OtelLog:         `{"trace_id": "abc123"}`,
	}

	projectID := "test-project"
	logger := logp.NewLogger("test")

	event, err := CreateEvent(row, projectID, logger)

	assert.NoError(err)
	assert.Equal(testTime, event.Timestamp)

	// Check MetricSetFields
	expectedFields := mapstr.M{
		"endpoint":          "https://us-central1-aiplatform.googleapis.com",
		"deployed_model_id": "model-123456",
		"logging_time":      testTime,
		"request_id":        12345.67,
		"request_payload":   []string{"prompt1", "prompt2"},
		"response_payload":  []string{"response1", "response2"},
		"model":             "gemini-2.5-pro",
		"model_version":     "1.0",
		"api_method":        "generateContent",
		"full_request":      map[string]interface{}{"inputs": []interface{}{"test"}},
		"full_response":     map[string]interface{}{"outputs": []interface{}{"result"}},
		"metadata":          map[string]interface{}{"user_id": "user123"},
		"otel_log":          map[string]interface{}{"trace_id": "abc123"},
	}

	assert.Equal(expectedFields, event.MetricSetFields)

	// Check RootFields
	expectedRootFields := mapstr.M{
		"cloud.provider":   "gcp",
		"cloud.project.id": projectID,
	}
	assert.Equal(expectedRootFields, event.RootFields)

	// Check that ID is generated
	assert.NotEmpty(event.ID)
	assert.Len(event.ID, 20) // generateEventID returns 20 character hash
}

func TestCreateEventWithInvalidJSON(t *testing.T) {
	assert := assert.New(t)

	testTime := time.Date(2023, 12, 1, 10, 30, 45, 0, time.UTC)
	row := VertexAILogRow{
		Endpoint:        "https://us-central1-aiplatform.googleapis.com",
		DeployedModelID: "model-123456",
		LoggingTime:     testTime,
		RequestID:       12345.67,
		RequestPayload:  []string{"prompt1"},
		ResponsePayload: []string{"response1"},
		Model:           "gemini-2.5-pro",
		ModelVersion:    "1.0",
		APIMethod:       "generateContent",
		FullRequest:     `{"invalid": json}`, // Invalid JSON
		FullResponse:    `{}`,
		Metadata:        `{}`,
		OtelLog:         `{}`,
	}

	projectID := "test-project"
	logger := logp.NewLogger("test")

	event, err := CreateEvent(row, projectID, logger)

	assert.NoError(err) // Should not error, but log warning
	
	// Invalid JSON should be stored as raw string
	fullRequestField, err := event.MetricSetFields.GetValue("full_request.raw")
	assert.NoError(err)
	assert.Equal(`{"invalid": json}`, fullRequestField)
}

func TestGenerateEventID(t *testing.T) {
	testTime := time.Date(2023, 12, 1, 10, 30, 45, 0, time.UTC)
	row := VertexAILogRow{
		LoggingTime:    testTime,
		RequestID:      12345.67,
		RequestPayload: []string{"prompt1", "prompt2"},
	}

	id1 := generateEventID(row)
	id2 := generateEventID(row)

	// Same input should produce same ID
	assert.Equal(t, id1, id2)
	assert.Len(t, id1, 20)

	// Different input should produce different ID
	row.RequestID = 98765.43
	id3 := generateEventID(row)
	assert.NotEqual(t, id1, id3)
}

func TestEventsMapping(t *testing.T) {
	assert := assert.New(t)

	testTime := time.Date(2023, 12, 1, 10, 30, 45, 0, time.UTC)
	rows := []VertexAILogRow{
		{
			Endpoint:        "https://us-central1-aiplatform.googleapis.com",
			DeployedModelID: "model-123456",
			LoggingTime:     testTime,
			RequestID:       12345.67,
			RequestPayload:  []string{"prompt1"},
			ResponsePayload: []string{"response1"},
			Model:           "gemini-2.5-pro",
			ModelVersion:    "1.0",
			APIMethod:       "generateContent",
			FullRequest:     `{}`,
			FullResponse:    `{}`,
			Metadata:        `{}`,
			OtelLog:         `{}`,
		},
		{
			Endpoint:        "https://us-west1-aiplatform.googleapis.com",
			DeployedModelID: "model-789012",
			LoggingTime:     testTime.Add(time.Hour),
			RequestID:       67890.12,
			RequestPayload:  []string{"prompt2"},
			ResponsePayload: []string{"response2"},
			Model:           "gemini-1.5-pro",
			ModelVersion:    "2.0",
			APIMethod:       "predict",
			FullRequest:     `{}`,
			FullResponse:    `{}`,
			Metadata:        `{}`,
			OtelLog:         `{}`,
		},
	}

	projectID := "test-project"
	logger := logp.NewLogger("test")

	events := EventsMapping(rows, projectID, logger)

	assert.Len(events, 2)
	assert.Equal("model-123456", events[0].MetricSetFields["deployed_model_id"])
	assert.Equal("model-789012", events[1].MetricSetFields["deployed_model_id"])
}

