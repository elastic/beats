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

package master

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"master": s.Object{
			"cpus_percent":                           c.Int("master/cpus_percent"),
			"cpus_used":                              c.Int("master/cpus_used"),
			"cpus_total":                             c.Int("master/cpus_total"),
			"cpus_revocable_percent":                 c.Int("master/cpus_revocable_percent"),
			"cpus_revocable_total":                   c.Int("master/cpus_revocable_total"),
			"cpus_revocable_used":                    c.Int("master/cpus_revocable_used"),
			"disk_percent":                           c.Int("master/disk_percent"),
			"disk_used":                              c.Int("master/disk_used"),
			"disk_total":                             c.Int("master/disk_total"),
			"disk_revocable_percent":                 c.Int("master/disk_revocable_percent"),
			"disk_revocable_total":                   c.Int("master/disk_revocable_total"),
			"disk_revocable_used":                    c.Int("master/disk_revocable_used"),
			"gpus_percent":                           c.Int("master/gpus_percent"),
			"gpus_used":                              c.Int("master/gpus_used"),
			"gpus_total":                             c.Int("master/gpus_total"),
			"gpus_revocable_percent":                 c.Int("master/gpus_revocable_percent"),
			"gpus_revocable_total":                   c.Int("master/gpus_revocable_total"),
			"gpus_revocable_used":                    c.Int("master/gpus_revocable_used"),
			"mem_percent":                            c.Int("master/mem_percent"),
			"mem_used":                               c.Int("master/mem_used"),
			"mem_total":                              c.Int("master/mem_total"),
			"mem_revocable_percent":                  c.Int("master/mem_revocable_percent"),
			"mem_revocable_total":                    c.Int("master/mem_revocable_total"),
			"mem_revocable_used":                     c.Int("master/mem_revocable_used"),
			"elected":                                c.Int("master/elected"),
			"uptime_secs":                            c.Float("master/uptime_secs"),
			"slave_registrations":                    c.Int("master/slave_registrations"),
			"slave_reregistrations":                  c.Int("master/slave_reregistrations"),
			"slave_unreachable_scheduled":            c.Int("master/slave_unreachable_scheduled"),
			"slave_unreachable_canceled":             c.Int("master/slave_unreachable_canceled"),
			"slave_unreachable_completed":            c.Int("master/slave_unreachable_completed"),
			"slaves_active":                          c.Int("master/slaves_active"),
			"slaves_connected":                       c.Int("master/slaves_connected"),
			"slaves_disconnected":                    c.Int("master/slaves_disconnected"),
			"slaves_inactive":                        c.Int("master/slaves_inactive"),
			"slaves_unreachable":                     c.Int("master/slaves_unreachable"),
			"frameworks_active":                      c.Int("master/frameworks_active"),
			"frameworks_connected":                   c.Int("master/frameworks_connected"),
			"frameworks_disconnected":                c.Int("master/frameworks_disconnected"),
			"frameworks_inactive":                    c.Int("master/frameworks_inactive"),
			"outstanding_offers":                     c.Int("master/outstanding_offers"),
			"tasks_error":                            c.Int("master/tasks_error"),
			"tasks_failed":                           c.Int("master/tasks_failed"),
			"tasks_finished":                         c.Int("master/tasks_finished"),
			"tasks_killed":                           c.Int("master/tasks_killed"),
			"tasks_killing":                          c.Int("master/tasks_killing"),
			"tasks_lost":                             c.Int("master/tasks_lost"),
			"tasks_running":                          c.Int("master/tasks_running"),
			"tasks_staging":                          c.Int("master/tasks_staging"),
			"tasks_starting":                         c.Int("master/tasks_starting"),
			"tasks_unreachable":                      c.Int("master/tasks_unreachable"),
			"invalid_executor_to_framework_messages": c.Int("master/invalid_executor_to_framework_messages"),
			"invalid_framework_to_executor_messages": c.Int("master/invalid_framework_to_executor_messages"),
			"invalid_operation_status_update_acknowledgements": c.Int("master/invalid_operation_status_update_acknowledgements"),
			"invalid_status_update_acknowledgements":           c.Int("master/invalid_status_update_acknowledgements"),
			"invalid_status_updates":                           c.Int("master/invalid_status_updates"),
			"dropped_messages":                                 c.Int("master/dropped_messages"),
			"messages_authenticate":                            c.Int("master/messages_authenticate"),
			"messages_deactivate_framework":                    c.Int("master/messages_deactivate_framework"),
			"messages_decline_offers":                          c.Int("master/messages_decline_offers"),
			"messages_executor_to_framework":                   c.Int("master/messages_executor_to_framework"),
			"messages_exited_executor":                         c.Int("master/messages_exited_executor"),
			"messages_framework_to_executor":                   c.Int("master/messages_framework_to_executor"),
			"messages_kill_task":                               c.Int("master/messages_kill_task"),
			"messages_launch_tasks":                            c.Int("master/messages_launch_tasks"),
			"messages_operation_status_update_acknowledgement": c.Int("master/messages_operation_status_update_acknowledgement"),
			"messages_reconcile_operations":                    c.Int("master/messages_reconcile_operations"),
			"messages_reconcile_tasks":                         c.Int("master/messages_reconcile_tasks"),
			"messages_register_framework":                      c.Int("master/messages_register_framework"),
			"messages_register_slave":                          c.Int("master/messages_register_slave"),
			"messages_reregister_framework":                    c.Int("master/messages_reregister_framework"),
			"messages_reregister_slave":                        c.Int("master/messages_reregister_slave"),
			"messages_resource_request":                        c.Int("master/messages_resource_request"),
			"messages_revive_offers":                           c.Int("master/messages_revive_offers"),
			"messages_status_update":                           c.Int("master/messages_status_update"),
			"messages_status_update_acknowledgement":           c.Int("master/messages_status_update_acknowledgement"),
			"messages_unregister_framework":                    c.Int("master/messages_unregister_framework"),
			"messages_unregister_slave":                        c.Int("master/messages_unregister_slave"),
			"messages_update_slave":                            c.Int("master/messages_update_slave"),
			"recovery_slave_removals":                          c.Int("master/recovery_slave_removals"),
			"slave_removals":                                   c.Int("master/slave_removals"),
			"slave_Removals": s.Object{
				"reason_registered":   c.Int("master/slave_removals/reason_registered"),
				"reason_unhealthy":    c.Int("master/slave_removals/reason_unhealthy"),
				"reason_unregistered": c.Int("master/slave_removals/reason_unregistered"),
			},
			"valid_framework_to_executor_messages":           c.Int("master/valid_framework_to_executor_messages"),
			"valid_operation_status_update_acknowledgements": c.Int("master/valid_operation_status_update_acknowledgements"),
			"valid_status_update_acknowledgements":           c.Int("master/valid_status_update_acknowledgements"),
			"valid_status_updates":                           c.Int("master/valid_status_updates"),
			"valid_executor_to_framework_messages":           c.Int("master/valid_executor_to_framework_messages"),
			"event_queue_dispatches":                         c.Int("master/event_queue_dispatches"),
			"event_queue_http_requests":                      c.Int("master/event_queue_http_requests"),
			"event_queue_messages":                           c.Int("master/event_queue_messages"),
			"operator_event_stream_subscribers":              c.Int("master/operator_event_stream_subscribers", s.Optional),
		},
		"system": s.Object{
			"cpus_total":      c.Int("system/cpus_total"),
			"load_15min":      c.Float("system/load_15min"),
			"load_5min":       c.Float("system/load_5min"),
			"load_1min":       c.Float("system/load_1min"),
			"mem_free_bytes":  c.Int("system/mem_free_bytes"),
			"mem_total_bytes": c.Int("system/mem_total_bytes"),
		},
		"registrar": s.Object{
			"state_fetch_ms": c.Float("registrar/state_fetch_ms"),
			"state_store_ms": c.Float("registrar/state_store_ms"),
			"state_store_MS": s.Object{
				"max":   c.Float("registrar/state_store_ms/max", s.Optional),
				"min":   c.Float("registrar/state_store_ms/min", s.Optional),
				"p50":   c.Float("registrar/state_store_ms/p50", s.Optional),
				"p90":   c.Float("registrar/state_store_ms/p90", s.Optional),
				"p95":   c.Float("registrar/state_store_ms/p95", s.Optional),
				"p99":   c.Float("registrar/state_store_ms/p99", s.Optional),
				"p999":  c.Float("registrar/state_store_ms/p999", s.Optional),
				"p9999": c.Float("registrar/state_store_ms/p9999", s.Optional),
				"count": c.Int("registrar/state_store_ms/count", s.Optional),
			},
			"log": s.Object{
				"recovered":     c.Int("registrar/log/recovered", s.Optional),
				"ensemble_size": c.Int("registrar/log/ensemble_size", s.Optional),
			},
		},
		"allocator": s.Object{
			"mesos": s.Object{
				"allocation_run_ms": c.Float("allocator/mesos/allocation_run_ms"),
				"allocation_run_MS": s.Object{
					"count": c.Int("allocator/mesos/allocation_run_ms/count", s.Optional),
					"max":   c.Float("allocator/mesos/allocation_run_ms/max", s.Optional),
					"min":   c.Float("allocator/mesos/allocation_run_ms/min", s.Optional),
					"p50":   c.Float("allocator/mesos/allocation_run_ms/p50", s.Optional),
					"p90":   c.Float("allocator/mesos/allocation_run_ms/p90", s.Optional),
					"p95":   c.Float("allocator/mesos/allocation_run_ms/p95", s.Optional),
					"p99":   c.Float("allocator/mesos/allocation_run_ms/p99", s.Optional),
					"p999":  c.Float("allocator/mesos/allocation_run_ms/p999", s.Optional),
					"p9999": c.Float("allocator/mesos/allocation_run_ms/p9999", s.Optional),
				},
				"allocation_runs":           c.Int("allocator/mesos/allocation_runs"),
				"allocation_run_latency_ms": c.Float("allocator/mesos/allocation_run_latency_ms"),
				"allocation_run_latency_MS": s.Object{
					"count": c.Int("allocator/mesos/allocation_run_latency_ms/count", s.Optional),
					"max":   c.Float("allocator/mesos/allocation_run_latency_ms/max", s.Optional),
					"min":   c.Float("allocator/mesos/allocation_run_latency_ms/min", s.Optional),
					"p50":   c.Float("allocator/mesos/allocation_run_latency_ms/p50", s.Optional),
					"p90":   c.Float("allocator/mesos/allocation_run_latency_ms/p90", s.Optional),
					"p95":   c.Float("allocator/mesos/allocation_run_latency_ms/p95", s.Optional),
					"p99":   c.Float("allocator/mesos/allocation_run_latency_ms/p99", s.Optional),
					"p999":  c.Float("allocator/mesos/allocation_run_latency_ms/p999", s.Optional),
					"p9999": c.Float("allocator/mesos/allocation_run_latency_ms/p9999", s.Optional),
				},
				"event_queue_dispatches": c.Int("allocator/mesos/event_queue_dispatches"),
				"resources": s.Object{
					"cpus": s.Object{
						"offered_or_allocated": c.Int("allocator/mesos/resources/cpus/offered_or_allocated"),
						"total":                c.Int("allocator/mesos/resources/cpus/total"),
					},
					"disk": s.Object{
						"offered_or_allocated": c.Int("allocator/mesos/resources/disk/offered_or_allocated"),
						"total":                c.Int("allocator/mesos/resources/disk/total"),
					},
					"mem": s.Object{
						"offered_or_allocated": c.Int("allocator/mesos/resources/mem/offered_or_allocated"),
						"total":                c.Int("allocator/mesos/resources/mem/total"),
					},
				},
			},
		},
	}
)

func eventMapping(content []byte, reporter mb.ReporterV2) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Mesos stats API response")
	}

	event, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure applying schema")
	}

	reporter.Event(mb.Event{MetricSetFields: event})
	return nil
}
