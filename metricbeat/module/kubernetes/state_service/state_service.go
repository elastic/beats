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

package state_service

import (
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "state_service",
		NewServiceMetricSet,
		mb.WithHostParser(p.HostParser))
}

// ServiceMetricSet is a prometheus based MetricSet that looks for
// mb.ModuleDataKey prefixed fields and puts then at the module level
//
// Copying the code from other kube state metrics, this should be improved to
// avoid all these ugly tricks
type ServiceMetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
	mapping    *p.MetricsMapping
}

// NewServiceMetricSet returns a prometheus based metricset for Services
func NewServiceMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := p.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}

	return &ServiceMetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
		mapping: &p.MetricsMapping{
			Metrics: map[string]p.MetricMap{
				"kube_service_info": p.InfoMetric(),
				"kube_service_labels": p.ExtendedInfoMetric(
					p.Configuration{
						StoreNonMappedLabels:     true,
						NonMappedLabelsPlacement: mb.ModuleDataKey + ".labels",
						MetricProcessingOptions:  []p.MetricOption{p.OpLabelKeyPrefixRemover("label_")},
					}),
				"kube_service_created":                      p.Metric("created", p.OpUnixTimestampValue()),
				"kube_service_spec_type":                    p.InfoMetric(),
				"kube_service_spec_external_ip":             p.InfoMetric(),
				"kube_service_status_load_balancer_ingress": p.InfoMetric(),
			},
			Labels: map[string]p.LabelMap{
				"namespace":        p.KeyLabel(mb.ModuleDataKey + ".namespace"),
				"service":          p.KeyLabel("name"),
				"cluster_ip":       p.Label("cluster_ip"),
				"external_name":    p.Label("external_name"),
				"external_ip":      p.Label("external_ip"),
				"load_balancer_ip": p.Label("load_balancer_ip"),
				"type":             p.Label("type"),
				"ip":               p.Label("ingress_ip"),
				"hostname":         p.Label("ingress_hostname"),
			},
		},
	}, nil
}

// Fetch prometheus metrics and treats those prefixed by mb.ModuleDataKey as
// module rooted fields at the event that gets reported
func (m *ServiceMetricSet) Fetch(reporter mb.ReporterV2) {
	events, err := m.prometheus.GetProcessedMetrics(m.mapping)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}

	for _, event := range events {
		event[mb.NamespaceKey] = "service"
		reported := reporter.Event(mb.TransformMapStrToEvent("kubernetes", event, nil))
		if !reported {
			m.Logger().Debug("error trying to emit event")
			return
		}
	}
	return
}
