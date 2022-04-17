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

package vsphere

import (
	"net/url"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

var HostParser = parse.URLHostParserBuilder{
	DefaultScheme: "https",
	DefaultPath:   "/sdk",
}.Build()

// MetricSet type defines all fields of the MetricSet.
type MetricSet struct {
	mb.BaseMetricSet
	Insecure bool
	HostURL  *url.URL
}

// NewMetricSet creates a new instance of the MetricSet.
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	config := struct {
		Insecure bool `config:"insecure"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	u, err := url.Parse(base.HostData().URI)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		HostURL:       u,
		Insecure:      config.Insecure,
	}, nil
}
