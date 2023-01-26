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

package state_container

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/autodiscover/providers/kubernetes"
	//"github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	k8smod "github.com/elastic/beats/v7/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"strings"
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{
		"kube_pod_info":           prometheus.InfoMetric(),
		"kube_pod_container_info": prometheus.InfoMetric(),

		"kube_pod_container_resource_requests": prometheus.Metric("resource.requests"),
		"kube_pod_container_resource_limits":   prometheus.Metric("resource.limits"),

		"kube_pod_container_status_ready":   prometheus.BooleanMetric("status.ready"),
		"kube_pod_container_status_running": prometheus.BooleanMetric("status.phase"),

		"kube_pod_container_status_restarts_total":         prometheus.Metric("status.restarts.count"),
		"kube_pod_container_status_waiting_reason":         prometheus.Metric("status.reason"),
		"kube_pod_container_status_terminated_reason":      prometheus.Metric("status.reason"),
		"kube_pod_container_status_last_terminated_reason": prometheus.Metric("status.last_terminated_reason"),
	},

	Labels: map[string]prometheus.LabelMap{
		"pod":          prometheus.KeyLabel(mb.ModuleDataKey + ".pod.name"),
		"namespace":    prometheus.KeyLabel(mb.ModuleDataKey + ".namespace"),
		"container":    prometheus.KeyLabel("name"),
		"node":         prometheus.Label(mb.ModuleDataKey + ".node.name"),
		"container_id": prometheus.KeyLabel("id"),
		"resource":     prometheus.Label("resource"),
		"unit":         prometheus.Label("unit"),
		"reason":       prometheus.Label("reason"),
	},
}

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "state_container", New, mb.WithHostParser(prometheus.HostParser))
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	prometheus prometheus.Prometheus
	enricher   util.Enricher
	mod        k8smod.Module
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := prometheus.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}
	mod, ok := base.Module().(k8smod.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of kubernetes module")
	}
	return &MetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
		enricher:      util.NewContainerMetadataEnricher(base, mod.GetMetricsRepo(), false),
		mod:           mod,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	m.enricher.Start()

	families, err := m.mod.GetStateMetricsFamilies(m.prometheus)
	if err != nil {
		return fmt.Errorf("error getting families: %w", err)
	}
	events, err := m.prometheus.ProcessMetrics(families, mapping)
	if err != nil {
		return fmt.Errorf("error getting event: %w", err)
	}

	m.enricher.Enrich(events)

	for _, event := range events {
		// applying ECS to kubernetes.container.id in the form <container.runtime>://<container.id>
		// copy to ECS fields the kubernetes.container.image, kubernetes.container.name
		containerFields := mapstr.M{}
		if containerID, ok := event["id"]; ok {
			// we don't expect errors here, but if any we would obtain an
			// empty string
			cID, ok := (containerID).(string)
			if !ok {
				m.Logger().Debugf("Error while casting containerID: %s", ok)
			}
			split := strings.Index(cID, "://")
			if split != -1 {
				kubernetes.ShouldPut(containerFields, "runtime", cID[:split], m.Logger())

				kubernetes.ShouldPut(containerFields, "id", cID[split+3:], m.Logger())
			}
		}
		if containerImage, ok := event["image"]; ok {
			cImage, ok := (containerImage).(string)
			if !ok {
				m.Logger().Debugf("Error while casting containerImage: %s", ok)
			}

			kubernetes.ShouldPut(containerFields, "image.name", cImage, m.Logger())
			// remove kubernetes.container.image field as value is the same as ECS container.image.name field
			kubernetes.ShouldDelete(event, "image", m.Logger())
		}

		e, err := util.CreateEvent(event, "kubernetes.container")
		if err != nil {
			m.Logger().Error(err)
		}

		if len(containerFields) > 0 {
			if e.RootFields != nil {
				e.RootFields.DeepUpdate(mapstr.M{
					"container": containerFields,
				})
			} else {
				e.RootFields = mapstr.M{
					"container": containerFields,
				}
			}
		}

		if reported := reporter.Event(e); !reported {
			return nil
		}
	}

	return nil
}

// Close stops this metricset
func (m *MetricSet) Close() error {
	m.enricher.Stop()
	return nil
}
