// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

func TestGetErrorCodeHTTPError(t *testing.T) {
	httpErr := &utils.HTTPResponse{StatusCode: 404}
	err := fmt.Errorf("wrapped error: %w", httpErr)

	result := getErrorCode(err)

	assert.Equal(t, "HTTP_404", result, "Expected HTTP_404 for HTTPResponse with status code 404")
}

func TestGetErrorCodeUnknownError(t *testing.T) {
	err := errors.New("some generic error")

	result := getErrorCode(err)

	assert.Equal(t, "UNKNOWN_ERROR", result, "Expected UNKNOWN_ERROR for non-HTTPResponse")
}

func TestGetErrorCodeNilError(t *testing.T) {
	result := getErrorCode(nil)

	assert.Equal(t, "UNKNOWN_ERROR", result, "Expected UNKNOWN_ERROR for nil error")
}

func TestGetSurfaceErrorWithColon(t *testing.T) {
	err := errors.New("root error: additional context")
	result := getSurfaceError(err)
	assert.Equal(t, "root error", result)
}

func TestGetSurfaceErrorWithoutColon(t *testing.T) {
	err := errors.New("root error")
	result := getSurfaceError(err)
	assert.Equal(t, "root error", result)
}

func TestGetSurfaceErrorNilError(t *testing.T) {
	result := getSurfaceError(nil)
	assert.Equal(t, "", result)
}

func TestGetResourceID(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		expectedValue string
	}{
		{
			name: "DEPLOYMENT_ID is set",
			envVars: map[string]string{
				"DEPLOYMENT_ID": "deployment-123",
			},
			expectedValue: "deployment-123",
		},
		{
			name: "PROJECT_ID is set",
			envVars: map[string]string{
				"PROJECT_ID": "project-456",
			},
			expectedValue: "project-456",
		},
		{
			name: "RESOURCE_ID is set",
			envVars: map[string]string{
				"RESOURCE_ID": "resource-789",
			},
			expectedValue: "resource-789",
		},
		{
			name: "No environment variables are set",
			envVars: map[string]string{
				"DEPLOYMENT_ID": "",
				"PROJECT_ID":    "",
				"RESOURCE_ID":   "",
			},
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Call the function
			result := getResourceID()

			// Assert the result
			assert.Equal(t, tt.expectedValue, result)

			// Clean up environment variables
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}
