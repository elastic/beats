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

// LogAndSendErrorEventWithRandomTransactionId sends an error event with a random transaction id to the reporter with the provided details.
func LogAndSendErrorEventWithRandomTransactionId(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string) {
	LogAndSendErrorEvent(err, clusterInfo, r, metricSetName, path, utils.NewUUID())
}

// LogAndSendErrorEventWithoutClusterInfo sends an error event without cluster info to the reporter with the provided details.
func LogAndSendErrorEventWithoutClusterInfo(err error, r mb.ReporterV2, metricSetName string) {
	LogAndSendErrorEvent(err, &utils.ClusterInfo{
		Version: utils.ClusterInfoVersion{Number: &version.V{}},
	}, r, metricSetName, "/", utils.NewUUID())
}

// LogAndSendErrorEvent sends an error event to the reporter with the provided details.
func LogAndSendErrorEvent(err error, clusterInfo *utils.ClusterInfo, r mb.ReporterV2, metricSetName string, path string, transactionID string) {
	logp.Err("Error fetching data for metricset %s: %s", metricSetName, err)

	r.Event(createError(clusterInfo, err, path, transactionID))
}

func extractPathAndQuery(fullURL string) (string, string) {
	if parsedURL, err := url.Parse(fullURL); err != nil {
		// explicitly avoid returning an error here as metricset endpoint must be hit correctly
		// if not, error events won't contain path/query and will be noticed in observability dashboards
		return "", ""
	} else {
		return parsedURL.Path, parsedURL.RawQuery
	}
}

func getHTTPResponseBodyInfo(err error) (int, string, string) {
	var httpErr *utils.HTTPResponse
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode, fmt.Sprintf("HTTP_%d", httpErr.StatusCode), httpErr.Err.Error()
	}

	var clusterErr *utils.ClusterInfoError
	if errors.As(err, &clusterErr) {
		return 0, "CLUSTER_NOT_READY", clusterErr.Message
	}

	var versionErr *utils.VersionMismatchError
	if errors.As(err, &versionErr) {
		return 0, "VERSION_MISMATCH", fmt.Sprintf("expected %s, got %s", versionErr.ExpectedVersion, versionErr.ActualVersion)
	}

	if err == nil {
		return 0, "UNEXPECTED_ERROR", "unknown error"
	}

	return 0, "UNKNOWN_ERROR", err.Error()
}

// Create a new Metricbeat Event object containing expected fields and the dynamic portion.
func createError(info *utils.ClusterInfo, err error, path string, transactionID string) mb.Event {
	status, errorCode, body := getHTTPResponseBodyInfo(err)
	path, query := extractPathAndQuery(path)

	return mb.Event{
		ModuleFields: mapstr.M{
			"cluster": mapstr.M{
				"id":      info.ClusterID,
				"name":    info.ClusterName,
				"version": info.Version.Number.String(),
			},
			"transaction_id": transactionID,
		},
		RootFields: mapstr.M{
			"error": mapstr.M{
				"code":    errorCode,
				"message": body,
			},
			"event": mapstr.M{
				"kind": "metric",
				"type": "error",
			},
			"http": mapstr.M{
				"request": mapstr.M{
					"method": http.MethodGet,
				},
				"response": mapstr.M{
					"status_code": status,
				},
			},
			"orchestrator": mapstr.M{
				"resource": mapstr.M{
					"id": utils.GetResourceID(),
				},
			},
			"url": mapstr.M{
				"path":  path,
				"query": query,
			},
		},
	}
}
