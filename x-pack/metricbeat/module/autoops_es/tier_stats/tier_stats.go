// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tier_stats

import (
	"encoding/json"
	"net/http"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

const (
	TierStatsMetricSet = "tier_stats"
	// The timeout keeps Elasticsearch from waiting on slow nodes for longer than the shortest period this metricset runs with (10s).
	TierStatsPath = "/_nodes/metrics?timeout=5s"
)

// The actions of the search tier and the indexing tier whose request durations are collected, by the kind of request.
// Each one is a separate histogram named `es.<kind>.<action>.duration`.
var (
	restActions  = []string{"search", "msearch", "esql", "count", "search_scroll", "prometheus_query", "index", "update", "bulk", "prometheus_write"}
	asyncActions = []string{"esql", "eql", "sql", "async_search", "reindex", "update_by_query", "delete_by_query"}
)

// requestedMetrics are the names of all of the metrics that are requested (and the only ones that are used).
var requestedMetrics = buildRequestedMetrics()

func buildRequestedMetrics() []string {
	metrics := make([]string, 0, len(restActions)+len(asyncActions))

	for _, action := range restActions {
		metrics = append(metrics, "es.rest."+action+".duration")
	}

	for _, action := range asyncActions {
		metrics = append(metrics, "es.async."+action+".duration")
	}

	return metrics
}

// requestBody is the body of the `_nodes/metrics` request, which requires the exact list of metrics to return.
func requestBody() []byte {
	body, err := json.Marshal(map[string][]string{"metrics": requestedMetrics})

	if err != nil {
		// a map of strings cannot fail to marshal
		panic(err)
	}

	return body
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	metricset.AddServerlessAutoOpsMetricSet(TierStatsMetricSet, TierStatsPath, metricset.RequestOptions{
		Method: http.MethodPost,
		Body:   requestBody(),
	}, eventsMapping)
}
