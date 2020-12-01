// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package website

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/iis"

	"github.com/elastic/beats/v7/metricbeat/helper/windows/pdh"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

const ecsProcessId = "process.pid"

// Reader will contain the config options
type WebsiteReader struct {
	query    pdh.Query    // PDH Query
	executed bool         // Indicates if the query has been executed.
	log      *logp.Logger //
	config   iis.Config   // Metricset configuration
}

var websiteCounters = map[string]string{
	"network.total_bytes_received":      "\\Web Service(*)\\Total Bytes Received",
	"network.total_bytes_sent":          "\\Web Service(*)\\Total Bytes Sent",
	"network.bytes_sent_per_sec":        "\\Web Service(*)\\Bytes Sent/sec",
	"network.bytes_received_per_sec":    "\\Web Service(*)\\Bytes Received/sec",
	"network.current_connections":       "\\Web Service(*)\\Current Connections",
	"network.maximum_connections":       "\\Web Service(*)\\Maximum Connections",
	"network.total_connection_attempts": "\\Web Service(*)\\Total Connection Attempts (all instances)",
	"network.total_get_requests":        "\\Web Service(*)\\Total Get Requests",
	"network.get_requests_per_sec":      "\\Web Service(*)\\Get Requests/sec",
	"network.total_post_requests":       "\\Web Service(*)\\Total Post Requests",
	"network.post_requests_per_sec":     "\\Web Service(*)\\Post Requests/sec",
	"network.total_delete_requests":     "\\Web Service(*)\\Total Delete Requests",
	"network.delete_requests_per_sec":   "\\Web Service(*)\\Delete Requests/sec",
	"network.service_uptime":            "\\Web Service(*)\\Service Uptime",
	"network.total_put_requests":        "\\Web Service(*)\\Total PUT Requests",
	"network.put_requests_per_sec":      "\\Web Service(*)\\PUT Requests/sec",
}

// newReader creates a new instance of Reader.
func NewReader(config iis.Config) (*WebsiteReader, error) {
	var query pdh.Query
	if err := query.Open(); err != nil {
		return nil, err
	}
	r := &WebsiteReader{
		query:  query,
		log:    logp.NewLogger("website"),
		config: config,
	}

	err := r.InitCounters()
	if err != nil {
		return nil, errors.Wrap(err, "error loading counters for existing app pools")
	}
	return r, nil
}

// initAppPools will check for any new instances and add them to the counter list
func (r *WebsiteReader) InitCounters() error {
	for _, value := range websiteCounters {

		if err := r.query.AddCounter(value, "*", "float", true); err != nil {
			return errors.Wrapf(err, `failed to add counter (query="%v")`, value)
		}
	}

	return nil
}

// read executes a query and returns those values in an event.
func (r *WebsiteReader) Read() ([]mb.Event, error) {
	// Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	if err := r.query.CollectData(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := r.query.GetFormattedCounterArrayValues()
	_ = values
	if err != nil {
		r.query.Close()
		return nil, errors.Wrap(err, "failed formatting counter values")
	}
	var events []mb.Event
	//eventGroup := r.mapEvents(values)
	r.executed = true
	results := make([]mb.Event, 0, len(events))
	//for _, val := range eventGroup {
	//	results = append(results, val)
	//}
	return results, nil
}

// close will close the PDH query for now.
func (r *WebsiteReader) Close() error {
	return r.query.Close()
}
