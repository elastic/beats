// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("oracle", "sysmetric", New,
		mb.WithHostParser(oracle.HostParser))
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	extractor         sysmetricExtractMethod
	connectionDetails oracle.ConnectionDetails
	Patterns          []string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := oracle.ConnectionDetails{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("error parsing config file %w", err)
	}
	return &MetricSet{
		BaseMetricSet:     base,
		connectionDetails: config,
		Patterns:          config.Patterns,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) (err error) {
	db, err := oracle.NewConnection(&m.connectionDetails)
	if err != nil {
		return fmt.Errorf("error creating connection to Oracle %w", err)
	}
	defer db.Close()

	m.extractor = &sysmetricExtractor{db: db, patterns: m.Patterns}

	events, err := m.extractAndTransform(ctx)
	if err != nil {
		return err
	}

	m.Load(ctx, events, reporter)

	return nil
}

//Load is the L of an ETL. In this case, takes the events and sends them to Elasticseach
func (m *MetricSet) Load(ctx context.Context, events []mb.Event, reporter mb.ReporterV2) {
	for _, event := range events {
		if reported := reporter.Event(event); !reported {
			return
		}
	}
}
