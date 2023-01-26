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

package state_node

import (
	sm "github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
)

var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_node_info": p.InfoMetric(),

		"kube_node_status_capacity":    p.Metric("status.capacity"),
		"kube_node_status_allocatable": p.Metric("status.allocatable"),
		"kube_node_status_condition":   p.BooleanMetric("status.condition"),
		"kube_node_spec_unschedulable": p.BooleanMetric("status.unschedulable"),
	},
	Labels: map[string]p.LabelMap{
		"node":      p.KeyLabel("name"),
		"unit":      p.KeyLabel("unit"),
		"status":    p.KeyLabel("status"),
		"resource":  p.KeyLabel("resource"),
		"condition": p.KeyLabel("condition"),
	},
}

// init registers the MetricSet with the central registry.
func init() {
	sm.Init(util.NodeResource, mapping)
}
