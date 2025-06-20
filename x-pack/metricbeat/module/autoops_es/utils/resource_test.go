// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetResourceID(t *testing.T) {
	t.Cleanup(ClearResourceID)

	// Initial state
	assert.Equal(t, "", GetResourceID(), "Initial resourceID should be empty")

	// After SetResourceID
	SetResourceID("test-id")
	assert.Equal(t, "test-id", GetResourceID(), "resourceID should be 'test-id' after setting")
}

func TestSetResourceID(t *testing.T) {
	t.Cleanup(ClearResourceID)

	SetResourceID("another-id")
	assert.Equal(t, "another-id", resourceID, "Global resourceID variable should be 'another-id'") // Test internal state
	assert.Equal(t, "another-id", GetResourceID(), "GetResourceID should return 'another-id'")
}

func TestClearResourceID(t *testing.T) {
	t.Cleanup(ClearResourceID)

	SetResourceID("id-to-clear")
	assert.Equal(t, "id-to-clear", GetResourceID(), "resourceID should be 'id-to-clear' before clearing")

	ClearResourceID()
	assert.Equal(t, "", GetResourceID(), "resourceID should be empty after clearing")
	assert.Equal(t, "", resourceID, "Global resourceID variable should be empty after clearing") // Test internal state
}

func TestGetAndSetResourceID(t *testing.T) {
	// Overall cleanup for the resourceID global variable after all subtests in TestGetAndSetResourceID are done.
	t.Cleanup(ClearResourceID)

	tests := []struct {
		name               string
		initialResourceID  string
		envVars            map[string]string
		expectedResourceID string // Expected return from GetAndSetResourceID
		expectedFinalID    string // Expected value of global resourceID after the call
	}{
		{
			name:               "resourceID already set",
			initialResourceID:  "pre-existing-id",
			envVars:            map[string]string{"DEPLOYMENT_ID": "env-dep-id"}, // This env var should not be used
			expectedResourceID: "pre-existing-id",
			expectedFinalID:    "pre-existing-id",
		},
		{
			name:               "resourceID empty, DEPLOYMENT_ID set",
			initialResourceID:  "",
			envVars:            map[string]string{"DEPLOYMENT_ID": "env-dep-id"},
			expectedResourceID: "env-dep-id",
			expectedFinalID:    "env-dep-id",
		},
		{
			name:               "resourceID empty, DEPLOYMENT_ID empty, PROJECT_ID set",
			initialResourceID:  "",
			envVars:            map[string]string{"PROJECT_ID": "env-proj-id"},
			expectedResourceID: "env-proj-id",
			expectedFinalID:    "env-proj-id",
		},
		{
			name:               "resourceID empty, DEPLOYMENT_ID empty, PROJECT_ID empty, RESOURCE_ID (env) set",
			initialResourceID:  "",
			envVars:            map[string]string{"RESOURCE_ID": "env-res-id"},
			expectedResourceID: "env-res-id",
			expectedFinalID:    "env-res-id",
		},
		{
			name:               "resourceID empty, no relevant env vars set",
			initialResourceID:  "",
			envVars:            map[string]string{"OTHER_VAR": "other_value"}, // Irrelevant var
			expectedResourceID: "",
			expectedFinalID:    "",
		},
		{
			name:               "precedence: DEPLOYMENT_ID > PROJECT_ID > RESOURCE_ID (env)",
			initialResourceID:  "",
			envVars:            map[string]string{"DEPLOYMENT_ID": "env-dep-id-prec", "PROJECT_ID": "env-proj-id-prec", "RESOURCE_ID": "env-res-id-prec"},
			expectedResourceID: "env-dep-id-prec",
			expectedFinalID:    "env-dep-id-prec",
		},
		{
			name:               "precedence: PROJECT_ID > RESOURCE_ID (env), DEPLOYMENT_ID empty",
			initialResourceID:  "",
			envVars:            map[string]string{"DEPLOYMENT_ID": "", "PROJECT_ID": "env-proj-id-prec2", "RESOURCE_ID": "env-res-id-prec2"},
			expectedResourceID: "env-proj-id-prec2",
			expectedFinalID:    "env-proj-id-prec2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global resourceID state for each sub-test
			ClearResourceID()

			if tt.initialResourceID != "" {
				SetResourceID(tt.initialResourceID)
			}

			// Set environment variables for this specific sub-test.
			// t.Setenv handles cleanup (restoring original value) for each variable at the end of the sub-test.
			// Ensure all relevant keys are explicitly set (even to "") for the test's context.
			relevantKeys := []string{"DEPLOYMENT_ID", "PROJECT_ID", "RESOURCE_ID"}
			for _, key := range relevantKeys {
				if val, ok := tt.envVars[key]; ok {
					t.Setenv(key, val)
				} else {
					// If not specified in test case, ensure it's empty/unset for the test's context.
					// os.Getenv will return "" for a variable set to "".
					t.Setenv(key, "")
				}
			}

			actualID := GetAndSetResourceID()
			assert.Equal(t, tt.expectedResourceID, actualID, "GetAndSetResourceID returned unexpected value")
			assert.Equal(t, tt.expectedFinalID, GetResourceID(), "Global resourceID has unexpected value after GetAndSetResourceID")
		})
	}
}
