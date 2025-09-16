// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vertexai_logs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
	"google.golang.org/api/iterator"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	metricsetName = "vertexai_logs"
)

func init() {
	mb.Registry.MustAddMetricSet(gcp.ModuleName, metricsetName, New)
}

type MetricSet struct {
	mb.BaseMetricSet
	config          config
	logger          *logp.Logger
	lastLoggingTime *time.Time
}

type config struct {
	Period              time.Duration `config:"period" validate:"required"`
	ProjectID           string        `config:"project_id" validate:"required"`
	TableID             string        `config:"table_id" validate:"required"`
	CredentialsFilePath string        `config:"credentials_file_path"`
	CredentialsJSON     string        `config:"credentials_json"`
	TimeLookbackHours   int           `config:"time_lookback_hours"`
}

func (c config) Validate() error {
	if c.CredentialsFilePath == "" && c.CredentialsJSON == "" {
		return errors.New("no credentials_file_path or credentials_json specified")
	}

	if c.ProjectID == "" {
		return errors.New("project_id is required")
	}

	if c.TableID == "" {
		return errors.New("table_id is required")
	}

	parts := strings.Split(c.TableID, ".")
	if len(parts) != 3 {
		return fmt.Errorf("table_id must be in format 'project_id.dataset_id.table_name', got: %s", c.TableID)
	}

	if c.TimeLookbackHours < 0 {
		return errors.New("time_lookback_hours must be non-negative")
	}

	return nil
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	m := &MetricSet{
		BaseMetricSet: base,
		logger:        base.Logger().Named(metricsetName),
	}

	if err := base.Module().UnpackConfig(&m.config); err != nil {
		return nil, fmt.Errorf("unpack vertexai_logs config failed: %w", err)
	}

	// Set defaults
	if m.config.TimeLookbackHours == 0 {
		m.config.TimeLookbackHours = 1 // Default: 1 hour
	}

	m.logger.Debugf("metricset config: project_id=%s, dataset_id=%s, table_name=%s, time_lookback=%dh",
		m.config.ProjectID, getDatasetID(m.config.TableID), getTableName(m.config.TableID),
		m.config.TimeLookbackHours)
	return m, nil
}

func getDatasetID(tableID string) string {
	parts := strings.Split(tableID, ".")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}

func getTableName(tableID string) string {
	parts := strings.Split(tableID, ".")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	var opt []option.ClientOption
	if m.config.CredentialsFilePath != "" && m.config.CredentialsJSON != "" {
		return errors.New("both credentials_file_path and credentials_json specified, you must use only one of them")
	} else if m.config.CredentialsFilePath != "" {
		opt = []option.ClientOption{option.WithCredentialsFile(m.config.CredentialsFilePath)}
	} else if m.config.CredentialsJSON != "" {
		opt = []option.ClientOption{option.WithCredentialsJSON([]byte(m.config.CredentialsJSON))}
	} else {
		return errors.New("no credentials_file_path or credentials_json specified")
	}

	client, err := bigquery.NewClient(ctx, m.config.ProjectID, opt...)
	if err != nil {
		return fmt.Errorf("error creating bigquery client: %w", err)
	}
	defer client.Close()

	datasetID := getDatasetID(m.config.TableID)
	dataset := client.Dataset(datasetID)
	meta, err := dataset.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("error getting dataset metadata: %w", err)
	}

	events, err := m.queryVertexAILogs(ctx, client, meta.Location)
	if err != nil {
		return fmt.Errorf("queryVertexAILogs failed: %w", err)
	}

	m.logger.Debugf("Total %d events created for vertexai_logs", len(events))
	for _, event := range events {
		reporter.Event(event)
	}

	// Update watermark with latest logging_time from events
	m.updateLastLoggingTime(events)

	return nil
}

func (m *MetricSet) queryVertexAILogs(ctx context.Context, client *bigquery.Client, location string) ([]mb.Event, error) {
	query := m.generateQuery()
	m.logger.Debug("bigquery query = ", query)

	q := client.Query(query)
	q.Location = location

	job, err := q.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery Run failed: %w", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery Wait failed: %w", err)
	}

	if err := status.Err(); err != nil {
		return nil, fmt.Errorf("bigquery status error: %w", err)
	}

	it, err := job.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading from bigquery job failed: %w", err)
	}

	var rows []VertexAILogRow
	for {
		var row VertexAILogRow
		err := it.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bigquery RowIterator Next failed: %w", err)
		}
		rows = append(rows, row)
	}

	events := EventsMapping(rows, m.config.ProjectID, m.logger)
	return events, nil
}

func (m *MetricSet) generateQuery() string {
	escapedTableID := fmt.Sprintf("`%s`", m.config.TableID)

	var whereClause string
	if m.lastLoggingTime != nil {
		// Incremental query: get records after last processed time
		whereClause = fmt.Sprintf("logging_time >= TIMESTAMP('%s')",
			m.lastLoggingTime.Format("2006-01-02 15:04:05.000000"))
		m.logger.Debugf("Using incremental query from logging_time: %s", m.lastLoggingTime.Format(time.RFC3339))
	} else {
		// First run: use timestamp filter with lookback
		whereClause = fmt.Sprintf("logging_time >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL %d HOUR)",
			m.config.TimeLookbackHours)
		m.logger.Debugf("Using initial query with timestamp filter: %d hours", m.config.TimeLookbackHours)
	}

	query := fmt.Sprintf(`
SELECT
	IFNULL(endpoint, '') AS endpoint,
	IFNULL(deployed_model_id, '') AS deployed_model_id,
	logging_time,
	IFNULL(CAST(request_id AS STRING), '') AS request_id,
	IFNULL(request_payload, []) AS request_payload,
	IFNULL(response_payload, []) AS response_payload,
	IFNULL(model, '') AS model,
	IFNULL(model_version, '') AS model_version,
	IFNULL(api_method, '') AS api_method,
	IFNULL(TO_JSON_STRING(full_request), '{}') AS full_request,
	IFNULL(TO_JSON_STRING(full_response), '{}') AS full_response,
	IFNULL(TO_JSON_STRING(metadata), '{}') AS metadata
FROM
	%s
WHERE
	%s
	AND logging_time IS NOT NULL
ORDER BY
	logging_time ASC`,
		escapedTableID, whereClause)

	return query
}

// updateLastLoggingTime updates the watermark with the latest logging_time from events

func (m *MetricSet) updateLastLoggingTime(events []mb.Event) {
	if len(events) == 0 {
		return
	}

	// Since query is sorted by logging_time ASC, the last event has the latest time
	lastEvent := events[len(events)-1]
	if loggingTimeField, exists := lastEvent.MetricSetFields["logging_time"]; exists {
		if loggingTime, ok := loggingTimeField.(time.Time); ok && !loggingTime.IsZero() {
			// Store in UTC for consistency
			utcTime := loggingTime.UTC()
			m.lastLoggingTime = &utcTime
			m.logger.Debugf("Updated last logging time to: %s", loggingTime.Format(time.RFC3339))
		}
	}
}
