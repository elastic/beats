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
	sm "github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_job_owner": p.InfoMetric(),

		"kube_job_status_active":    p.Metric("pods.active"),
		"kube_job_status_failed":    p.Metric("pods.failed"),
		"kube_job_status_succeeded": p.Metric("pods.succeeded"),
		"kube_job_spec_completions": p.Metric("completions.desired"),

		"kube_job_spec_parallelism":       p.Metric("parallelism.desired"),
		"kube_job_created":                p.Metric("time.created", p.OpUnixTimestampValue()),
		"kube_job_status_completion_time": p.Metric("time.completed", p.OpUnixTimestampValue()),

		"kube_job_complete": p.BooleanMetric("status.complete"),
		"kube_job_failed":   p.BooleanMetric("status.failed"),
	},

	Labels: map[string]p.LabelMap{
		"job_name":            p.KeyLabel("name"),
		"namespace":           p.KeyLabel(mb.ModuleDataKey + ".namespace"),
		"owner_kind":          p.KeyLabel("owner.kind"),
		"owner_name":          p.KeyLabel("owner.name"),
		"owner_is_controller": p.KeyLabel("owner.is_controller"),
		"condition":           p.KeyLabel("condition"),
	},
}

// init registers the MetricSet with the central registry.
func init() {
	sm.Init(util.JobResource, mapping)
}
