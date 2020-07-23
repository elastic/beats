// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package status fetches MySQL server status metrics.

For more information on the query it uses, see:
http://dev.mysql.com/doc/refman/5.7/en/show-status.html
*/
package performance

import (
	"context"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/helper/sql"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mysql"
	"github.com/elastic/beats/v7/metricbeat/module/mysql/query"
)

func init() {
	mb.Registry.MustAddMetricSet("mysql", "performance", New,
		mb.WithHostParser(mysql.ParseDSN),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching MySQL server status.
type MetricSet struct {
	mb.BaseMetricSet
	db     *sql.DbClient
	config struct {
		Queries   []query.Query `config:"queries" validate:"nonzero,required"`
		Namespace string        `config:"namespace" validate:"nonzero,required"`
	}
}

// New creates and returns a new MetricSet instance.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The mysql 'performance' metricset is beta.")

	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch fetches status messages from a mysql host.
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	if m.db == nil {
		var err error
		m.db, err = sql.NewDBClient("mysql", m.HostData().URI, m.Logger())
		if err != nil {
			return errors.Wrap(err, "mysql-status fetch failed")
		}
	}

	err := m.fetchEventsStatements(ctx, reporter)
	if err != nil {
		return err
	}

	err = m.fetchTableIoWaits(ctx, reporter)
	if err != nil {
		return err
	}

	return nil
}

func (m *MetricSet) fetchEventsStatements(ctx context.Context, reporter mb.ReporterV2) error {
	mss, err := m.db.FetchTableMode(ctx, `SELECT digest_text,
		count_star,
		avg_timer_wait,
		max_timer_wait,
		last_seen,
		quantile_95
		FROM performance_schema.events_statements_summary_by_digest
		ORDER BY avg_timer_wait DESC
		LIMIT 10`)
	if err != nil {
		return err
	}
	for _, ms := range mss {
		replaceUnderscores := true
		event := query.TransformMapStrToEvent(ms, "performance", "events_statements", replaceUnderscores)
		reporter.Event(event)
	}

	return nil
}

func (m *MetricSet) fetchTableIoWaits(ctx context.Context, reporter mb.ReporterV2) error {
	mss, err := m.db.FetchTableMode(ctx, `SELECT object_schema, object_name, index_name, count_fetch
          FROM performance_schema.table_io_waits_summary_by_index_usage
          WHERE count_fetch > 0`)
	if err != nil {
		return err
	}
	for _, ms := range mss {
		replaceUnderscores := true
		event := query.TransformMapStrToEvent(ms, "performance", "table_io_waits", replaceUnderscores)
		reporter.Event(event)
	}

	return nil
}

// Close closes the database connection and prevents future queries.
func (m *MetricSet) Close() error {
	if m.db == nil {
		return nil
	}
	return errors.Wrap(m.db.Close(), "failed to close mysql database client")
}
