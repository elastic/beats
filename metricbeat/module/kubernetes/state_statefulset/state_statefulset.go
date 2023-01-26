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

package state_statefulset

import (
	sm "github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_statefulset_created":                    p.Metric("created"),
		"kube_statefulset_metadata_generation":        p.Metric("generation.desired"),
		"kube_statefulset_status_observed_generation": p.Metric("generation.observed"),
		"kube_statefulset_replicas":                   p.Metric("replicas.desired"),
		"kube_statefulset_status_replicas":            p.Metric("replicas.observed"),
		"kube_statefulset_status_replicas_ready":      p.Metric("replicas.ready"),
	},

	Labels: map[string]p.LabelMap{
		"statefulset": p.KeyLabel("name"),
		"namespace":   p.KeyLabel(mb.ModuleDataKey + ".namespace"),
	},
}

// init registers the MetricSet with the central registry.
func init() {
	sm.Init(util.StatefulSetResource, mapping)
}
