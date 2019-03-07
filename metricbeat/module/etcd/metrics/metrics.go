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
			"etcd_server_has_leader":    prometheus.Metric("server.has_leader"),
			"leader_changes_seen_total": prometheus.Metric("server.leader_changes_seen_total"),
			"proposals_committed_total": prometheus.Metric("server.proposals_committed_total"),
			"proposals_applied_total":   prometheus.Metric("server.proposals_applied_total"),
			"proposals_pending":         prometheus.Metric("server.proposals_pending"),
			"proposals_failed_total":    prometheus.Metric("server.proposals_failed_total"),

			// Disk
			"wal_fsync_duration_seconds":      prometheus.Metric("disk.wal_fsync_duration_seconds"),
			"backend_commit_duration_seconds": prometheus.Metric("disk.backend_commit_duration_seconds"),

			// Netowrk
			"peer_sent_bytes_total":            prometheus.Metric("network.peer_sent_bytes_total"),
			"peer_received_bytes_total":        prometheus.Metric("network.peer_received_bytes_total"),
			"peer_sent_failures_total":         prometheus.Metric("network.peer_sent_failures_total"),
			"peer_received_failures_total":     prometheus.Metric("network.peer_received_failures_total"),
			"peer_round_trip_time_seconds":     prometheus.Metric("network.peer_round_trip_time_seconds"),
			"client_grpc_sent_bytes_total":     prometheus.Metric("network.client_grpc_sent_bytes_total"),
			"client_grpc_received_bytes_total": prometheus.Metric("network.client_grpc_received_bytes_total"),

			// Snapshot
			"snapshot_save_total_duration_seconds": prometheus.Metric("snapshot.save_total_duration_seconds"),
		},
	}

	mb.Registry.MustAddMetricSet("etcd", "metrics",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
