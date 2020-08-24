// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/pkg/errors"
	"google.golang.org/api/option"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/googlecloud"
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
	mb.Registry.MustAddMetricSet(googlecloud.ModuleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	config         config
	bigQueryConfig bigQueryConfig
	logger         *logp.Logger
}

type config struct {
	ProjectID           string `config:"project_id" validate:"required"`
	CredentialsFilePath string `config:"credentials_file_path"`
	period              *duration.Duration
}

// bigQueryConfig holds a configuration specific for billing metricset.
type bigQueryConfig struct {
	DatasetID    string `config:"dataset_id" validate:"required"`
	TablePattern string `config:"table_pattern" validate:"required"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The gcp '%s' metricset is beta.", metricsetName)

	m := &MetricSet{BaseMetricSet: base}
	if err := base.Module().UnpackConfig(&m.config); err != nil {
		return nil, err
	}

	m.config.period = &duration.Duration{
		Seconds: int64(m.Module().Config().Period.Seconds()),
	}

	m.logger = logp.NewLogger(metricsetName)
	return m, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (err error) {
	m.Logger().Debugf("billing config: %v", m.config)

	// find current month
	month := getCurrentMonth()

	opt := []option.ClientOption{option.WithCredentialsFile(m.config.CredentialsFilePath)}
	client, err := bigquery.NewClient(ctx, m.config.ProjectID, opt...)
	if err != nil {
		return errors.Wrap(err, "error creating bigquery client")
	}

	defer client.Close()

	tableMetas, err := getTables(ctx, client, m.bigQueryConfig.DatasetID, m.bigQueryConfig.TablePattern)
	if err != nil {
		return errors.Wrap(err, "getTables failed")
	}

	for _, tableMeta := range tableMetas {
		query := `
		SELECT
		  	invoice.month,
		  	project.id,
		  	cost_type,
		  SUM(cost)
			+ SUM(IFNULL((SELECT SUM(c.amount)
						  FROM UNNEST(credits) c), 0))
			AS total,
		  (SUM(CAST(cost * 1000000 AS int64))
			+ SUM(IFNULL((SELECT SUM(CAST(c.amount * 1000000 as int64))
						  FROM UNNEST(credits) c), 0))) / 1000000
			AS total_exact
		FROM ` + tableMeta.tableFullID + `
		WHERE project.id != 'null'
		AND invoice.month = ` + month + `
		AND cost_type = 'regular'
		GROUP BY 1, 2, 3
		ORDER BY 1 ASC, 2 ASC, 3 ASC
	`
		q := client.Query(query)

		// Location must match that of the dataset(s) referenced in the query.
		q.Location = tableMeta.location
		// Run the query and print results when the query job is completed.
		job, err := q.Run(ctx)
		if err != nil {
			return err
		}
		status, err := job.Wait(ctx)
		if err != nil {
			return err
		}
		if err := status.Err(); err != nil {
			return err
		}
		it, err := job.Read(ctx)
		for {
			var row []bigquery.Value
			err := it.Next(&row)
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			createEvents(row)
		}
	}

	// m.Logger().Debugf("Total %d of events are created for billing", len(events))
	//for _, event := range events {
	//	reporter.Event(event)
	//}
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

func createEvents(rows []bigquery.Value) {
	for _, row := range rows {
		fmt.Println("row = ", row)
	}
}
