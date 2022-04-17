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

package openmetrics

import (
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	// HostParser validates OpenMetrics URLs
	HostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

// MetricSetBuilder returns a builder function for a new OpenMetrics metricset using the given mapping
func MetricSetBuilder(mapping *MetricsMapping) func(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		openmetrics, err := NewOpenMetricsClient(base)
		if err != nil {
			return nil, err
		}
		return &openmetricsMetricSet{
			BaseMetricSet: base,
			openmetrics:   openmetrics,
			mapping:       mapping,
		}, nil
	}
}

type openmetricsMetricSet struct {
	mb.BaseMetricSet
	openmetrics OpenMetrics
	mapping     *MetricsMapping
}

func (m *openmetricsMetricSet) Fetch(r mb.ReporterV2) error {
	return m.openmetrics.ReportProcessedMetrics(m.mapping, r)
}
