// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownershiprometheus. Elasticsearch B.V. licenses this file to you under
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
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func getPodMapping() (mm map[string]prometheus.MetricMap, lm map[string]prometheus.LabelMap) {
	return map[string]prometheus.MetricMap{
			"kube_pod_info":                                    prometheus.InfoMetric(),
			"kube_pod_start_time":                              prometheus.Metric("pod.started", prometheus.OpUnixTimestampValue()),
			"kube_pod_completion_time":                         prometheus.Metric("pod.completed", prometheus.OpUnixTimestampValue()),
			"kube_pod_owner":                                   prometheus.Metric("pod.owner"),
			"kube_pod_labels":                                  prometheus.ExtendedInfoMetric(prometheus.Configuration{StoreNonMappedLabels: true, NonMappedLabelsPlacement: "labels"}),
			"kube_pod_status_phase":                            prometheus.LabelMetric("pod.status.phase", "phase"),
			"kube_pod_status_ready":                            prometheus.LabelMetric("pod.status.ready", "condition"),
			"kube_pod_status_scheduled":                        prometheus.LabelMetric("pod.status.scheduled", "condition"),
			"kube_pod_container_info":                          prometheus.InfoMetric(),
			"kube_pod_container_status_waiting":                prometheus.KeywordMetric("pod.container.status.state", "waiting"),
			"kube_pod_container_status_waiting_reason":         prometheus.LabelMetric("pod.container.status.reason", "reason"),
			"kube_pod_container_status_running":                prometheus.KeywordMetric("pod.container.status.state", "running"),
			"kube_pod_container_status_terminated":             prometheus.KeywordMetric("pod.container.status.state", "terminated"),
			"kube_pod_container_status_terminated_reason":      prometheus.LabelMetric("pod.container.status.reason", "reason"),
			"kube_pod_container_status_last_terminated_reason": prometheus.LabelMetric("pod.container.status.last_terminated_reason", "reason"),
			"kube_pod_container_status_ready":                  prometheus.KeywordMetric("pod.container.status.state", "ready"),
			"kube_pod_container_status_restarts_total":         prometheus.Metric("pod.container.status.restarts.count"),
			"kube_pod_container_resource_requests":             prometheus.Metric("pod.container.resource.requests.value"),
			"kube_pod_container_resource_limits":               prometheus.Metric("pod.container.resource.limits.value"),
			// not using this set of metrics being redundant
			// "kube_pod_container_resource_requests_cpu_cores":   prometheus.Metric("pod.container.resource.requests.cpu.cores"),
			// "kube_pod_container_resource_requests_memory_bytes": prometheus.Metric("pod.container.resource.requests.memory.bytes"),
			// "kube_pod_container_resource_limits_cpu_cores":      prometheus.Metric("pod.container.resource.limits.cpu.cores"),
			// "kube_pod_container_resource_limits_memory_bytes": prometheus.Metric("pod.container.resource.limits.memory.bytes"),
			"kube_pod_created":                                      prometheus.Metric("pod.created", prometheus.OpUnixTimestampValue()),
			"kube_pod_restart_policy":                               prometheus.LabelMetric("pod.restart.policy", "type"),
			"kube_pod_init_container_info":                          prometheus.ExtendedInfoMetric(prometheus.Configuration{ExtraFields: common.MapStr{"pod.init_container": true}}),
			"kube_pod_init_container_status_waiting":                prometheus.KeywordMetric("pod.container.status.state", "waiting"),
			"kube_pod_init_container_status_waiting_reason":         prometheus.LabelMetric("pod.container.status.reason", "reason"),
			"kube_pod_init_container_status_running":                prometheus.KeywordMetric("pod.container.status.state", "running"),
			"kube_pod_init_container_status_terminated":             prometheus.KeywordMetric("pod.container.status.state", "terminated"),
			"kube_pod_init_container_status_terminated_reason":      prometheus.LabelMetric("pod.container.status.reason", "reason"),
			"kube_pod_init_container_status_last_terminated_reason": prometheus.LabelMetric("pod.container.status.last_terminated_reason", "reason"),
			"kube_pod_init_container_status_ready":                  prometheus.KeywordMetric("pod.container.status.state", "ready"),
			"kube_pod_init_container_status_restarts_total":         prometheus.Metric("pod.container.status.restarts.count"),
			"kube_pod_init_container_resource_limits":               prometheus.Metric("pod.container.resource.limits.value"),
			"kube_pod_spec_volumes_persistentvolumeclaims_info":     prometheus.InfoMetric(),
			"kube_pod_spec_volumes_persistentvolumeclaims_readonly": prometheus.BooleanMetric("pod.volume.readonly"),
			"kube_pod_status_scheduled_time":                        prometheus.Metric("pod.scheduled", prometheus.OpUnixTimestampValue()),
			"kube_pod_status_unschedulable":                         prometheus.KeywordMetric("pod.container.status.state", "unschedulable"),
		},
		map[string]prometheus.LabelMap{
			"namespace": prometheus.KeyLabel(mb.ModuleDataKey + ".namespace"),
			"pod":       prometheus.KeyLabel("pod.name"),
			"container": prometheus.KeyLabel("pod.container.name"),
			"volume":    prometheus.KeyLabel("pod.volume.name"), // will this collide with other volume labels?

			"host_ip":               prometheus.Label("host.ip"), // TODO ECS
			"pod_ip":                prometheus.Label("pod.ip"),
			"node":                  prometheus.Label("node.name"),
			"created_by_kind":       prometheus.Label("created_by.kind"), // prefix pod?
			"created_by_name":       prometheus.Label("created_by.name"), // prefix pod?
			"uid":                   prometheus.Label("uid"),             // prefix pod?
			"priority_class":        prometheus.Label("pod.priority_class"),
			"owner_kind":            prometheus.Label("owner_kind"),          // prefix pod?
			"owner_name":            prometheus.Label("owner_name"),          // prefix pod?
			"owner_is_controller":   prometheus.Label("owner_is_controller"), // prefix pod?
			"image_id":              prometheus.Label("pod.container.image.id"),
			"image":                 prometheus.Label("pod.container.image.name"),
			"container_id":          prometheus.Label("pod.container.id"),
			"resource":              prometheus.KeyLabel("resource"),       // prefix pod.container? used by requets and limits
			"unit":                  prometheus.KeyLabel("unit"),           // prefix pod.container.  used by requets and limits
			"persistentvolumeclaim": prometheus.KeyLabel("pod.volume.pvc"), // will this collide with other volume labels?

		}
}
