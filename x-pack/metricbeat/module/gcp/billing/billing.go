// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	// metricsetName is the name of this metricset
	metricsetName = "billing"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(gcp.ModuleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config config
	logger *logp.Logger
}

type config struct {
	Period              time.Duration `config:"period" validate:"required"`
	ProjectID           string        `config:"project_id" validate:"required"`
	CredentialsFilePath string        `config:"credentials_file_path"`
	CredentialsJSON     string        `config:"credentials_json"`
	DatasetID           string        `config:"dataset_id" validate:"required"`
	TablePattern        string        `config:"table_pattern"`
	CostType            string        `config:"cost_type"`
}

// Validate checks for deprecated config options
func (c config) Validate() error {
	if c.CredentialsFilePath == "" && c.CredentialsJSON == "" {
		return errors.New("no credentials_file_path or credentials_json specified")
	}

	if c.CostType != "" {
		// cost_type can only be regular, tax, adjustment, or rounding error
		costTypes := []string{"regular", "tax", "adjustment", "rounding error"}
		if stringInSlice(c.CostType, costTypes) {
			return nil
		}
		return fmt.Errorf("given cost_type %s is not in supported list %s", c.CostType, costTypes)
	}

	if c.Period.Hours() < 24 {
		return fmt.Errorf("collection period for billing metricset %s cannot be less than 24 hours", c.Period)
	}
	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The gcp '%s' metricset is beta.", metricsetName)

	m := &MetricSet{
		BaseMetricSet: base,
		logger:        logp.NewLogger(metricsetName),
	}

	if err := base.Module().UnpackConfig(&m.config); err != nil {
		return nil, fmt.Errorf("unpack billing config failed: %w", err)
	}

	m.Logger().Debugf("metricset config: %v", m.config)
	return m, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (err error) {
	// find current month
	month := getCurrentMonth()

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
		return fmt.Errorf("gerror creating bigquery client: %w", err)
	}

	defer client.Close()

	// default table_pattern for query is "gcp_billing_export_v1"
	if m.config.TablePattern == "" {
		m.logger.Warn("table_pattern is not set in config, \"gcp_billing_export_v1\" will be used by default.")
		m.config.TablePattern = "gcp_billing_export_v1"
	}

	// default cost_type for query is "regular"
	if m.config.CostType == "" {
		m.logger.Warn("cost_type is not set in config, \"regular\" will be used by default.")
		m.config.CostType = "regular"
	}

	tableMetas, err := getTables(ctx, client, m.config.DatasetID, m.config.TablePattern)
	if err != nil {
		return fmt.Errorf("getTables failed: %w", err)
	}

	var events []mb.Event
	for _, tableMeta := range tableMetas {
		eventsPerQuery, err := m.queryBigQuery(ctx, client, tableMeta, month, m.config.CostType)
		if err != nil {
			return fmt.Errorf("queryBigQuery failed: %w", err)
		}

		events = append(events, eventsPerQuery...)
	}

	m.Logger().Debugf("Total %d of events are created for billing", len(events))
	for _, event := range events {
		reporter.Event(event)
	}
	return nil
}

func getCurrentMonth() string {
	currentTime := time.Now()
	return fmt.Sprintf("%04d%02d", currentTime.Year(), int(currentTime.Month()))
}

type tableMeta struct {
	tableFullID string
	location    string
}

func getTables(ctx context.Context, client *bigquery.Client, datasetID string, tablePattern string) ([]tableMeta, error) {
	dit := client.Datasets(ctx)
	var tables []tableMeta

	for {
		dataset, err := dit.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return tables, err
		}

		meta, err := client.Dataset(dataset.DatasetID).Metadata(ctx)
		if err != nil {
			return tables, err
		}

		// compare with given dataset_id
		if dataset.DatasetID != datasetID {
			continue
		}

		tit := dataset.Tables(ctx)
		for {
			var tableMeta tableMeta
			table, err := tit.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return tables, err
			}

			// make sure table ID fits the given table_pattern
			if strings.HasPrefix(table.TableID, tablePattern) {
				tableMeta.tableFullID = table.ProjectID + "." + table.DatasetID + "." + table.TableID
				tableMeta.location = meta.Location
				tables = append(tables, tableMeta)
			}
		}
	}
	return tables, nil
}

