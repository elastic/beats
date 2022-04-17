// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"fmt"
	"time"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/oracle"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("oracle", "tablespace", New,
		mb.WithHostParser(oracle.HostParser))
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	extractor         tablespaceExtractMethods
	connectionDetails oracle.ConnectionDetails
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := oracle.ConnectionDetails{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Warn the user if the collection period value is less than 1 minute.
	if CheckCollectionPeriod(base.Module().Config().Period) {
		base.Logger().Warn("The current value of period is significantly low and might waste cycles and resources. Please set the period value to at least 1 minute or more.")
	}

	return &MetricSet{
		BaseMetricSet:     base,
		connectionDetails: config,
	}, nil
}

// CheckCollectionPeriod method returns true if the period is less than 1 minute.
func CheckCollectionPeriod(period time.Duration) bool {
	return period < time.Minute
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (err error) {
	db, err := oracle.NewConnection(&m.connectionDetails)
	if err != nil {
		return fmt.Errorf("error creating connection to Oracle: %w", err)
	}
	defer db.Close()

	m.extractor = &tablespaceExtractor{db: db}

	events, err := m.extractAndTransform(ctx)
	if err != nil {
		return fmt.Errorf("error getting or interpreting data from Oracle: %w", err)
	}

	m.Load(ctx, events, reporter)

	return err
}

//Load is the L of an ETL. In this case, takes the events and sends them to Elasticseach
func (m *MetricSet) Load(ctx context.Context, events []mb.Event, reporter mb.ReporterV2) {
	for _, event := range events {
		if reported := reporter.Event(event); !reported {
			return
		}
	}
}
