// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package carbon

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil" //nolint:typecheck // civil is used for type casting
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
	metricsetName = "carbon"
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
	TableName           string        `config:"table_name"`
}

// Validate checks for deprecated config options
func (c config) Validate() error {
	if c.CredentialsFilePath == "" && c.CredentialsJSON == "" {
		return errors.New("no credentials_file_path or credentials_json specified")
	}

	if c.Period.Hours() < 24 {
		return fmt.Errorf("collection period for carbon footprint metricset %s cannot be less than 24 hours", c.Period)
	}
	return nil
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
		return nil, fmt.Errorf("unpack carbon footprint config failed: %w", err)
	}

	m.Logger().Debugf("metricset config: %v", m.config)
	return m, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (err error) {
	// find current month
	month := getReportMonth(time.Now())

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

	if m.config.TableName == "" {
		m.logger.Warn("table_name is not set in config, \"carbon_footprint\" will be used by default.")
		m.config.TableName = "carbon_footprint"
	}

	tableMeta, err := getTable(ctx, client, m.config.DatasetID, m.config.TableName)
	if err != nil {
		return fmt.Errorf("getTables failed: %w", err)
	}

	var events []mb.Event
	eventsPerQuery, err := m.queryBigQuery(ctx, client, tableMeta, month)
	if err != nil {
		return fmt.Errorf("queryBigQuery failed: %w", err)
	}

	events = append(events, eventsPerQuery...)

	m.Logger().Debugf("Total %d of events are created for carbon footprint", len(events))
	for _, event := range events {
		reporter.Event(event)
	}
	return nil
}

// getReportMonth gets the year-month of the latest expected report.
// GCP creates new reports on the 15 of each month. So if the date is below
// that, we fetch the previous month.
func getReportMonth(now time.Time) string {
	if now.Day() < 15 {
		now = now.AddDate(0, -1, 0)
	}
	return fmt.Sprintf("%04d-%02d-01", now.Year(), int(now.Month()))
}

type tableMeta struct {
	tableFullID string
	location    string
}

func getTable(ctx context.Context, client *bigquery.Client, datasetID string, tableName string) (*tableMeta, error) {
	dit := client.Datasets(ctx)

	for {
		dataset, err := dit.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		meta, err := client.Dataset(dataset.DatasetID).Metadata(ctx)
		if err != nil {
			return nil, err
		}

		// compare with given dataset_id
		if dataset.DatasetID != datasetID {
			continue
		}

		tit := dataset.Tables(ctx)
		for {
			table, err := tit.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, err
			}

			if table.TableID == tableName {
				return &tableMeta{
					tableFullID: table.ProjectID + "." + table.DatasetID + "." + table.TableID,
					location:    meta.Location,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find table '%s'", tableName)
}

func (m *MetricSet) queryBigQuery(ctx context.Context, client *bigquery.Client, tableMeta *tableMeta, month string) ([]mb.Event, error) {
	var events []mb.Event

	query := generateQuery(tableMeta.tableFullID, month)
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
	if err != nil {
		return events, err
	}
	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			err = fmt.Errorf("bigquery RowIterator Next failed: %w", err)
			m.logger.Error(err)
			return events, err
		}

		if len(row) == 12 {
			events = append(events, createEvents(row, m.config.ProjectID))
		}
	}
	return events, nil
}

func createEvents(rowItems []bigquery.Value, projectID string) mb.Event {
	event := mb.Event{}
	event.MetricSetFields = mapstr.M{
		"project_id":          rowItems[1],
		"project_name":        rowItems[2],
		"service_id":          rowItems[4],
		"service_description": rowItems[5],
		"region":              rowItems[6],

		"footprint.scope1":          rowItems[7],
		"footprint.scope2.location": rowItems[8],
		"footprint.scope2.market":   rowItems[9],
		"footprint.scope3":          rowItems[10],
		"footprint.offsets":         rowItems[11],
	}

	event.RootFields = mapstr.M{
		"cloud.provider":     "gcp",
		"cloud.project.id":   projectID,
		"cloud.project.name": rowItems[2],
		"cloud.account.id":   rowItems[3],
	}

	event.ID = generateEventID(rowItems)
	return event
}

func generateEventID(rowItems []bigquery.Value) string {
	// create eventID using hash of usage_month + project.id + project.name + service.description + region
	// This will prevent more than one carbon metric getting collected for the same month.
	eventID := rowItems[0].(civil.Date).String() +
		rowItems[1].(string) +
		rowItems[2].(string) +
		rowItems[3].(string) +
		rowItems[5].(string)

	h := sha256.New()
	h.Write([]byte(eventID))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:20]
}

// generateQuery returns the query to be used by the BigQuery client to retrieve monthly
// cost types breakdown.
func generateQuery(tableName, month string) string {
	// The table name is user provided, so it may contains special characters.
	// In order to allow any character in the table identifier, use the Quoted identifier format.
	// See https://github.com/elastic/beats/issues/26855
	// NOTE: is not possible to escape backtics (`) in a multiline string
	escapedTableName := fmt.Sprintf("`%s`", tableName)
	query := fmt.Sprintf(`
SELECT
	usage_month,
	project.number,
	project.id,
	billing_account_id,
	service.id,
	service.description,
	location.region,

	carbon_footprint_kgCO2e.scope1,
	carbon_footprint_kgCO2e.scope2.location_based,
	carbon_footprint_kgCO2e.scope2.market_based,
	carbon_footprint_kgCO2e.scope3,
	carbon_offsets_kgCO2e
FROM %s
WHERE project.id IS NOT NULL
AND usage_month = '%s'
ORDER BY usage_month ASC;`, escapedTableName, month)
	return query
}