func (m *MetricSet) queryBigQuery(ctx context.Context, client *bigquery.Client, tableMeta tableMeta, month string, costType string) ([]mb.Event, error) {
	var events []mb.Event

	query := generateQuery(tableMeta.tableFullID, month, costType)
	m.logger.Debug("bigquery query = ", query)

	q := client.Query(query)

	// Location must match that of the dataset(s) referenced in the query.
	q.Location = tableMeta.location

	// Run the query and print results when the query job is completed.
	job, err := q.Run(ctx)
	if err != nil {
		err = fmt.Errorf("bigquery Run failed: %w", err)
		m.logger.Error(err)
		return events, err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		err = fmt.Errorf("bigquery Wait failed: %w", err)
		m.logger.Error(err)
		return events, err
	}

	if err := status.Err(); err != nil {
		err = fmt.Errorf("bigquery status error: %w", err)
		m.logger.Error(err)
		return events, err
	}

	it, err := job.Read(ctx)
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			err = fmt.Errorf("bigquery RowIterator Next failed: %w", err)
			m.logger.Error(err)
			return events, err
		}

		if len(row) == 6 {
			events = append(events, createEvents(row, m.config.ProjectID))
		}
	}
	return events, nil
}

func createEvents(rowItems []bigquery.Value, projectID string) mb.Event {
	event := mb.Event{}
	event.MetricSetFields = mapstr.M{
		"invoice_month":      rowItems[0],
		"project_id":         rowItems[1],
		"project_name":       rowItems[2],
		"billing_account_id": rowItems[3],
		"cost_type":          rowItems[4],
		"total":              rowItems[5],
	}

	event.RootFields = mapstr.M{
		"cloud.provider":     "gcp",
		"cloud.project.id":   projectID,
		"cloud.project.name": rowItems[2],
		"cloud.account.id":   rowItems[3],
	}

	// create eventID for each current_date + invoice_month + project_id + cost_type
	currentDate := getCurrentDate()
	event.ID = generateEventID(currentDate, rowItems)
	return event
}

func getCurrentDate() string {
	currentTime := time.Now()
	return fmt.Sprintf("%04d%02d%02d", currentTime.Year(), int(currentTime.Month()), currentTime.Day())
}

func generateEventID(currentDate string, rowItems []bigquery.Value) string {
	// create eventID using hash of current_date + invoice.month + project.id + project.name
	// This will prevent more than one billing metric getting collected in the same day.
	eventID := currentDate + rowItems[0].(string) + rowItems[1].(string) + rowItems[2].(string)
	h := sha256.New()
	h.Write([]byte(eventID))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:20]
}

// generateQuery returns the query to be used by the BigQuery client to retrieve monthly
// cost types breakdown.
func generateQuery(tableName, month, costType string) string {
	// The table name is user provided, so it may contains special characters.
	// In order to allow any character in the table identifier, use the Quoted identifier format.
	// See https://github.com/elastic/beats/issues/26855
	// NOTE: is not possible to escape backtics (`) in a multiline string
	escapedTableName := fmt.Sprintf("`%s`", tableName)
	query := fmt.Sprintf(`
SELECT
	invoice.month,
	project.id,
	project.name,
	billing_account_id,
	cost_type,
	(SUM(CAST(cost * 1000000 AS int64))
	+ SUM(IFNULL((SELECT SUM(CAST(c.amount * 1000000 as int64)) FROM UNNEST(credits) c), 0))) / 1000000
	AS total_exact
FROM %s
WHERE project.id IS NOT NULL
AND invoice.month = '%s'
AND cost_type = '%s'
GROUP BY 1, 2, 3, 4, 5
ORDER BY 1 ASC, 2 ASC, 3 ASC, 4 ASC, 5 ASC;`, escapedTableName, month, costType)
	return query
}
