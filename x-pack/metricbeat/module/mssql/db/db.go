// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package db

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/mssql"
)

func init() {
	mb.Registry.MustAddMetricSet("mssql", "db", New,
		mb.DefaultMetricSet(),
		mb.WithHostParser(mssql.HostParser))
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	log     *logp.Logger
	fetcher *mssql.Fetcher
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The mssql db metricset is beta.")

	logger := logp.NewLogger("mssql.db").With("host", base.HostData().SanitizedURI)

	fetcher, err := mssql.NewFetcher(base.HostData().URI,
		[]string{"SELECT * FROM sys.dm_db_log_space_usage;"}, &schema, logger)
	if err != nil {
		return nil, errors.Wrap(err, "error creating fetcher")
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logger,
		fetcher:       fetcher,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	m.fetcher.Report(reporter)
}

// Close the connection to the server at the engine level
func (m *MetricSet) Close() error {
	return m.fetcher.Close()
}
