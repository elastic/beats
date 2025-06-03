// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

// ErrEvent represents an error event in the system.
type ErrEvent struct {
	ErrorCode    string `json:"error_code"`    // Code identifying the specific error type
	ErrorMessage string `json:"error_message"` // Full error message
	ResourceID   string `json:"resource_id"`   // Cloud Resource ID (deployment, project, or cloud connected resource)
	ClusterID    string `json:"cluster_id"`    // Optional cluster identifier (can be unknown for authentication errors)
	Path         string `json:"path"`          // API path of the request (without DNS/host portion)
	MetricSet    string `json:"metric_set"`    // Name of the metric set that generated the error
	Context      string `json:"context"`       // Additional contextual information like index name, node, template, etc.
}

// SendErrorEventWithRandomTransactionId sends an error event with a random transaction id to the reporter with the provided details.
func SendErrorEventWithRandomTransactionId(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string) {
	SendErrorEvent(err, clusterInfo, r, metricSetName, path, utils.NewUUIDV4())
}

// SendErrorEvent sends an error event to the reporter with the provided details.
func SendErrorEvent(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string, transactionID string) {
	errorCode := getErrorCode(err)
	resourceId := getResourceID()
	lastError := getSurfaceError(err)

	errEvent := ErrEvent{
		ErrorCode:    errorCode,
		ErrorMessage: lastError,
		ResourceID:   resourceId,
		ClusterID:    clusterInfo.ClusterID,
		Path:         path,
		MetricSet:    metricSetName,
		Context:      err.Error(),
	}

	r.Event(CreateEvent(clusterInfo, mapstr.M{"error": errEvent}, transactionID))
}

// SendErrorEventWithoutClusterInfo sends an error event without cluster info to the reporter with the provided details.
func SendErrorEventWithoutClusterInfo(err error, r mb.ReporterV2, metricSetName string) {
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
		ErrorCode:    errorCode,
		ErrorMessage: lastError,
		ResourceID:   resourceId,
		ClusterID:    emptyClusterInfo.ClusterID,
		Path:         "/",
		MetricSet:    metricSetName,
		Context:      err.Error(),
	}

	r.Event(CreateEventWithRandomTransactionId(emptyClusterInfo, mapstr.M{"error": errEvent}))
}

func getErrorCode(err error) string {
	var httpErr *utils.HTTPResponse
	if errors.As(err, &httpErr) {
		return fmt.Sprintf("HTTP_%d", httpErr.StatusCode)
	}
	return "UNKNOWN_ERROR"
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
