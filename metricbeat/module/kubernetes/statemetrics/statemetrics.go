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

package statemetrics

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/pkg/errors"
)

func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "statemetrics",
		new,
		mb.WithHostParser(prometheus.HostParser))
}

// stateMetricsMetricSet TODO
type stateMetricsMetricSet struct {
	mb.BaseMetricSet
	promClient prometheus.Prometheus
	mapping    *prometheus.MetricsMapping
}

type blackListFilter struct {
}

type whiteListFilter struct {
}

// Config for StateMetrics metricset
type Config struct {
	BlackList blackListFilter `config:"blacklist_filter"`
	WhiteList whiteListFilter `config:"whitelist_filter"`
}

// getGroupMappingsFn function instances can be found at this package to feed the
// metric and label mappings that have not been filtered through configuration
type getGroupMappingsFn func() (mm map[string]prometheus.MetricMap, lm map[string]prometheus.LabelMap)

// new returns a mb.MetricSet object that can fetch and report
// kube-state-metrics metrics
func new(base mb.BaseMetricSet) (mb.MetricSet, error) {

	// TODO config filters

	// ksmGroups maps configuration item strings that identify each
	// kubernetes state metrics group with the function that returns
	// metrics and label mappings for that group
	ksmGroups := map[string]getGroupMappingsFn{
		"certificatesigningrequest": getCertificateSigningRequestMapping,
	}

	metricsMap := map[string]prometheus.MetricMap{}
	labelsMap := map[string]prometheus.LabelMap{}
	for g, fn := range ksmGroups {
		logp.Debug("kubernetes", "Adding mappings for kube-state-metrics group %q", g)
		metrics, labels := fn()
		for k, v := range metrics {
			metricsMap[k] = v
		}
		for k, v := range labels {
			labelsMap[k] = v
		}
	}

	promClient, err := prometheus.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}

	return &stateMetricsMetricSet{
		BaseMetricSet: base,
		promClient:    promClient,
		mapping: &prometheus.MetricsMapping{
			Metrics: metricsMap,
			Labels:  labelsMap,
		}}, nil
}

func (s *stateMetricsMetricSet) Fetch(reporter mb.ReporterV2) error {
	events, err := s.promClient.GetProcessedMetrics(s.mapping)
	if err != nil {
		return errors.Wrap(err, "error getting metrics")
	}

	for _, event := range events {
		var moduleFieldsMapStr common.MapStr
		moduleFields, ok := event[mb.ModuleDataKey]
		if ok {
			moduleFieldsMapStr, ok = moduleFields.(common.MapStr)
			if !ok {
				s.Logger().Errorf("error trying to convert '%s' from event to common.MapStr", mb.ModuleDataKey)
			}
			delete(event, mb.ModuleDataKey)
		}

		// moving labels from kubernetes.statemetrics.labels to kubernetes.labels
		// should make aggregation by labels easier for all kubernetes objects,
		// not only the ones found at kube state metrics
		labels, err := event.GetValue("labels")
		if err == nil {
			if moduleFieldsMapStr == nil {
				moduleFieldsMapStr = common.MapStr{}
			}
			_, err = moduleFieldsMapStr.Put("labels", labels)
			if err != nil {
				s.Logger().Errorf("error moving labels from kubernetes.statemetrics.labels to kubernetes.labels: %s", err.Error())
			} else {
				err = event.Delete("labels")
				if err != nil {
					s.Logger().Errorf("error deleting labels from kubernetes.statemetrics.labels: %s", err.Error())
				}
			}
		}

		if reported := reporter.Event(mb.Event{
			MetricSetFields: event,
			ModuleFields:    moduleFieldsMapStr,
			Namespace:       "kubernetes.statemetrics",
		}); !reported {
			return nil
		}
	}
	return nil
}
