// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package ccm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to get a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

func TestGetCloudConnectedModeAPIURL(t *testing.T) {
	testCases := []struct {
		name        string
		envVarValue *string // Pointer to distinguish "set to empty" vs "not set by this test"
		expectedURL string
	}{
		{
			name:        "env var set to custom value",
			envVarValue: stringPtr("http://custom.api.url.from.test"),
			expectedURL: "http://custom.api.url.from.test",
		},
		{
			name:        "env var set to empty string",
			envVarValue: stringPtr(""),                        // Explicitly set to empty
			expectedURL: DEFAULT_CLOUD_CONNECTED_MODE_API_URL, // Should fall back to default
		},
		{
			name:        "env var not set by this test (relies on clean state or t.Setenv restoration)",
			envVarValue: nil,                                  // Indicates t.Setenv should not be called for this key in this subtest
			expectedURL: DEFAULT_CLOUD_CONNECTED_MODE_API_URL, // Should fall back to default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVarValue != nil {
				t.Setenv(CLOUD_CONNECTED_MODE_API_URL_NAME, *tc.envVarValue)
			}

			actualURL := getCloudConnectedModeAPIURL()
			assert.Equal(t, tc.expectedURL, actualURL)
		})
	}
}
