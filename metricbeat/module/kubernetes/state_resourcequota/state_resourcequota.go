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

package state_resourcequota

import (
	"fmt"

	p "github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	k8smod "github.com/menderesk/beats/v7/metricbeat/module/kubernetes"
)

func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "state_resourcequota",
		NewResourceQuotaMetricSet,
		mb.WithHostParser(p.HostParser))
}

// ResourceQuotaMetricSet is a prometheus based MetricSet that looks for
// mb.ModuleDataKey prefixed fields and puts then at the module level
//
// Copying the code from other kube state metrics, this should be improved to
// avoid all these ugly tricks
type ResourceQuotaMetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
	mapping    *p.MetricsMapping
	mod        k8smod.Module
}

// NewResourceQuotaMetricSet returns a prometheus based metricset for ResourceQuotas
func NewResourceQuotaMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := p.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}
	mod, ok := base.Module().(k8smod.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of kubernetes module")
	}
	return &ResourceQuotaMetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
		mod:           mod,
		mapping: &p.MetricsMapping{
			Metrics: map[string]p.MetricMap{
				"kube_resourcequota_created": p.Metric("created.sec"),
				"kube_resourcequota":         p.Metric("quota"),
			},
			Labels: map[string]p.LabelMap{
				"namespace":     p.KeyLabel(mb.ModuleDataKey + ".namespace"),
				"resourcequota": p.KeyLabel("name"),

				"resource": p.KeyLabel("resource"),
				"type":     p.KeyLabel("type"),
			},
		},
	}, nil
}

// Fetch prometheus metrics and treats those prefixed by mb.ModuleDataKey as
// module rooted fields at the event that gets reported
func (m *ResourceQuotaMetricSet) Fetch(reporter mb.ReporterV2) {
	families, err := m.mod.GetStateMetricsFamilies(m.prometheus)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}
	events, err := m.prometheus.ProcessMetrics(families, m.mapping)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}

	for _, event := range events {
		event[mb.NamespaceKey] = "resourcequota"
		reported := reporter.Event(mb.TransformMapStrToEvent("kubernetes", event, nil))
		if !reported {
			m.Logger().Debug("error trying to emit event")
			return
		}
	}
	return
}
