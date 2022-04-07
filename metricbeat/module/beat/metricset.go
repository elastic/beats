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

package beat

import (
	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// MetricSet can be used to build other metricsets within the Beat module.
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
	XPackEnabled bool
}

// NewMetricSet creates a metricset that can be used to build other metricsets
// within the Beat module.
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	config := DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	ms := &MetricSet{
		base,
		http,
		config.XPackEnabled,
	}

	return ms, nil
}
