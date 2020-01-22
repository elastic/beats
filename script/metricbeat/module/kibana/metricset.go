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

package kibana

import (
	"github.com/elastic/beats/metricbeat/mb"
)

// MetricSet can be used to build other metricsets within the Kibana module.
type MetricSet struct {
	mb.BaseMetricSet
	XPackEnabled bool
}

// NewMetricSet creates a metricset that can be used to build other metricsets
// within the Kibana module.
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	config := DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		config.XPackEnabled,
	}, nil
}
