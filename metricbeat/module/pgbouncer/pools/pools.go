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

package pools

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/pgbouncer"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("pgbouncer", "pools", New,
		mb.WithHostParser(pgbouncer.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*pgbouncer.MetricSet
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := pgbouncer.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It publishes the event which is then forwarded to the output. In case of an error, an error is reported.
func (m *MetricSet) Fetch(ctx context.Context, reporter mb.ReporterV2) error {
	results, err := m.QueryStats(ctx, "SHOW POOLS;")
	if err != nil {
		return fmt.Errorf("error in QueryStats: %w", err)
	}

	for _, result := range results {
		var data mapstr.M

		data, err := schema.Apply(result)
		if err != nil {
			return fmt.Errorf("error mapping result: %w", err)
		}
		reporter.Event(mb.Event{
			MetricSetFields: data,
		})
	}
	return nil
}
