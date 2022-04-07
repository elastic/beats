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

package pool

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/php_fpm"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("php_fpm", "pool", New,
		mb.WithHostParser(php_fpm.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		http,
	}, nil
}

// Fetch gathers data for the pool metricset
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	content, err := m.HTTP.FetchContent()
	if err != nil {
		return errors.Wrap(err, "error in http fetch")
	}
	var stats map[string]interface{}
	err = json.Unmarshal(content, &stats)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling json")
	}
	event, err := schema.Apply(stats)
	if err != nil {
		return errors.Wrap(err, "error in event mapping")
	}
	reporter.Event(mb.Event{
		MetricSetFields: event,
	})
	return nil
}
