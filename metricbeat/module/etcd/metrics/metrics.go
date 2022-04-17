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

package metrics

import (
	"github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			// Server
			"etcd_server_has_leader":                prometheus.Metric("server.has_leader"),
			"etcd_server_leader_changes_seen_total": prometheus.Metric("server.leader_changes.count"),
			"etcd_server_proposals_committed_total": prometheus.Metric("server.proposals_committed.count"),
			"etcd_server_proposals_pending":         prometheus.Metric("server.proposals_pending.count"),
			"etcd_server_proposals_failed_total":    prometheus.Metric("server.proposals_failed.count"),
			"grpc_server_started_total":             prometheus.Metric("server.grpc_started.count"),
			"grpc_server_handled_total":             prometheus.Metric("server.grpc_handled.count"),

			// Disk
			"etcd_mvcc_db_total_size_in_bytes": prometheus.Metric("disk.mvcc_db_total_size.bytes"),
			"etcd_disk_wal_fsync_duration_seconds": prometheus.Metric("disk.wal_fsync_duration.ns",
				prometheus.OpMultiplyBuckets(1000000000)),
			"etcd_disk_backend_commit_duration_seconds": prometheus.Metric("disk.backend_commit_duration.ns",
				prometheus.OpMultiplyBuckets(1000000000)),

			// Memory
			"go_memstats_alloc_bytes": prometheus.Metric("memory.go_memstats_alloc.bytes"),

			// Network
			"etcd_network_client_grpc_sent_bytes_total":     prometheus.Metric("network.client_grpc_sent.bytes"),
			"etcd_network_client_grpc_received_bytes_total": prometheus.Metric("network.client_grpc_received.bytes"),
		},
		ExtraFields: map[string]string{"api_version": "3"},
		Namespace:   "etcd",
	}

	mb.Registry.MustAddMetricSet("etcd", "metrics",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
