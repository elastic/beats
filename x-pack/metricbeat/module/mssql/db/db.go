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

type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The mssql db metricset is experimental.")
	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger("mssql.db").With("host", base.HostData().SanitizedURI),
	}, nil
}

func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	fetcher, err := mssql.NewFetcher(m.HostData().URI,
		[]string{"SELECT * FROM sys.dm_db_log_space_usage;"}, &schema, m.log)
	if err != nil {
		reporter.Error(errors.Wrap(err, "error creating fetcher"))
		return
	}

	if fetcher.Error != nil {
		reporter.Error(fetcher.Error)
		return
	}

	for _, e := range fetcher.Results {
		reporter.Event(mb.Event{
			MetricSetFields: e,
		})
	}
}
