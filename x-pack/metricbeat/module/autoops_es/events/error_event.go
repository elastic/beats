// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

// ErrEvent represents an error event in the system.
type ErrEvent struct {
	ErrorCode      string `json:"error.code"`                 // Code identifying the specific error type
	ErrorMessage   string `json:"error.message"`              // Main error message
	StackTrace     string `json:"error.stack_trace"`          // All errors concatenated similar to a stack trace
	ResourceID     string `json:"orchestrator.resource.id"`   // Cloud Resource ID (deployment, project, or cloud connected resource)
	ClusterID      string `json:"orchestrator.cluster.id"`    // Optional cluster identifier (can be unknown for authentication errors)
	URLPath        string `json:"url.path"`                   // API path of the request (without DNS/host portion)
	Query          string `json:"url.query"`                  // Query parameters of the HTTP request
	MetricSet      string `json:"event.dataset"`              // Name of the metric set that generated the error
	HTTPMethod     string `json:"http.request.method"`        // HTTP method of the request
	HTTPStatusCode int    `json:"http.response.status_code"`  // HTTP response status code
	HTTPResponse   string `json:"http.response.body.content"` // HTTP response body content
}

// SendErrorEventWithRandomTransactionId sends an error event with a random transaction id to the reporter with the provided details.
func SendErrorEventWithRandomTransactionId(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string) {
	SendErrorEvent(err, clusterInfo, r, metricSetName, path, utils.NewUUIDV4())
}

// SendErrorEvent sends an error event to the reporter with the provided details.
func SendErrorEvent(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string, transactionID string) {
	path, query := extractPathAndQuery(path)
	status, errorCode, body := getHTTPResponseBodyInfo(err)
	resourceId := getResourceID()
	lastError := getSurfaceError(err)

	errEvent := ErrEvent{
		ErrorCode:      errorCode,
		ErrorMessage:   lastError,
		StackTrace:     err.Error(),
		ResourceID:     resourceId,
		ClusterID:      clusterInfo.ClusterID,
		URLPath:        path,
		Query:          query,
		HTTPMethod:     http.MethodGet, // GET is the default HTTP method on module creation for all metricsets
		HTTPStatusCode: status,
		HTTPResponse:   body,
		MetricSet:      metricSetName,
	}

	r.Event(CreateEvent(clusterInfo, mapstr.M{"error": errEvent}, transactionID))
}

// SendErrorEventWithoutClusterInfo sends an error event without cluster info to the reporter with the provided details.
func SendErrorEventWithoutClusterInfo(err error, r mb.ReporterV2, metricSetName string) {
	status, _, body := getHTTPResponseBodyInfo(err)
	errorCode := "CLUSTER_NOT_READY"
	resourceId := getResourceID()
	lastError := getSurfaceError(err)

	emptyClusterInfo := &utils.ClusterInfo{
		ClusterName: "",
		ClusterID:   "",
		Version: utils.ClusterInfoVersion{
			Number:       &version.V{},
			Distribution: "",
		},
	}

	errEvent := ErrEvent{
		ErrorCode:      errorCode,
		ErrorMessage:   lastError,
		StackTrace:     err.Error(),
		ResourceID:     resourceId,
		ClusterID:      emptyClusterInfo.ClusterID,
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
	return 0, "UNKNOWN_ERROR", ""
}

func getSurfaceError(err error) string {
	if err != nil {
		parts := strings.SplitN(err.Error(), ":", 2) // Split the error message at the first colon
		return strings.TrimSpace(parts[0])           // Return the first part, trimmed of whitespace
	}
	return ""
}

func getResourceID() string {
	if deploymentID := os.Getenv("DEPLOYMENT_ID"); deploymentID != "" {
		return deploymentID
	} else if projectID := os.Getenv("PROJECT_ID"); projectID != "" {
		return projectID
	} else if resourceID := os.Getenv("RESOURCE_ID"); resourceID != "" {
		return resourceID
	}

	return ""
}
