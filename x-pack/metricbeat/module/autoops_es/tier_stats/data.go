// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tier_stats

import (
	"strconv"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	e "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const (
	nodeTierAttribute          = "node.tier"
	serviceInstanceIDAttribute = "service.instance.id"
	statusCodeAttribute        = "http.response.status_code"
	histogramType              = "histogram"

	// the root-level field with the milliseconds since the previous scrape
	timestampDiffField = "timestamp_diff"
)

// NodesMetricsResponse is the response of `_nodes/metrics`, which has one entry per node. Everything that is not needed is ignored.
type NodesMetricsResponse struct {
	ResourceMetrics []ResourceMetrics `json:"resource_metrics"`
}

// ResourceMetrics are the metrics of one node.
type ResourceMetrics struct {
	ResourceAttributes map[string]any    `json:"resource_attributes"`
	Metrics            map[string]Metric `json:"metrics"`
}

// Metric is one metric of a node. Only histograms are used, whose `count` is the number of requests and `sum` is their total duration in
// milliseconds. Both are cumulative since the node started.
type Metric struct {
	Type         string        `json:"type"`
	Measurements []Measurement `json:"measurements"`
}

// Measurement is the histogram for one set of attributes (e.g., the HTTP status code of the response).
type Measurement struct {
	Sum        float64        `json:"sum"`
	Count      int64          `json:"count"`
	Attributes map[string]any `json:"attributes"`
}

// seriesKey identifies one cumulative histogram of one node.
type seriesKey struct {
	instance string
	metric   string
	status   int
}

// seriesValue is the cumulative histogram: the number of requests and their total duration in milliseconds.
type seriesValue struct {
	count int64
	sum   float64
}

// snapshot is what a response says about the cumulative histograms at the time of a scrape, in the form that is needed to compare it with
// another one.
type snapshot struct {
	series map[seriesKey]seriesValue
	// the nodes (by their instance ID) and the tier that each one belongs to, including the nodes that have no histograms
	instances map[string]string
}

// readSnapshot reads the response, which is only the part of it that is used:
//
//   - nodes without an instance ID or a tier cannot be attributed to a tier, so they are ignored
//   - only the requested metrics are used, and only if they are histograms
//   - measurements without an HTTP status code are ignored
//   - measurements of the same node, metric, and HTTP status code (which only differ in other attributes) are added together
func readSnapshot(response *NodesMetricsResponse) snapshot {
	current := snapshot{
		series:    map[seriesKey]seriesValue{},
		instances: map[string]string{},
	}

	for _, node := range response.ResourceMetrics {
		instance, hasInstance := stringAttribute(node.ResourceAttributes, serviceInstanceIDAttribute)
		tier, hasTier := stringAttribute(node.ResourceAttributes, nodeTierAttribute)

		if !hasInstance || !hasTier {
			continue
		}

		current.instances[instance] = tier

		for _, name := range requestedMetrics {
			metric, found := node.Metrics[name]

			if !found || metric.Type != histogramType {
				continue
			}

			for _, measurement := range metric.Measurements {
				status, ok := statusCode(measurement.Attributes)

				if !ok {
					continue
				}

				key := seriesKey{instance: instance, metric: name, status: status}
				value := current.series[key]
				value.count += measurement.Count
				value.sum += measurement.Sum
				current.series[key] = value
			}
		}
	}

	return current
}

// stringAttribute gets a non-empty string attribute.
func stringAttribute(attributes map[string]any, name string) (string, bool) {
	value, ok := attributes[name].(string)

	return value, ok && value != ""
}

// statusCode gets the HTTP status code of a measurement, which is a number but is tolerated as a string.
func statusCode(attributes map[string]any) (int, bool) {
	switch value := attributes[statusCodeAttribute].(type) {
	case float64:
		if value > 0 {
			return int(value), true
		}
	case string:
		if code, err := strconv.Atoi(value); err == nil && code > 0 {
			return code, true
		}
	}

	return 0, false
}

// eventsMapping reports the change since the previous scrape as one event per tier and HTTP status code.
//
// Only the nodes that answered are in the response, so the sums of a scrape in which a node did not answer do not include it.
func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, response *NodesMetricsResponse) error {
	docs := tierStatsState.compute(readSnapshot(response), time.Now().UnixMilli())

	if len(docs) == 0 {
		return nil
	}

	transactionID := utils.NewUUID()

	for index, doc := range docs {
		// the milliseconds between the scrapes is not specific to this metricset: it is the root-level timestamp_diff that other
		// documents (such as the ones of the shards) already have, so it is not one of the fields of the metricset
		timestampDiff := doc[timestampDiffField]
		delete(doc, timestampDiffField)

		event := e.CreateEvent(info, doc, transactionID)
		event.RootFields[timestampDiffField] = timestampDiff

		e.ReportEvent(r, event, index, len(docs))
	}

	return nil
}
