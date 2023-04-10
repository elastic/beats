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
	"github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

// mapping stores the state metrics we want to fetch and will be used by this metricset
var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		// Make everything in "kube_job_owner" available for use in the Labels section, below.
		"kube_job_owner": p.InfoMetric(),
		// These fields are mapped 1:1 from their KSM metrics.
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

// Register metricset
func init() {
	kubernetes.Init(util.JobResource, mapping)
}
