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

package state_deployment

import (
	"github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

// mapping stores the state metrics we want to fetch and will be used by this metricset
var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_deployment_metadata_generation":         p.InfoMetric(),
		"kube_deployment_status_replicas_updated":     p.Metric("replicas.updated"),
		"kube_deployment_status_replicas_unavailable": p.Metric("replicas.unavailable"),
		"kube_deployment_status_replicas_available":   p.Metric("replicas.available"),
		"kube_deployment_spec_replicas":               p.Metric("replicas.desired"),
		/*
			This is how deployment_status_condition field will be exported:

			kube_deployment_status_condition{namespace="default",deployment="test-deployment",condition="Available",status="true"} 0
			kube_deployment_status_condition{namespace="default",deployment="test-deployment",condition="Available",status="false"} 1
			kube_deployment_status_condition{namespace="default",deployment="test-deployment",condition="Available",status="unknown"} 0
			kube_deployment_status_condition{namespace="default",deployment="test-deployment",condition="Progressing",status="true"} 1
			kube_deployment_status_condition{namespace="default",deployment="test-deployment",condition="Progressing",status="false"} 0
			kube_deployment_status_condition{namespace="default",deployment="test-deployment",condition="Progressing",status="unknown"} 0
		*/
		"kube_deployment_status_condition": p.LabelMetric("status", "status", p.OpFilterMap(
			"condition", map[string]string{
				"Progressing": "progressing",
				"Available":   "available",
			},
		)), //The current status conditions of a deployment
		"kube_deployment_spec_paused": p.BooleanMetric("paused"),
	},

	Labels: map[string]p.LabelMap{
		"deployment": p.KeyLabel("name"),
		"namespace":  p.KeyLabel(mb.ModuleDataKey + ".namespace"),
	},
}

// Register metricset
func init() {
	kubernetes.Init(util.DeploymentResource, mapping)
}
