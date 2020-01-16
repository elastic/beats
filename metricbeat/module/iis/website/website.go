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

// +build windows

package website

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// Config for the windows perfmon metricset.
type Config struct {
	Hosts []string `config:"hosts"`
}

// init registers the partition MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("iis", "website", New)
}

// MetricSet type defines all fields of the partition MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	log *logp.Logger
}

// New creates a new instance of the website MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The perfmon metricset is beta")

	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		log:           logp.NewLogger("website"),
	}, nil
}

// Fetch fetches events and reports them upstream
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	return nil
}
