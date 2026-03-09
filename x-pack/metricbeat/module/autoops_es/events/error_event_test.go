// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package events

import (
	"errors"
	"net/http"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
				Err:        errors.New("Not Found"),
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
			expectedBody:   "some other error",
		},
		{
			name:           "Error is nil",
			inputError:     nil,
			expectedStatus: 0,
			expectedCode:   "UNEXPECTED_ERROR",
			expectedBody:   "unknown error",
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

// MockReporter is a mock implementation of mb.ReporterV2
type MockReporter struct {
	mock.Mock
}

func (m *MockReporter) Event(event mb.Event) bool {
	args := m.Called(event)
	return args.Bool(0)
}

func (m *MockReporter) Error(err error) bool {
	args := m.Called(err)
	return args.Bool(0)
}

func TestLogAndSendErrorEventWithoutClusterInfoDefaultValues(t *testing.T) {
	mockReporter := new(MockReporter)
	mockReporter.On("Event", mock.Anything).Return(true)

	err := errors.New("test error")
	metricSetName := "test_metricset"

	LogAndSendErrorEventWithoutClusterInfo(err, mockReporter, metricSetName)

	mockReporter.AssertCalled(t, "Event", mock.MatchedBy(func(event mb.Event) bool {
		errorField, ok := event.RootFields["error"].(mapstr.M)
		require.True(t, ok)
		require.Equal(t, "UNKNOWN_ERROR", auto_ops_testing.GetObjectValue(errorField, "code"))
		require.Equal(t, "test error", auto_ops_testing.GetObjectValue(errorField, "message"))

		urlField, ok := event.RootFields["url"].(mapstr.M)
		require.True(t, ok)
		require.Equal(t, "/", auto_ops_testing.GetObjectValue(urlField, "path"))
		require.Equal(t, "", auto_ops_testing.GetObjectValue(urlField, "query"))

		httpField, ok := event.RootFields["http"].(mapstr.M)
		require.True(t, ok)
		require.Equal(t, http.MethodGet, auto_ops_testing.GetObjectValue(httpField, "request.method"))
		require.Equal(t, 0, auto_ops_testing.GetObjectValue(httpField, "response.status_code"))

		assert.NotEmpty(t, auto_ops_testing.GetObjectValue(event.ModuleFields, "transaction_id"))

		return true
	}))
}

func TestLogAndSendErrorEventWithoutClusterInfoNonDefaultValues(t *testing.T) {
	mockReporter := new(MockReporter)
	mockReporter.On("Event", mock.Anything).Return(true)

	err := &utils.HTTPResponse{
		StatusCode: 500,
		Status:     "HTTP_500",
		Body:       "Internal Server Error",
		Err:        errors.New("server encountered an unexpected condition"),
	}
	metricSetName := "custom_metricset"

	LogAndSendErrorEventWithoutClusterInfo(err, mockReporter, metricSetName)

	mockReporter.AssertCalled(t, "Event", mock.MatchedBy(func(event mb.Event) bool {
		errorField, ok := event.RootFields["error"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "HTTP_500", auto_ops_testing.GetObjectValue(errorField, "code"))
		assert.Equal(t, "server encountered an unexpected condition", auto_ops_testing.GetObjectValue(errorField, "message"))

		urlField, ok := event.RootFields["url"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "/", auto_ops_testing.GetObjectValue(urlField, "path"))
		assert.Equal(t, "", auto_ops_testing.GetObjectValue(urlField, "query"))

		httpField, ok := event.RootFields["http"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, http.MethodGet, auto_ops_testing.GetObjectValue(httpField, "request.method"))
		assert.Equal(t, 500, auto_ops_testing.GetObjectValue(httpField, "response.status_code"))

		assert.NotEmpty(t, auto_ops_testing.GetObjectValue(event.ModuleFields, "transaction_id"))

		return true
	}))
}

func TestLogAndSendErrorEventDefaultValues(t *testing.T) {
	mockReporter := new(MockReporter)
	mockReporter.On("Event", mock.Anything).Return(true)

	err := errors.New("test error")
	clusterInfo := &utils.ClusterInfo{
		ClusterName: "",
		ClusterID:   "test-cluster-id",
		Version: utils.ClusterInfoVersion{
			Number: version.MustNew("8.0.0"),
		},
	}
	metricSetName := "test_metricset"
	path := "/test/path?query=string&other=param"

	LogAndSendErrorEvent(err, clusterInfo, mockReporter, metricSetName, path, "test-transaction-id")

	mockReporter.AssertCalled(t, "Event", mock.MatchedBy(func(event mb.Event) bool {
		errorField, ok := event.RootFields["error"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "UNKNOWN_ERROR", auto_ops_testing.GetObjectValue(errorField, "code"))
		assert.Equal(t, "test error", auto_ops_testing.GetObjectValue(errorField, "message"))

		urlField, ok := event.RootFields["url"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "/test/path", auto_ops_testing.GetObjectValue(urlField, "path"))
		assert.Equal(t, "query=string&other=param", auto_ops_testing.GetObjectValue(urlField, "query"))

		httpField, ok := event.RootFields["http"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, http.MethodGet, auto_ops_testing.GetObjectValue(httpField, "request.method"))
		assert.Equal(t, 0, auto_ops_testing.GetObjectValue(httpField, "response.status_code"))

		assert.Equal(t, "test-transaction-id", auto_ops_testing.GetObjectValue(event.ModuleFields, "transaction_id"))

		return true
	}))
}

func TestLogAndSendErrorEventNonDefaultValues(t *testing.T) {
	mockReporter := new(MockReporter)
	mockReporter.On("Event", mock.Anything).Return(true)

	err := &utils.HTTPResponse{
		StatusCode: 404,
		Status:     "HTTP_404",
		Body:       "Page Not Found",
		Err:        errors.New("error message is passed through"),
	}
	clusterInfo := &utils.ClusterInfo{
		ClusterName: "custom-name",
		ClusterID:   "custom-cluster-id",
		Version: utils.ClusterInfoVersion{
			Number: version.MustNew("8.0.0"),
		},
	}
	metricSetName := "custom_metricset"
	path := "/custom/path?param=value"

	LogAndSendErrorEvent(err, clusterInfo, mockReporter, metricSetName, path, "custom-transaction-id")

	mockReporter.AssertCalled(t, "Event", mock.MatchedBy(func(event mb.Event) bool {
		clusterField, ok := event.ModuleFields["cluster"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "custom-cluster-id", auto_ops_testing.GetObjectValue(clusterField, "id"))
		assert.Equal(t, "custom-name", auto_ops_testing.GetObjectValue(clusterField, "name"))
		assert.Equal(t, "8.0.0", auto_ops_testing.GetObjectValue(clusterField, "version"))

		errorField, ok := event.RootFields["error"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "HTTP_404", auto_ops_testing.GetObjectValue(errorField, "code"))
		assert.Equal(t, "error message is passed through", auto_ops_testing.GetObjectValue(errorField, "message"))

		urlField, ok := event.RootFields["url"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, "/custom/path", auto_ops_testing.GetObjectValue(urlField, "path"))
		assert.Equal(t, "param=value", auto_ops_testing.GetObjectValue(urlField, "query"))

		httpField, ok := event.RootFields["http"].(mapstr.M)
		require.True(t, ok)
		assert.Equal(t, http.MethodGet, auto_ops_testing.GetObjectValue(httpField, "request.method"))
		assert.Equal(t, 404, auto_ops_testing.GetObjectValue(httpField, "response.status_code"))

		assert.Equal(t, "custom-transaction-id", auto_ops_testing.GetObjectValue(event.ModuleFields, "transaction_id"))

		return true
	}))
}
