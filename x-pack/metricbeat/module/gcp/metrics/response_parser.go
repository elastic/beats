// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/gcp"
)

func newIncomingFieldExtractor(l *logp.Logger, mc metricsConfig) *incomingFieldExtractor {
	return &incomingFieldExtractor{logger: l, mc: mc}
}

type incomingFieldExtractor struct {
	logger *logp.Logger
	mc     metricsConfig
}

// KeyValuePoint is a struct to capture the information parsed in an instant of a single metric
type KeyValuePoint struct {
	Key       string
	Value     interface{}
	Labels    common.MapStr
	ECS       common.MapStr
	Timestamp time.Time
}

// extractTimeSeriesMetricValues valuable to send to Elasticsearch. This includes, for example, metric values, labels and timestamps
func (e *incomingFieldExtractor) extractTimeSeriesMetricValues(resp *monitoring.TimeSeries, aligner string) (points []KeyValuePoint) {
	points = make([]KeyValuePoint, 0)

	for _, point := range resp.Points {
		// Don't add point intervals that can't be "stated" at some timestamp.
		ts, err := e.getTimestamp(point)
		if err != nil {
			e.logger.Warn(err)
			continue
		}

		p := KeyValuePoint{
			Key:       remap(e.logger, cleanMetricNameString(resp.Metric.Type, aligner, e.mc)),
			Value:     getValueFromPoint(point),
			Timestamp: ts,
		}

		points = append(points, p)
	}

	return points
}

func (e *incomingFieldExtractor) getTimestamp(p *monitoring.Point) (ts time.Time, err error) {
	// Don't add point intervals that can't be "stated" at some timestamp.
	if p.Interval != nil {
		return p.Interval.EndTime.AsTime(), nil
	}

	return time.Time{}, fmt.Errorf("error trying to extract the timestamp from the point data")
}

func cleanMetricNameString(s string, aligner string, mc metricsConfig) string {
	if s == "" {
		return "unknown"
	}

	unprefixedMetric := mc.RemovePrefixFrom(s)
	replacedChars := strings.Replace(unprefixedMetric, "/", ".", -1)

	metricName := replacedChars + gcp.AlignersMapToSuffix[aligner]
	return metricName
}

