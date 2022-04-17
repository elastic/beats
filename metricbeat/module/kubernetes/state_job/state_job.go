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

package state_job

import (
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/common/kubernetes"
	p "github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	k8smod "github.com/menderesk/beats/v7/metricbeat/module/kubernetes"
	"github.com/menderesk/beats/v7/metricbeat/module/kubernetes/util"
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
			// Make everything in "kube_job_owner" available for use in the Labels section, below.
			"kube_job_owner": p.InfoMetric(),
			// These fields are mapped are mapped 1:1 from their KSM metrics.
			"kube_job_status_active":    p.Metric("pods.active"),
			"kube_job_status_failed":    p.Metric("pods.failed"),
			"kube_job_status_succeeded": p.Metric("pods.succeeded"),
			"kube_job_spec_completions": p.Metric("completions.desired"),
			// completions observed?
			"kube_job_spec_parallelism":       p.Metric("parallelism.desired"),
			"kube_job_created":                p.Metric("time.created", p.OpUnixTimestampValue()),
			"kube_job_status_completion_time": p.Metric("time.completed", p.OpUnixTimestampValue()),
			// These fields will be set to "true", "false", or "unknown" based on input that looks
			// like this:
			//
			// kube_job_complete{namespace="default",job_name="timer-27074308",condition="true"} 1
			// kube_job_complete{namespace="default",job_name="timer-27074308",condition="false"} 0
			// kube_job_complete{namespace="default",job_name="timer-27074308",condition="unknown"} 0
			"kube_job_complete": p.LabelMetric("status.complete", "condition", p.OpLowercaseValue()),
			"kube_job_failed":   p.LabelMetric("status.failed", "condition", p.OpLowercaseValue()),
		},

		Labels: map[string]p.LabelMap{
			// Jobs are uniquely identified by the combination of name and namespace.
			"job_name":  p.KeyLabel("name"),
			"namespace": p.KeyLabel(mb.ModuleDataKey + ".namespace"),
			// Add owner information provided by the "kube_job_owner" InfoMetric.
			"owner_kind":          p.Label("owner.kind"),
			"owner_name":          p.Label("owner.name"),
			"owner_is_controller": p.Label("owner.is_controller"),
		},
	}
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "state_job", New,
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
		enricher:      util.NewResourceMetadataEnricher(base, &kubernetes.Job{}, false),
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

		e, err := util.CreateEvent(event, "kubernetes.job")
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
