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

type metricsetConfig struct {
	MetricsFilters  MetricFilters `config:"metrics_filters" yaml:"metrics_filters,omitempty"`
	EnableExemplars bool          `config:"enable_exemplars" yaml:"enable_exemplars,omitempty"`
	EnableMetadata  bool          `config:"enable_metadata" yaml:"enable_metadata,omitempty"`
}

type MetricFilters struct {
	IncludeMetrics *[]string `config:"include" yaml:"include,omitempty"`
	ExcludeMetrics *[]string `config:"exclude" yaml:"exclude,omitempty"`
}

var defaultConfig = metricsetConfig{
	MetricsFilters: MetricFilters{
		IncludeMetrics: nil,
		ExcludeMetrics: nil},
	EnableExemplars: false,
	EnableMetadata:  false,
}

func (c *metricsetConfig) Validate() error {
	// validate configuration here
	return nil
}
