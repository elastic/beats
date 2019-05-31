// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/oracle"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("oracle", "tablespace", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	db        *sql.DB
	extractor tablespaceExtractMethods
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The oracle 'tablespace' metricset is experimental.")

	config := oracle.ConnectionDetails{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.Wrap(err, "error parsing config file")
	}

	db, err := oracle.NewConnection(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error creating connection to Oracle")
	}

	return &MetricSet{
		BaseMetricSet: base,
		db:            db,
		extractor:     &tablespaceExtractor{db: db},
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	events, err := m.eventMapping()
	if err != nil {
		return err
	}
	for _, event := range events {
		if reported := reporter.Event(event); !reported {
			m.Logger().Debug("event wasn't reported")
		}
	}

	return nil
}

// Close the connection to Oracle
func (m *MetricSet) Close() error {
	return m.db.Close()
}
