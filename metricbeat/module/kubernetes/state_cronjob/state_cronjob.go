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

package state_cronjob

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "state_cronjob",
		NewCronJobMetricSet,
		mb.WithHostParser(p.HostParser))
}

// CronJobMetricSet uses a prometheus based MetricSet that looks for
// mb.ModuleDataKey prefixed fields and puts then at the module level
//
// Copying the code from other kube state metrics, this should be improved to
// avoid all these ugly tricks
type CronJobMetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
	mapping    *p.MetricsMapping
}

// NewCronJobMetricSet returns a prometheus based metricset for CronJobs
func NewCronJobMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := p.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}

	return &CronJobMetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
		mapping: &p.MetricsMapping{
			Metrics: map[string]p.MetricMap{
				"kube_cronjob_info":                           p.InfoMetric(),
				"kube_cronjob_created":                        p.Metric("created.sec"),
				"kube_cronjob_status_active":                  p.Metric("active.count"),
				"kube_cronjob_status_last_schedule_time":      p.Metric("last_schedule.sec"),
				"kube_cronjob_next_schedule_time":             p.Metric("next_schedule.sec"),
				"kube_cronjob_spec_suspend":                   p.BooleanMetric("is_suspended"),
				"kube_cronjob_spec_starting_deadline_seconds": p.Metric("deadline.sec"),
			},
			Labels: map[string]p.LabelMap{
				"cronjob":            p.KeyLabel("name"),
				"namespace":          p.KeyLabel(mb.ModuleDataKey + ".namespace"),
				"schedule":           p.KeyLabel("schedule"),
				"concurrency_policy": p.KeyLabel("concurrency"),
			},
		},
	}, nil
}

// Fetch prometheus metrics and treats those prefixed by mb.ModuleDataKey as
// module rooted fields at the event that gets reported
//
// Copied from other kube state metrics.
func (m *CronJobMetricSet) Fetch(reporter mb.ReporterV2) error {
	events, err := m.prometheus.GetProcessedMetrics(m.mapping)
	if err != nil {
		return errors.Wrap(err, "error getting metrics")
	}

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
			Namespace:       "kubernetes.cronjob",
		}); !reported {
			return nil
		}
	}

	return nil
}