var reMapping = map[string]string{
	// gcp.compute metricset
	"firewall.dropped_bytes_count.value":                 "firewall.dropped.bytes",
	"instance.cpu.usage_time.value":                      "instance.cpu.usage_time.sec",
	"instance.cpu.utilization.value":                     "instance.cpu.usage.pct",
	"instance.disk.read_bytes_count.value":               "instance.disk.read.bytes",
	"instance.disk.write_bytes_count.value":              "instance.disk.write.bytes",
	"instance.memory.balloon.swap_in_bytes_count.value":  "instance.memory.balloon.swap_in.bytes",
	"instance.memory.balloon.swap_out_bytes_count.value": "instance.memory.balloon.swap_out.bytes",
	"instance.network.received_bytes_count.value":        "instance.network.ingress.bytes",
	"instance.network.received_packets_count.value":      "instance.network.ingress.packets.count",
	"instance.network.sent_bytes_count.value":            "instance.network.egress.bytes",
	"instance.network.sent_packets_count.value":          "instance.network.egress.packets.count",
	"instance.uptime.value":                              "instance.uptime.sec",
	"instance.uptime_total.value":                        "instance.uptime_total.sec",

	// gcp.gke metricset
	"container.cpu.core_usage_time.value":             "container.cpu.core_usage_time.sec",
	"container.cpu.limit_utilization.value":           "container.cpu.limit_utilization.pct",
	"container.cpu.request_utilization.value":         "container.cpu.request_utilization.pct",
	"container.ephemeral_storage.limit_bytes.value":   "container.ephemeral_storage.limit.bytes",
	"container.ephemeral_storage.request_bytes.value": "container.ephemeral_storage.request.bytes",
	"container.ephemeral_storage.used_bytes.value":    "container.ephemeral_storage.used.bytes",
	"container.memory.limit_bytes.value":              "container.memory.limit.bytes",
	"container.memory.limit_utilization.value":        "container.memory.limit_utilization.pct",
	"container.memory.page_fault_count.value":         "container.memory.page_fault.count",
	"container.memory.request_bytes.value":            "container.memory.request.bytes",
	"container.memory.request_utilization.value":      "container.memory.request_utilization.pct",
	"container.memory.used_bytes.value":               "container.memory.used.bytes",
	"container.restart_count.value":                   "container.restart.count",
	"container.uptime.value":                          "container.uptime.sec",
	"node.cpu.allocatable_utilization.value":          "node.cpu.allocatable_utilization.pct",
	"node.cpu.core_usage_time.value":                  "node.cpu.core_usage_time.sec",
	"node.ephemeral_storage.allocatable_bytes.value":  "node.ephemeral_storage.allocatable.bytes",
	"node.ephemeral_storage.total_bytes.value":        "node.ephemeral_storage.total.bytes",
	"node.ephemeral_storage.used_bytes.value":         "node.ephemeral_storage.used.bytes",
	"node.memory.allocatable_bytes.value":             "node.memory.allocatable.bytes",
	"node.memory.allocatable_utilization.value":       "node.memory.allocatable_utilization.pct",
	"node.memory.total_bytes.value":                   "node.memory.total.bytes",
	"node.memory.used_bytes.value":                    "node.memory.used.bytes",
	"node.network.received_bytes_count.value":         "node.network.received.bytes",
	"node.network.sent_bytes_count.value":             "node.network.sent.bytes",
	"node_daemon.cpu.core_usage_time.value":           "node_daemon.cpu.core_usage_time.sec",
	"node_daemon.memory.used_bytes.value":             "node_daemon.memory.used.bytes",
	"pod.network.received_bytes_count.value":          "pod.network.received.bytes",
	"pod.network.sent_bytes_count.value":              "pod.network.sent.bytes",
	"pod.volume.total_bytes.value":                    "pod.volume.total.bytes",
	"pod.volume.used_bytes.value":                     "pod.volume.used.bytes",
	"pod.volume.utilization.value":                    "pod.volume.utilization.pct",

	// gcp.loadbalancing metricset
	"https.backend_request_bytes_count.value":  "https.backend_request.bytes",
	"https.backend_request_count.value":        "https.backend_request.count",
	"https.backend_response_bytes_count.value": "https.backend_response.bytes",
	"https.request_bytes_count.value":          "https.request.bytes",
	"https.request_count.value":                "https.request.count",
	"https.response_bytes_count.value":         "https.response.bytes",
	"l3.external.egress_bytes_count.value":     "l3.external.egress.bytes",
	"l3.external.egress_packets_count.value":   "l3.external.egress_packets.count",
	"l3.external.ingress_bytes_count.value":    "l3.external.ingress.bytes",
	"l3.external.ingress_packets_count.value":  "l3.external.ingress_packets.count",
	"l3.internal.egress_bytes_count.value":     "l3.internal.egress.bytes",
	"l3.internal.egress_packets_count.value":   "l3.internal.egress_packets.count",
	"l3.internal.ingress_bytes_count.value":    "l3.internal.ingress.bytes",
	"l3.internal.ingress_packets_count.value":  "l3.internal.ingress_packets.count",
	"tcp_ssl_proxy.egress_bytes_count.value":   "tcp_ssl_proxy.egress.bytes",
	"tcp_ssl_proxy.ingress_bytes_count.value":  "tcp_ssl_proxy.ingress.bytes",

	// gcp.metrics metricset
	// NOTE: nothing here; if the user directly uses this metricset the mapping to ECS is
	// unpredictable.
	// Following the least surprise principle, instead of trying to convert it, leave it as it is.
	// To prevent users from using this metricset directly, proper metricset should be implemented
	// to adhere to ECS and Beats naming conventions.

	// gcp.pubsub metricset
	"snapshot.backlog_bytes.value":                                               "snapshot.backlog.bytes",
	"snapshot.backlog_bytes_by_region.value":                                     "snapshot.backlog_bytes_by_region.bytes",
	"snapshot.config_updates_count.value":                                        "snapshot.config_updates.count",
	"snapshot.oldest_message_age.value":                                          "snapshot.oldest_message_age.sec",
	"snapshot.oldest_message_age_by_region.value":                                "snapshot.oldest_message_age_by_region.sec",
	"subscription.ack_message_count.value":                                       "subscription.ack_message.count",
	"subscription.backlog_bytes.value":                                           "subscription.backlog.bytes",
	"subscription.byte_cost.value":                                               "subscription.byte_cost.bytes",
	"subscription.config_updates_count.value":                                    "subscription.config_updates.count",
	"subscription.dead_letter_message_count.value":                               "subscription.dead_letter_message.count",
	"subscription.mod_ack_deadline_message_count.value":                          "subscription.mod_ack_deadline_message.count",
	"subscription.mod_ack_deadline_message_operation_count.value":                "subscription.mod_ack_deadline_message_operation.count",
	"subscription.mod_ack_deadline_request_count.value":                          "subscription.mod_ack_deadline_request.count",
	"subscription.oldest_retained_acked_message_age.value":                       "subscription.oldest_retained_acked_message_age.sec",
	"subscription.oldest_retained_acked_message_age_by_region.value":             "subscription.oldest_retained_acked_message_age_by_region.value",
	"subscription.oldest_unacked_message_age.value":                              "subscription.oldest_unacked_message_age.sec",
	"subscription.oldest_unacked_message_age_by_region.value":                    "subscription.oldest_unacked_message_age_by_region.value",
	"subscription.pull_ack_message_operation_count.value":                        "subscription.pull_ack_message_operation.count",
	"subscription.pull_ack_request_count.value":                                  "subscription.pull_ack_request.count",
	"subscription.pull_message_operation_count.value":                            "subscription.pull_message_operation.count",
	"subscription.pull_request_count.value":                                      "subscription.pull_request.count",
	"subscription.push_request_count.value":                                      "subscription.push_request.count",
	"subscription.retained_acked_bytes.value":                                    "subscription.retained_acked.bytes",
	"subscription.retained_acked_bytes_by_region.value":                          "subscription.retained_acked_bytes_by_region.bytes",
	"subscription.seek_request_count.value":                                      "subscription.seek_request.count",
	"subscription.sent_message_count.value":                                      "subscription.sent_message.count",
	"subscription.streaming_pull_ack_message_operation_count.value":              "subscription.streaming_pull_ack_message_operation.count",
	"subscription.streaming_pull_ack_request_count.value":                        "subscription.streaming_pull_ack_request.count",
	"subscription.streaming_pull_message_operation_count.value":                  "subscription.streaming_pull_message_operation.count",
	"subscription.streaming_pull_mod_ack_deadline_message_operation_count.value": "subscription.streaming_pull_mod_ack_deadline_message_operation.count",
	"subscription.streaming_pull_mod_ack_deadline_request_count.value":           "subscription.streaming_pull_mod_ack_deadline_request.count",
	"subscription.streaming_pull_response_count.value":                           "subscription.streaming_pull_response.count",
	"subscription.unacked_bytes_by_region.value":                                 "subscription.unacked_bytes_by_region.bytes",
	"topic.byte_cost.value":                                                      "topic.byte_cost.bytes",
	"topic.config_updates_count.value":                                           "topic.config_updates.count",
	"topic.message_sizes.value":                                                  "topic.message_sizes.bytes",
	"topic.oldest_retained_acked_message_age_by_region.value":                    "topic.oldest_retained_acked_message_age_by_region.value",
	"topic.oldest_unacked_message_age_by_region.value":                           "topic.oldest_unacked_message_age_by_region.value",
	"topic.retained_acked_bytes_by_region.value":                                 "topic.retained_acked_bytes_by_region.bytes",
	"topic.send_message_operation_count.value":                                   "topic.send_message_operation.count",
	"topic.send_request_count.value":                                             "topic.send_request.count",
	"topic.streaming_pull_response_count.value":                                  "topic.streaming_pull_response.count",
	"topic.unacked_bytes_by_region.value":                                        "topic.unacked_bytes_by_region.bytes",

	// gcp.storage metricset
	"api.request_count.value":                        "api.request.count",
	"authz.acl_based_object_access_count.value":      "authz.acl_based_object_access.count",
	"authz.acl_operations_count.value":               "authz.acl_operations.count",
	"authz.object_specific_acl_mutation_count.value": "authz.object_specific_acl_mutation.count",
	"network.received_bytes_count.value":             "network.received.bytes",
	"network.sent_bytes_count.value":                 "network.sent.bytes",
	"storage.object_count.value":                     "storage.object.count",
	"storage.total_byte_seconds.value":               "storage.total_byte_seconds.bytes",
	"storage.total_bytes.value":                      "storage.total.bytes",

	// gcp.firestore metricset
	"document.delete_count.value": "document.delete.count",
	"document.read_count.value":   "document.read.count",
	"document.write_count.value":  "document.write.count",
	// gcp.dataproc
	"batch.spark.executors.value":                    "batch.spark.executors.count",
	"cluster.hdfs.datanodes.value":                   "cluster.hdfs.datanodes.count",
	"cluster.hdfs.storage_capacity.value":            "cluster.hdfs.storage_capacity.value",
	"cluster.hdfs.storage_utilization.value":         "cluster.hdfs.storage_utilization.value",
	"cluster.hdfs.unhealthy_blocks.value":            "cluster.hdfs.unhealthy_blocks.count",
	"cluster.job.failed_count.value":                 "cluster.job.failed.count",
	"cluster.job.running_count.value":                "cluster.job.running.count",
	"cluster.job.submitted_count.value":              "cluster.job.submitted.count",
	"cluster.operation.failed_count.value":           "cluster.operation.failed.count",
	"cluster.operation.running_count.value":          "cluster.operation.running.count",
	"cluster.operation.submitted_count.value":        "cluster.operation.submitted.count",
	"cluster.yarn.allocated_memory_percentage.value": "cluster.yarn.allocated_memory_percentage.value",
	"cluster.yarn.apps.value":                        "cluster.yarn.apps.count",
	"cluster.yarn.containers.value":                  "cluster.yarn.containers.count",
	"cluster.yarn.memory_size.value":                 "cluster.yarn.memory_size.value",
	"cluster.yarn.nodemanagers.value":                "cluster.yarn.nodemanagers.count",
	"cluster.yarn.pending_memory_size.value":         "cluster.yarn.pending_memory_size.value",
	"cluster.yarn.virtual_cores.value":               "cluster.yarn.virtual_cores.count",
}

func remap(l *logp.Logger, s string) string {
	var newS string

	if v, found := reMapping[s]; found {
		l.Debugf("remapping %s to %s", s, newS)
		return v
	}

	l.Debugf("no remap found for %s", s)
	return s
}

func getValueFromPoint(p *monitoring.Point) (out interface{}) {
	switch v := p.Value.Value.(type) {
	case *monitoring.TypedValue_DoubleValue:
		out = v.DoubleValue
	case *monitoring.TypedValue_BoolValue:
		out = v.BoolValue
	case *monitoring.TypedValue_Int64Value:
		out = v.Int64Value
	case *monitoring.TypedValue_StringValue:
		out = v.StringValue
	case *monitoring.TypedValue_DistributionValue:
		// Distribution values aren't simple values. Take a look at this
		out = v.DistributionValue
	}

	return out
}
