// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/oracle"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("oracle", "performance", New,
		mb.WithHostParser(oracle.HostParser))
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	extractor         performanceExtractMethods
	connectionDetails oracle.ConnectionDetails
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := oracle.ConnectionDetails{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrap(err, "error parsing config file")
	}

	return &MetricSet{
		BaseMetricSet:     base,
		connectionDetails: config,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	db, err := oracle.NewConnection(&m.connectionDetails)
	if err != nil {
		return errors.Wrap(err, "error creating connection to Oracle")
	}
	defer db.Close()

	m.extractor = &performanceExtractor{db: db}

	events, err := m.extractAndTransform(ctx)
	if err != nil {
		return err
	}

	for _, event := range events {
		if reported := reporter.Event(event); !reported {
			return nil
		}
	}

	return nil
}
