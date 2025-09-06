// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vertexai_logs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// VertexAILogRow represents a single row from BigQuery vertex AI logs table
type VertexAILogRow struct {
	Endpoint        string    `bigquery:"endpoint"`
	DeployedModelID string    `bigquery:"deployed_model_id"`
	LoggingTime     time.Time `bigquery:"logging_time"`
	RequestID       float64   `bigquery:"request_id"`
	RequestPayload  []string  `bigquery:"request_payload"`
	ResponsePayload []string  `bigquery:"response_payload"`
	Model           string    `bigquery:"model"`
	ModelVersion    string    `bigquery:"model_version"`
	APIMethod       string    `bigquery:"api_method"`
	FullRequest     string    `bigquery:"full_request"`
	FullResponse    string    `bigquery:"full_response"`
	Metadata        string    `bigquery:"metadata"`
	OtelLog         string    `bigquery:"otel_log"`
}

// CreateEvent creates a single mb.Event from a VertexAILogRow
func CreateEvent(row VertexAILogRow, projectID string, logger *logp.Logger) (mb.Event, error) {
	event := mb.Event{
		Timestamp: row.LoggingTime,
	}

	// Build the main metricset fields
	fields := mapstr.M{
		"endpoint":          row.Endpoint,
		"deployed_model_id": row.DeployedModelID,
		"logging_time":      row.LoggingTime,
		"request_id":        row.RequestID,
		"request_payload":   row.RequestPayload,
		"response_payload":  row.ResponsePayload,
		"model":             row.Model,
		"model_version":     row.ModelVersion,
		"api_method":        row.APIMethod,
	}

	// Process JSON fields with error handling
	if err := processJSONField(row.FullRequest, "full_request", fields, logger); err != nil {
		logger.Warnf("failed to process full_request: %v", err)
	}

	if err := processJSONField(row.FullResponse, "full_response", fields, logger); err != nil {
		logger.Warnf("failed to process full_response: %v", err)
	}

	if err := processJSONField(row.Metadata, "metadata", fields, logger); err != nil {
		logger.Warnf("failed to process metadata: %v", err)
	}

	if err := processJSONField(row.OtelLog, "otel_log", fields, logger); err != nil {
		logger.Warnf("failed to process otel_log: %v", err)
	}

	event.MetricSetFields = fields

	// Set cloud provider information
	event.RootFields = mapstr.M{
		"cloud.provider":   "gcp",
		"cloud.project.id": projectID,
	}

	// Generate unique event ID
	event.ID = generateEventID(row)

	return event, nil
}

// processJSONField processes a JSON string field and adds it to the fields map
func processJSONField(jsonStr, fieldName string, fields mapstr.M, logger *logp.Logger) error {
	if jsonStr == "" || jsonStr == "{}" {
		return nil
	}

	var parsedJSON interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsedJSON); err != nil {
		// If JSON parsing fails, store the raw string in a structured way
		fields[fieldName] = mapstr.M{"raw": jsonStr}
		return fmt.Errorf("failed to parse %s JSON: %w", fieldName, err)
	}

	fields[fieldName] = parsedJSON
	return nil
}

// generateEventID creates a unique event ID based on row data
func generateEventID(row VertexAILogRow) string {
	eventData := fmt.Sprintf("%d_%.0f_%d",
		row.LoggingTime.Unix(),
		row.RequestID,
		len(row.RequestPayload))

	h := sha256.New()
	h.Write([]byte(eventData))
	return hex.EncodeToString(h.Sum(nil))[:20]
}

// EventsMapping processes multiple VertexAILogRow items and creates events
func EventsMapping(rows []VertexAILogRow, projectID string, logger *logp.Logger) []mb.Event {
	var events []mb.Event

	for _, row := range rows {
		event, err := CreateEvent(row, projectID, logger)
		if err != nil {
			logger.Warnf("failed to create event from row: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events
}
