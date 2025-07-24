// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

// ErrorEvent represents an error event in the system.
// These events are used to report errors that occur during the execution of metricsets as a metricset field.
// Because how metricbeat works, at the end errors become mb.BeatEvent and another event sent as side effect containing error.message.
type ErrorEvent struct {
	ErrorCode      string `json:"error.code"`                 // Code identifying the specific error type
	ErrorMessage   string `json:"error.message"`              // Full error message
	URLPath        string `json:"url.path"`                   // API path of the request (without DNS/host portion)
	Query          string `json:"url.query"`                  // Query parameters of the HTTP request
	MetricSet      string `json:"event.dataset"`              // Name of the metric set that generated the error
	HTTPMethod     string `json:"http.request.method"`        // HTTP method of the request
	HTTPStatusCode int    `json:"http.response.status_code"`  // HTTP response status code
	HTTPResponse   string `json:"http.response.body.content"` // HTTP response body content
}

// LogAndSendErrorEventWithRandomTransactionId sends an error event with a random transaction id to the reporter with the provided details.
func LogAndSendErrorEventWithRandomTransactionId(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string) {
	LogAndSendErrorEvent(err, clusterInfo, r, metricSetName, path, utils.NewUUIDV4())
}

// LogAndSendErrorEvent sends an error event to the reporter with the provided details.
func LogAndSendErrorEvent(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string, transactionID string) {
	logp.Err("Error fetching data for metricset %s: %s", metricSetName, err)
	path, query := extractPathAndQuery(path)
	status, errorCode, body := getHTTPResponseBodyInfo(err)

	errEvent := ErrorEvent{
		ErrorCode:      errorCode,
		ErrorMessage:   err.Error(),
		URLPath:        path,
		Query:          query,
		HTTPMethod:     http.MethodGet, // GET is the default HTTP method on module creation for all metricsets
		HTTPStatusCode: status,
		HTTPResponse:   body,
		MetricSet:      metricSetName,
	}

	r.Event(CreateEvent(clusterInfo, mapstr.M{"error": errEvent}, transactionID))
}

// LogAndSendErrorEventWithoutClusterInfo sends an error event without cluster info to the reporter with the provided details.
func LogAndSendErrorEventWithoutClusterInfo(err error, r mb.ReporterV2, metricSetName string) {
	logp.Err("Error fetching data for metricset %s: %s", metricSetName, err)
	status, errorCode, body := getHTTPResponseBodyInfo(err)

	emptyClusterInfo := &utils.ClusterInfo{
		ClusterName: "",
		ClusterID:   "",
		Version: utils.ClusterInfoVersion{
			Number:       &version.V{},
			Distribution: "",
		},
	}

	errEvent := ErrorEvent{
		ErrorCode:      errorCode,
		ErrorMessage:   err.Error(),
		URLPath:        "/",
		Query:          "",
		HTTPMethod:     http.MethodGet, // GET is the default method on module creation
		HTTPStatusCode: status,         // when cluster is not ready API can return several different errors depending on the specific issue
		HTTPResponse:   body,
		MetricSet:      metricSetName,
	}

	r.Event(CreateEventWithRandomTransactionId(emptyClusterInfo, mapstr.M{"error": errEvent}))
}

func extractPathAndQuery(fullURL string) (string, string) {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		// explicitly avoid returning an error here as metricset endpoint must be hit correctly
		// if not, error events won't contain path/query and will be noticed in observability dashboards
		return "", ""
	}

	return parsedURL.Path, parsedURL.RawQuery
}

func getHTTPResponseBodyInfo(err error) (int, string, string) {
	var httpErr *utils.HTTPResponse
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode, fmt.Sprintf("HTTP_%d", httpErr.StatusCode), httpErr.Body
	}

	var clusterErr *utils.ClusterInfoError
	if errors.As(err, &clusterErr) {
		return 0, "CLUSTER_NOT_READY", clusterErr.Message
	}

	var versionErr *utils.VersionMismatchError
	if errors.As(err, &versionErr) {
		return 0, "VERSION_MISMATCH", fmt.Sprintf("expected %s, got %s", versionErr.ExpectedVersion, versionErr.ActualVersion)
	}

	return 0, "UNKNOWN_ERROR", ""
}
