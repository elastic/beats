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

package cpu

import (
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// Metricset for apiserver is a prometheus based metricset
type metricset struct {
	mb.BaseMetricSet
	prometheusClient     prometheus.Prometheus
	prometheusMappings   *prometheus.MetricsMapping
	preSystemCpuUsage    float64
	preContainerCpuUsage map[string]float64
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
			BaseMetricSet:        base,
			prometheusClient:     pc,
			prometheusMappings:   prometheusMappings,
			preSystemCpuUsage:    0.0,
			preContainerCpuUsage: map[string]float64{},
		}, nil
	}
}

// Fetch gathers information from the containerd and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	events, err := m.prometheusClient.GetProcessedMetrics(m.prometheusMappings)
	if err != nil {
		return errors.Wrap(err, "error getting metrics")
	}
	var systemTotalNs int64
	elToDel := -1
	for i, event := range events {
		systemTotalSeconds, err := event.GetValue("system.total")
		if err == nil {
			systemTotalNs = systemTotalSeconds.(int64) * 1000000000
			elToDel = i
			break
		}
	}
	if elToDel != -1 {
		events[elToDel] = events[len(events)-1] // Copy last element to index i.
		events[len(events)-1] = common.MapStr{} // Erase last element (write empty value).
		events = events[:len(events)-1]
	}

	for _, event := range events {
		// applying ECS to kubernetes.container.id in the form <container.runtime>://<container.id>
		// copy to ECS fields the kubernetes.container.image, kubernetes.container.name
		containerFields := common.MapStr{}
		var cID string
		if containerID, ok := event["id"]; ok {
			// we don't expect errors here, but if any we would obtain an
			// empty string
			cID = (containerID).(string)
			containerFields.Put("id", cID)
			event.Delete("id")
		}
		e, err := util.CreateEvent(event, "containerd.cpu")
		if err != nil {
			m.Logger().Error(err)
		}

		if len(containerFields) > 0 {
			if e.RootFields != nil {
				e.RootFields.DeepUpdate(common.MapStr{
					"container": containerFields,
				})
			} else {
				e.RootFields = common.MapStr{
					"container": containerFields,
				}
			}
		}
		cpuUsageTotal, err := event.GetValue("usage.total.ns")
		if err == nil {
			var contUsageDelta, systemUsageDelta, cpuUsagePct float64
			if cpuPreval, ok := m.preContainerCpuUsage[cID]; ok {
				contUsageDelta = cpuUsageTotal.(float64) - cpuPreval
				systemUsageDelta = float64(systemTotalNs) - m.preSystemCpuUsage
				m.Logger().Infof("contUsageDelta is %+v - %+v == %+v", cpuUsageTotal, cpuPreval, contUsageDelta)
				m.Logger().Infof("systemUsageDelta is %+v - %+v == %+v", systemTotalNs, m.preSystemCpuUsage, systemUsageDelta)
			} else {
				contUsageDelta = cpuUsageTotal.(float64)
				systemUsageDelta = float64(systemTotalNs)
				m.Logger().Infof("contUsageDelta is %+v - %+v == %+v", cpuUsageTotal, cpuPreval, contUsageDelta)
				m.Logger().Infof("systemUsageDelta is %+v - %+v == %+v", systemTotalNs, m.preSystemCpuUsage, systemUsageDelta)
			}
			if contUsageDelta == 0.0 || systemUsageDelta == 0.0 {
				m.Logger().Infof("SOMETHING IS ZERO")
				cpuUsagePct = 0.0
			} else {
				cpuUsagePct = (contUsageDelta / systemUsageDelta) * 100
			}
			m.Logger().Infof("cpuUsagePct for %+v is %+v", cID, cpuUsagePct)
			e.MetricSetFields.Put("usage.total.pct", cpuUsagePct)
			//Update values
			m.preContainerCpuUsage[cID] = cpuUsageTotal.(float64)
		}
		if reported := reporter.Event(e); !reported {
			return nil
		}
	}
	m.preSystemCpuUsage = float64(systemTotalNs)
	m.Logger().Infof("preContainerCpuUsage is %+v", m.preContainerCpuUsage)
	m.Logger().Infof("preSystemCpuUsage is %+v", m.preSystemCpuUsage)
	return nil
}
