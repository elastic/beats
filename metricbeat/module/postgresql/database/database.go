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

package database

import (
	"context"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	// Register postgresql database/sql driver
	_ "github.com/lib/pq"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("postgresql", "database", New,
		mb.WithHostParser(postgresql.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*postgresql.MetricSet
}

// New create a new instance of the postgresql database MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := postgresql.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	ctx := context.Background()
	results, err := m.QueryStats(ctx, "SELECT * FROM pg_stat_database")
	if err != nil {
		return errors.Wrap(err, "error in QueryStats")
	}

	for _, result := range results {
		data, _ := schema.Apply(result)
		reporter.Event(mb.Event{
			MetricSetFields: data,
		})
	}

	return nil
}
