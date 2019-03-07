package metrics

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			// Server
			"etcd_server_has_leader":                prometheus.Metric("server.has_leader"),
			"etcd_server_leader_changes_seen_total": prometheus.Metric("server.leader_changes_seen_total"),
			"etcd_server_proposals_committed_total": prometheus.Metric("server.proposals_committed_total"),
			"etcd_server_proposals_pending":         prometheus.Metric("server.proposals_pending"),
			"etcd_server_proposals_failed_total":    prometheus.Metric("server.proposals_failed_total"),
			"grpc_server_started_total":             prometheus.Metric("server.grpc_started_total"),
			"grpc_server_handled_total":             prometheus.Metric("server.grpc_handled_total"),

			// Disk
			"etcd_mvcc_db_total_size_in_bytes":          prometheus.Metric("disk.mvcc_db_total_size_in_bytes"),
			"etcd_disk_wal_fsync_duration_seconds":      prometheus.Metric("disk.wal_fsync_duration_seconds"),
			"etcd_disk_backend_commit_duration_seconds": prometheus.Metric("disk.backend_commit_duration_seconds"),

			// Memory
			"go_memstats_alloc_bytes": prometheus.Metric("memory.go_memstats_alloc_bytes"),

			// Network
			"etcd_network_client_grpc_sent_bytes_total":     prometheus.Metric("network.client_grpc_sent_bytes_total"),
			"etcd_network_client_grpc_received_bytes_total": prometheus.Metric("network.client_grpc_received_bytes_total"),
			"etcd_network_peer_sent_bytes_total":            prometheus.Metric("network.peer_sent_bytes_total"),
			"etcd_network_peer_received_bytes_total":        prometheus.Metric("network.peer_received_bytes_total"),
			"etcd_network_peer_sent_failures_total":         prometheus.Metric("network.peer_sent_failures_total"),
			"etcd_network_peer_received_failures_total":     prometheus.Metric("network.peer_received_failures_total"),
		},
	}

	mb.Registry.MustAddMetricSet("etcd", "metrics",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
