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

package mem

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/pgbouncer"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("pgbouncer", "mem", New,
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
	// Execute the "SHOW MEM;" query against the database.
	results, err := m.QueryStats(ctx, "SHOW MEM;")
	if err != nil {
		// Return the error if the query fails.
		return fmt.Errorf("error in QueryStats: %w", err)
	}

	// Initialize an empty map to store aggregated results.
	data := mapstr.M{}

	// Iterate over each result from the query.
	for _, result := range results {
		// Apply the predefined schema to the result to format it properly.
		tmpData, err := schema.Apply(result)
		if err != nil {
			// Log the error and skip this iteration if schema application fails.
			m.Logger().Errorf("Error applying schema: %v", err)
			continue
		}

		// Aggregate the formatted data into the data map.
		for k, v := range tmpData {
			data[k] = v
		}
	}

	// Check if there is any data collected.
	if len(data) > 0 {
		// Create and report an event with the collected data.
		reporter.Event(mb.Event{
			MetricSetFields: data,
		})
	}

	// Return nil to indicate successful completion.
	return nil
}
