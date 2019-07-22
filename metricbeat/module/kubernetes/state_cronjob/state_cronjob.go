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
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
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
			// "kube_cronjob_labels":          p.NeedTypeForThis("cronjob.labels", "label_run"),
			"kube_cronjob_info":                           p.InfoMetric(),
			"kube_cronjob_created":                        p.Metric("created.sec"),
			"kube_cronjob_status_active":                  p.Metric("active.count"),
			"kube_cronjob_status_last_schedule_time":      p.Metric("lastschedule.sec"),
			"kube_cronjob_next_schedule_time":             p.Metric("nextschedule.sec"),
			"kube_cronjob_spec_suspend":                   p.BooleanMetric("is_suspended"),
			"kube_cronjob_spec_starting_deadline_seconds": p.Metric("deadline.sec"),
		},

		Labels: map[string]p.LabelMap{
			"cronjob":            p.KeyLabel("name"),
			"namespace":          p.KeyLabel("namespace"),
			"schedule":           p.KeyLabel("schedule"),
			"concurrency_policy": p.KeyLabel("concurrency"),
		},
	}
)

func init() {

	mb.Registry.MustAddMetricSet("kubernetes", "state_cronjob",
		p.MetricSetBuilder(mapping),
		mb.WithHostParser(p.HostParser))

}
