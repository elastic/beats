// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

// The structure of error responses from the Cloud Connect API.
type cloudConnectErrorResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// The structure of error responses from Elasticsearch APIs. (Ignores sub-reasons aka "root_cause" and "status")
type elasticsearchErrorResponse struct {
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
}

// HTTPResponse represents a custom error containing HTTP status code, status, and the original error.
type HTTPResponse struct {
	StatusCode int
	Status     string
	Body       string
	Err        error
}

// Error implements the error interface for HTTPResponse, providing a formatted error message.
func (e HTTPResponse) Error() string {
	return e.Err.Error()
}

// Attempt to get a more structured error response from either Elasticsearch or Cloud Connect API.
func deserializeErrorResponse(resp *http.Response, body []byte) *HTTPResponse {
	var err error
	var status string
	var message string

	if elasticsearchResponse, deserializeErr := DeserializeData[elasticsearchErrorResponse](body); deserializeErr == nil && elasticsearchResponse.Error.Reason != "" {
		status = elasticsearchResponse.Error.Type
		message = elasticsearchResponse.Error.Reason

		err = fmt.Errorf("error from Elasticsearch [%s]: %s", status, message)
	} else if cloudConnectedResponse, deserializeErr := DeserializeData[cloudConnectErrorResponse](body); deserializeErr == nil && len(cloudConnectedResponse.Errors) > 0 {
		status = cloudConnectedResponse.Errors[0].Code
		message = cloudConnectedResponse.Errors[0].Message

		err = fmt.Errorf("error from Cloud Connect API [%s]: %s", status, message)
	} else {
		status = resp.Status
		message = string(body)

		err = fmt.Errorf("failed to fetch data: HTTP status %d with body %s", resp.StatusCode, message)
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     status,
		Body:       message,
		Err:        err,
	}
}

// HandleHTTPResponse handles the HTTP response and deserializes it into the specified type T.
// This will appropriately close the response body.
func HandleHTTPResponse[T any](resp *http.Response, err error) (*T, error) {
	if err != nil {
		return nil, &HTTPResponse{
			StatusCode: 0,
			Status:     "failed to send request",
			Body:       "",
			Err:        fmt.Errorf("failed to send request: %w", err),
		}
	}

	defer resp.Body.Close()

	if body, readErr := io.ReadAll(resp.Body); readErr != nil {
		return nil, &HTTPResponse{
			StatusCode: resp.StatusCode,
			Status:     "failed to read response body",
			Body:       "",
			Err:        readErr,
		}
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, deserializeErrorResponse(resp, body)
	} else if data, deserializeErr := DeserializeData[T](body); deserializeErr != nil {
		return nil, &HTTPResponse{
			StatusCode: resp.StatusCode,
			Status:     "failed to deserialize data",
			Body:       string(body),
			Err:        deserializeErr,
		}
	} else {
		return data, nil
	}
}

// FetchAPIData fetches data from the specified path using the provided MetricSet and deserializes it into the specified type T.
func FetchAPIData[T any](m *elasticsearch.MetricSet, path string) (*T, error) {
	m.SetServiceURI(path)

	return HandleHTTPResponse[T](m.FetchResponse()) //nolint:bodyclose // the handler closes the body
}

// Deserialize the data to match the expected type, T. Note that success does not mean that fields are populated, which requires a schema
// to validate!
func DeserializeData[T any](content []byte) (*T, error) {
	var data T

	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to deserialize data: %w", err)
	}

	return &data, nil
}
