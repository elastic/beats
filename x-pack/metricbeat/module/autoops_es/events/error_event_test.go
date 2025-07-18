// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"errors"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"

	"github.com/stretchr/testify/assert"
)

func TestExtractPathAndQuery(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedPath  string
		expectedQuery string
	}{
		{
			name:          "Valid URL with path and query",
			input:         "https://example.com/path/to/resource?param1=value1&param2=value2",
			expectedPath:  "/path/to/resource",
			expectedQuery: "param1=value1&param2=value2",
		},
		{
			name:          "Valid URL with path only",
			input:         "https://example.com/path/to/resource",
			expectedPath:  "/path/to/resource",
			expectedQuery: "",
		},
		{
			name:          "Valid URL with query only",
			input:         "https://example.com/?param1=value1",
			expectedPath:  "/",
			expectedQuery: "param1=value1",
		},
		{
			name:          "Invalid URL",
			input:         "://invalid-url",
			expectedPath:  "",
			expectedQuery: "",
		},
		{
			name:          "Empty URL",
			input:         "",
			expectedPath:  "",
			expectedQuery: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, query := extractPathAndQuery(tt.input)

			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, tt.expectedQuery, query)
		})
	}
}

func TestGetHTTPResponseBodyInfo(t *testing.T) {
	tests := []struct {
		name           string
		inputError     error
		expectedStatus int
		expectedCode   string
		expectedBody   string
	}{
		{
			name: "Error is of type HTTPResponse",
			inputError: &utils.HTTPResponse{
				StatusCode: 404,
				Body:       "Not Found",
			},
			expectedStatus: 404,
			expectedCode:   "HTTP_404",
			expectedBody:   "Not Found",
		},
		{
			name: "Error is of type ClusterInfoError",
			inputError: &utils.ClusterInfoError{
				Message: "Cluster not ready",
			},
			expectedStatus: 0,
			expectedCode:   "CLUSTER_NOT_READY",
			expectedBody:   "Cluster not ready",
		},
		{
			name: "Error is of type VersionMismatchError",
			inputError: &utils.VersionMismatchError{
				ExpectedVersion: "7.10.0",
				ActualVersion:   "7.9.3",
			},
			expectedStatus: 0,
			expectedCode:   "VERSION_MISMATCH",
			expectedBody:   "expected 7.10.0, got 7.9.3",
		},
		{
			name:           "Error is not of a known type",
			inputError:     errors.New("some other error"),
			expectedStatus: 0,
			expectedCode:   "UNKNOWN_ERROR",
			expectedBody:   "",
		},
		{
			name:           "Error is nil",
			inputError:     nil,
			expectedStatus: 0,
			expectedCode:   "UNKNOWN_ERROR",
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, body := getHTTPResponseBodyInfo(tt.inputError)

			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedCode, code)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
}

func TestGetResourceID(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectedValue string
	}{
		{
			name: "AUTOOPS_DEPLOYMENT_ID is set",
			envVars: map[string]string{
				"AUTOOPS_DEPLOYMENT_ID": "deployment-123",
			},
			expectedValue: "deployment-123",
		},
		{
			name: "AUTOOPS_PROJECT_ID is set",
			envVars: map[string]string{
				"AUTOOPS_PROJECT_ID": "project-456",
			},
			expectedValue: "project-456",
		},
		{
			name: "AUTOOPS_RESOURCE_ID is set",
			envVars: map[string]string{
				"AUTOOPS_RESOURCE_ID": "resource-789",
			},
			expectedValue: "resource-789",
		},
		{
			name: "No environment variables are set",
			envVars: map[string]string{
				"AUTOOPS_DEPLOYMENT_ID": "",
				"AUTOOPS_PROJECT_ID":    "",
				"AUTOOPS_RESOURCE_ID":   "",
			},
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Call the function
			result := utils.GetAndSetResourceID()

			t.Cleanup(utils.ClearResourceID)

			// Assert the result
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}
