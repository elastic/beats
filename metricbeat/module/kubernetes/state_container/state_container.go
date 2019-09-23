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
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
	// Nanocores conversion 10^9
	nanocores = 1000000000
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()

	// Mapping of state metrics
	mapping = &p.MetricsMapping{
		Metrics: map[string]p.MetricMap{
			"kube_pod_info":                                     p.InfoMetric(),
			"kube_pod_container_info":                           p.InfoMetric(),
			"kube_pod_container_resource_limits_cpu_cores":      p.Metric("cpu.limit.cores"),
			"kube_pod_container_resource_requests_cpu_cores":    p.Metric("cpu.request.cores"),
			"kube_pod_container_resource_limits_memory_bytes":   p.Metric("memory.limit.bytes"),
			"kube_pod_container_resource_requests_memory_bytes": p.Metric("memory.request.bytes"),
			"kube_pod_container_status_ready":                   p.BooleanMetric("status.ready"),
			"kube_pod_container_status_restarts":                p.Metric("status.restarts"),
			"kube_pod_container_status_restarts_total":          p.Metric("status.restarts"),
			"kube_pod_container_status_running":                 p.KeywordMetric("status.phase", "running"),
			"kube_pod_container_status_terminated":              p.KeywordMetric("status.phase", "terminated"),
			"kube_pod_container_status_waiting":                 p.KeywordMetric("status.phase", "waiting"),
			"kube_pod_container_status_terminated_reason":       p.LabelMetric("status.reason", "reason"),
			"kube_pod_container_status_waiting_reason":          p.LabelMetric("status.reason", "reason"),
		},

		Labels: map[string]p.LabelMap{
			"pod":       p.KeyLabel(mb.ModuleDataKey + ".pod.name"),
			"container": p.KeyLabel("name"),
			"namespace": p.KeyLabel(mb.ModuleDataKey + ".namespace"),

			"node":         p.Label(mb.ModuleDataKey + ".node.name"),
			"container_id": p.Label("id"),
			"image":        p.Label("image"),
		},
	}
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("kubernetes", "state_container", New, hostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
	enricher   util.Enricher
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := p.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
		enricher:      util.NewContainerMetadataEnricher(base, false),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	m.enricher.Start()

	events, err := m.prometheus.GetProcessedMetrics(mapping)
	if err != nil {
		return errors.Wrap(err, "error getting event")
	}

	m.enricher.Enrich(events)

	// Calculate deprecated nanocores values
	for _, event := range events {
		if request, ok := event["cpu.request.cores"]; ok {
			if requestCores, ok := request.(float64); ok {
				event["cpu.request.nanocores"] = requestCores * nanocores
			}
		}

		if limit, ok := event["cpu.limit.cores"]; ok {
			if limitCores, ok := limit.(float64); ok {
				event["cpu.limit.nanocores"] = limitCores * nanocores
			}
		}

		var moduleFieldsMapStr common.MapStr
		moduleFields, ok := event[mb.ModuleDataKey]
		if ok {
			moduleFieldsMapStr, ok = moduleFields.(common.MapStr)
			if !ok {
				m.Logger().Errorf("error trying to convert '%s' from event to common.MapStr", mb.ModuleDataKey)
			}
		}
		delete(event, mb.ModuleDataKey)

		if reported := reporter.Event(mb.Event{
			MetricSetFields: event,
			ModuleFields:    moduleFieldsMapStr,
			Namespace:       "kubernetes.container",
		}); !reported {
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
