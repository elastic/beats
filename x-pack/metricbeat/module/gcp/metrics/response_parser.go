// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
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
func (e *incomingFieldExtractor) extractTimeSeriesMetricValues(resp *monitoring.TimeSeries, aligner string) (points []KeyValuePoint, err error) {
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

	return points, nil
}

func (e *incomingFieldExtractor) getTimestamp(p *monitoring.Point) (ts time.Time, err error) {
	// Don't add point intervals that can't be "stated" at some timestamp.
	if p.Interval != nil {
		if ts, err = ptypes.Timestamp(p.Interval.EndTime); err != nil {
			return time.Time{}, errors.Errorf("error trying to parse timestamp '%#v' from metric\n", p.Interval.EndTime)
		}
		return ts, nil
	}

	return time.Time{}, errors.New("error trying to extract the timestamp from the point data")
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
		//TODO Distribution values aren't simple values. Take a look at this
		out = v.DistributionValue
	}

	return out
}
