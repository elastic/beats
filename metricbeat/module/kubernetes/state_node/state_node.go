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

package state_node

import (
	"fmt"

	"github.com/elastic/beats/v8/libbeat/common/kubernetes"
	p "github.com/elastic/beats/v8/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
	k8smod "github.com/elastic/beats/v8/metricbeat/module/kubernetes"
	"github.com/elastic/beats/v8/metricbeat/module/kubernetes/util"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()

	mapping = &p.MetricsMapping{
		Metrics: map[string]p.MetricMap{
			"kube_node_info":                            p.InfoMetric(),
			"kube_node_status_allocatable_pods":         p.Metric("pod.allocatable.total"),
			"kube_node_status_capacity_pods":            p.Metric("pod.capacity.total"),
			"kube_node_status_capacity_memory_bytes":    p.Metric("memory.capacity.bytes"),
			"kube_node_status_allocatable_memory_bytes": p.Metric("memory.allocatable.bytes"),
			"kube_node_status_capacity_cpu_cores":       p.Metric("cpu.capacity.cores"),
			"kube_node_status_allocatable_cpu_cores":    p.Metric("cpu.allocatable.cores"),
			"kube_node_status_capacity": p.Metric("", p.OpFilterMap(
				"resource", map[string]string{
					"pods":   "pod.capacity.total",
					"cpu":    "cpu.capacity.cores",
					"memory": "memory.capacity.bytes",
				},
			)),
			"kube_node_status_allocatable": p.Metric("", p.OpFilterMap(
				"resource", map[string]string{
					"pods":   "pod.allocatable.total",
					"cpu":    "cpu.allocatable.cores",
					"memory": "memory.allocatable.bytes",
				},
			)),
			"kube_node_spec_unschedulable": p.BooleanMetric("status.unschedulable"),
			"kube_node_status_ready":       p.LabelMetric("status.ready", "condition"),
			"kube_node_status_condition": p.LabelMetric("status", "status", p.OpFilterMap(
				"condition", map[string]string{
					"Ready":          "ready",
					"MemoryPressure": "memory_pressure",
					"DiskPressure":   "disk_pressure",
					"OutOfDisk":      "out_of_disk",
					"PIDPressure":    "pid_pressure",
				},
			)),
		},
		Labels: map[string]p.LabelMap{
			"node": p.KeyLabel("name"),
		},
	}
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "state_node", New,
		mb.WithHostParser(hostParser),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
	enricher   util.Enricher
	mod        k8smod.Module
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := p.NewPrometheusClient(base)
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
		enricher:      util.NewResourceMetadataEnricher(base, &kubernetes.Node{}, false),
		mod:           mod,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	m.enricher.Start()

	families, err := m.mod.GetStateMetricsFamilies(m.prometheus)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}
	events, err := m.prometheus.ProcessMetrics(families, mapping)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}

	m.enricher.Enrich(events)
	for _, event := range events {

		e, err := util.CreateEvent(event, "kubernetes.node")
		if err != nil {
			m.Logger().Error(err)
		}

		if reported := reporter.Event(e); !reported {
			m.Logger().Debug("error trying to emit event")
			return
		}
	}

	return
}

// Close stops this metricset
func (m *MetricSet) Close() error {
	m.enricher.Stop()
	return nil
}
