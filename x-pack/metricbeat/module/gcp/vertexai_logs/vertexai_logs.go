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
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

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
	config config
	logger *logp.Logger
}

type config struct {
	Period              time.Duration `config:"period" validate:"required"`
	ProjectID           string        `config:"project_id" validate:"required"`
	TableID             string        `config:"table_id" validate:"required"`
	CredentialsFilePath string        `config:"credentials_file_path"`
	CredentialsJSON     string        `config:"credentials_json"`
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

	m.logger.Debugf("metricset config: project_id=%s, dataset_id=%s, table_name=%s",
		m.config.ProjectID, getDatasetID(m.config.TableID), getTableName(m.config.TableID))
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

	query := fmt.Sprintf(`
SELECT
	IFNULL(endpoint, '') AS endpoint,
	IFNULL(deployed_model_id, '') AS deployed_model_id,
	logging_time,
	CAST(IFNULL(request_id, 0) AS FLOAT64) AS request_id,
	IFNULL(request_payload, []) AS request_payload,
	IFNULL(response_payload, []) AS response_payload,
	IFNULL(model, '') AS model,
	IFNULL(model_version, '') AS model_version,
	IFNULL(api_method, '') AS api_method,
	IFNULL(TO_JSON_STRING(full_request), '{}') AS full_request,
	IFNULL(TO_JSON_STRING(full_response), '{}') AS full_response,
	IFNULL(TO_JSON_STRING(metadata), '{}') AS metadata,
	IFNULL(TO_JSON_STRING(otel_log), '{}') AS otel_log
FROM
	%s
WHERE
	logging_time IS NOT NULL
ORDER BY
	logging_time DESC
LIMIT 10000;`,
		escapedTableID)

	return query
}
