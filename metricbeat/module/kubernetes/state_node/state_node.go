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
	"github.com/elastic/beats/v7/metricbeat/helper/kubernetes"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

// mapping stores the state metrics we want to fetch and will be used by this metricset
var mapping = &p.MetricsMapping{
	Metrics: map[string]p.MetricMap{
		"kube_node_info": p.InfoMetric(),

		"kube_node_status_capacity": p.Metric("", p.OpFilterMap(
			"resource", map[string]string{
				"pods":   "pod.capacity.total",
				"cpu":    "cpu.capacity.cores",
				"memory": "memory.capacity.bytes",
			},
		)),
		"kube_node_status_allocatable": p.Metric("", p.OpFilterMap(
			"resource", map[string]string{
				"pods":   "pod.allocatable.total",
				"cpu":    "cpu.allocatable.cores",
				"memory": "memory.allocatable.bytes",
			},
		)),
		"kube_node_spec_unschedulable": p.BooleanMetric("status.unschedulable"),

		"kube_node_status_condition": p.LabelMetric("status", "status", p.OpFilterMap(
			"condition", map[string]string{
				"Ready":              "ready",
				"MemoryPressure":     "memory_pressure",
				"DiskPressure":       "disk_pressure",
				"OutOfDisk":          "out_of_disk",
				"PIDPressure":        "pid_pressure",
				"NetworkUnavailable": "network_unavailable",
			},
		)),
	},
	Labels: map[string]p.LabelMap{
		"node": p.KeyLabel("name"),

		// from info metric "kube_node_info"
		"kubelet_version": p.Label("kubelet.version"),
	},
}

// Register metricset
func init() {
	kubernetes.Init(util.NodeResource, mapping)
}
