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

package state_replicaset

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"
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
			"kube_replicaset_metadata_generation":           p.InfoMetric(),
			"kube_replicaset_status_fully_labeled_replicas": p.Metric("replicas.labeled"),
			"kube_replicaset_status_observed_generation":    p.Metric("replicas.observed"),
			"kube_replicaset_status_ready_replicas":         p.Metric("replicas.ready"),
			"kube_replicaset_spec_replicas":                 p.Metric("replicas.desired"),
			"kube_replicaset_status_replicas":               p.Metric("replicas.available"),
		},

		Labels: map[string]p.LabelMap{
			"replicaset": p.KeyLabel("name"),
			"namespace":  p.KeyLabel(mb.ModuleDataKey + ".namespace"),
		},
	}
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("kubernetes", "state_replicaset", New, hostParser); err != nil {
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
		enricher:      util.NewResourceMetadataEnricher(base, &kubernetes.ReplicaSet{}, false),
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	m.enricher.Start()

	events, err := m.prometheus.GetProcessedMetrics(mapping)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}

	m.enricher.Enrich(events)
	for _, event := range events {

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
			Namespace:       "kubernetes.replicaset",
		}); !reported {
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
