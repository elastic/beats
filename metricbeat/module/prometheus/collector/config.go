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

package collector

import (
	"github.com/elastic/beats/libbeat/logp"
)

type metricsetConfig struct {
	MetricsFilters MetricFilters `config:"metrics_filters"`
}

type MetricFilters struct {
	IncludeMetrics *[]string `config:"include_metrics"`
	IgnoreMetrics  *[]string `config:"ignore_metrics"`
}

var defaultConfig = metricsetConfig{
	MetricsFilters: MetricFilters{
		IncludeMetrics: nil,
		IgnoreMetrics:  nil},
}

func (c *metricsetConfig) Validate() error {
	if c.MetricsFilters.IncludeMetrics != nil && c.MetricsFilters.IgnoreMetrics == nil {
		logp.Debug("prometheus", "include_metrics and ignore_metrics are complementary and cannot be used together. Falling back to using only include_metrics")
		c.MetricsFilters.IgnoreMetrics = nil
	}
	return nil
}
