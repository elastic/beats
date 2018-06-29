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

package apiserver

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"apiserver_request_count":     prometheus.Metric("request.count"),
			"apiserver_request_latencies": prometheus.Metric("request.latency"),
		},

		Labels: map[string]prometheus.LabelMap{
			"client":      prometheus.KeyLabel("request.client"),
			"resource":    prometheus.KeyLabel("request.resource"),
			"scope":       prometheus.KeyLabel("request.scope"),
			"subresource": prometheus.KeyLabel("request.subresource"),
			"verb":        prometheus.KeyLabel("request.verb"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "apiserver",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
