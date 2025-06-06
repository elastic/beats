// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
)

// HTTPResponse represents a custom error containing HTTP status code, status, and the original error.
type HTTPResponse struct {
	StatusCode int
	Status     string
	Body       string
	Err        error
}

// Error implements the error interface for HTTPResponse, providing a formatted error message.
func (e HTTPResponse) Error() string {
	return fmt.Sprintf("%s: HTTP error %s", e.Err.Error(), e.Status)
}

// FetchAPIData fetches data from the specified path using the provided MetricSet and deserializes it into the specified type T.
func FetchAPIData[T any](m *elasticsearch.MetricSet, path string) (*T, error) {
	m.SetServiceURI(path)

	resp, err := m.FetchResponse()

	if err != nil {
		return nil, &HTTPResponse{
			StatusCode: 0,
			Status:     "500 Internal Server Error",
			Body:       "",
			Err:        err,
		}
	}

	defer resp.Body.Close()

	if body, readErr := io.ReadAll(resp.Body); readErr != nil {
		return nil, &HTTPResponse{
			StatusCode: resp.StatusCode,
			Status:     "failed to read response body",
			Body:       "",
			Err:        err,
		}
	} else if resp.StatusCode != 200 {
		return nil, &HTTPResponse{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
			Err:        fmt.Errorf("failed to fetch data"),
		}
	} else if data, deserializeErr := DeserializeData[T](body); deserializeErr != nil {
		return nil, &HTTPResponse{
			StatusCode: resp.StatusCode,
			Status:     "500 Internal Server Error",
			Body:       string(body),
			Err:        deserializeErr,
		}
	} else {
		return data, nil
	}
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
