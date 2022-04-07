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
	"fmt"

	"github.com/elastic/beats/v8/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/kubernetes/util"
)

// Metricset for apiserver is a prometheus based metricset
type metricset struct {
	mb.BaseMetricSet
	prometheusClient   prometheus.Prometheus
	prometheusMappings *prometheus.MetricsMapping
}

var _ mb.ReportingMetricSetV2Error = (*metricset)(nil)

// getMetricsetFactory as required by` mb.Registry.MustAddMetricSet`
func getMetricsetFactory(prometheusMappings *prometheus.MetricsMapping) mb.MetricSetFactory {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		pc, err := prometheus.NewPrometheusClient(base)
		if err != nil {
			return nil, err
		}
		return &metricset{
			BaseMetricSet:      base,
			prometheusClient:   pc,
			prometheusMappings: prometheusMappings,
		}, nil
	}
}

// Fetch gathers information from the apiserver and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	events, err := m.prometheusClient.GetProcessedMetrics(m.prometheusMappings)
	if err != nil {
		return fmt.Errorf("error getting metrics: %w", err)
	}

	rcPost14 := false
	for _, event := range events {
		if ok, _ := event.HasKey("request.count"); ok {
			rcPost14 = true
			break
		}
	}

	for _, event := range events {
		// Hack: super ugly trick. An improvement would be to add pipeline/lifecycle
		// to metrics retrieval in general, so mappings, retrieved metrics, ... can be
		// modified on events. Current design is limiting.
		if ok, _ := event.HasKey("request.beforev14.count"); ok {
			if rcPost14 {
				if bothInformed, _ := event.HasKey("request.count"); !bothInformed {
					continue
				}
				util.ShouldDelete(event, "request.beforev14", m.Logger())
			} else {
				v, err := event.GetValue("request.beforev14.count")
				if err != nil {
					reporter.Error(err)
					continue
				}
				util.ShouldPut(event, "request.count", v, m.Logger())
				util.ShouldDelete(event, "request.beforev14", m.Logger())
			}
		}

		reporter.Event(mb.Event{
			MetricSetFields: event,
			Namespace:       m.prometheusMappings.Namespace,
		})
	}

	return nil
}
