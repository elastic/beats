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

package kubernetes

import (
	"fmt"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	k8smod "github.com/elastic/beats/v7/metricbeat/module/kubernetes"
)

const prefix = "state_"

/*
mappings stores the metrics for each metricset. The key of the map is the name of the metricset
and the values are the mapping of the metricset metrics.
E.g: mappings[state_cronjob] = &{map[kube_cronjob_created: prometheus.Metric}
*/
var mappings = map[string]*prometheus.MetricsMapping{}

// Lock to control concurrent read/writes
var lock sync.RWMutex

// Init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func Init(name string, mapping *prometheus.MetricsMapping) {
	name = prefix + name
	lock.Lock()
	mappings[name] = mapping
	lock.Unlock()
	mb.Registry.MustAddMetricSet("kubernetes", name, New, mb.WithHostParser(prometheus.HostParser))
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	prometheusClient  prometheus.Prometheus
	prometheusMapping *prometheus.MetricsMapping
	mod               k8smod.Module
	enricher          util.Enricher
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheusClient, err := prometheus.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}
	mod, ok := base.Module().(k8smod.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of kubernetes module")
	}

	lock.Lock()
	mapping := mappings[base.Name()]
	lock.Unlock()

	return &MetricSet{
		BaseMetricSet:     base,
		prometheusClient:  prometheusClient,
		prometheusMapping: mapping,
		enricher:          util.NewResourceMetadataEnricher(base, strings.ReplaceAll(base.Name(), prefix, ""), mod.GetMetricsRepo(), false),
		mod:               mod,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	m.enricher.Start()

	families, err := m.mod.GetStateMetricsFamilies(m.prometheusClient)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}
	events, err := m.prometheusClient.ProcessMetrics(families, m.prometheusMapping)
	if err != nil {
		m.Logger().Error(err)
		reporter.Error(err)
		return
	}

	m.enricher.Enrich(events)
	for _, event := range events {
		// The name of the metric state can be obtained by using m.BaseMetricSet.Name(). However, names that start with state_* (e.g. state_cronjob)
		// need to have that prefix removed. So, for example, strings.ReplaceAll("state_cronjob", "state_", "") would result in just cronjob.
		e, err := util.CreateEvent(event, "kubernetes."+strings.ReplaceAll(m.BaseMetricSet.Name(), "state_", ""))
		if err != nil {
			m.Logger().Error(err)
		}

		if reported := reporter.Event(e); !reported {
			m.Logger().Debug("error trying to emit event")
			return
		}
	}
}

// Close stops this metricset
func (m *MetricSet) Close() error {
	m.enricher.Stop()
	return nil
}
