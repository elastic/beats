// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/mssql"
)

func init() {
	mb.Registry.MustAddMetricSet("mssql", "server", New,
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
	cfgwarn.Experimental("The mssql os metricset is experimental.")

	logger := logp.NewLogger("mssql.server").With("host", base.HostData().SanitizedURI)

	fetcher, err := mssql.NewFetcher(base.HostData().URI, []string{
		"SELECT * FROM sys.dm_server_services;",
		"SELECT * FROM sys.dm_server_memory_dumps;", // TODO Does not return anything?
	}, &schema, logger)
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
