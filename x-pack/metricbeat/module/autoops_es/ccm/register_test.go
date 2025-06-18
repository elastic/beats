// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package ccm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/version"
	"github.com/stretchr/testify/assert"
)

func TestRegisterCloudConnectedCluster(t *testing.T) {
	v := version.MustNew("8.0.0")
	clusterInfo := &utils.ClusterInfo{
		ClusterID:   "test-cluster-id",
		ClusterName: "test-cluster-name",
		Version: utils.ClusterInfoVersion{
			Number: v,
		},
	}
	lic := &license{
		UID:  "test-license-uid",
		Type: "platinum",
	}

	testCases := []struct {
		name          string
		apiKey        string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		expectError   bool
		expectedResID string
	}{
		{
			name:   "success",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Log request body
				bodyBytes, bodyErr := io.ReadAll(r.Body)
				if bodyErr != nil {
					http.Error(w, "cannot read body", http.StatusInternalServerError)
					return
				}
				r.Body.Close()                                    // Close the original body
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body for further reading

				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key", r.Header.Get("Authorization"))
				var payload map[string]any
				err := json.NewDecoder(r.Body).Decode(&payload)
				if err != nil {
					http.Error(w, "cannot decode payload", http.StatusBadRequest)
					return
				}
				if payload == nil {
					http.Error(w, "decoded payload is nil", http.StatusBadRequest)
					return
				}

				clusterPayload, ok := payload["self_managed_cluster"].(map[string]any)
				if !ok {
					http.Error(w, "invalid self_managed_cluster payload", http.StatusBadRequest)
					return
				}
				assert.Equal(t, "test-cluster-id", clusterPayload["id"])

				licensePayload, ok := payload["license"].(map[string]any)
				if !ok {
					http.Error(w, "invalid license payload", http.StatusBadRequest)
					return
				}
				assert.Equal(t, "test-license-uid", licensePayload["uid"])

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err = w.Write([]byte(`{"id": "registered-cluster-id"}`))
				assert.NoError(t, err)
			},
			expectError:   false,
			expectedResID: "registered-cluster-id",
		},
		{
			name:   "api error",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(`{"error": "internal server error"}`))
				assert.NoError(t, err)
			},
			expectError: true,
		},
		{
			name:   "malformed json response",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"id": "registered-cluster-id"`)) // Malformed JSON
				assert.NoError(t, err)
			},
			expectError: true,
		},
		{
			name:          "client error no server",
			apiKey:        "test-api-key",
			serverHandler: nil, // No server will be running for this case, effectively
			expectError:   true,
		},
		{
			name:   "success with 201 created",
			apiKey: "test-api-key-201",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key-201", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, err := w.Write([]byte(`{"id": "created-cluster-id"}`))
				assert.NoError(t, err)
			},
			expectError:   false,
			expectedResID: "created-cluster-id",
		},
		{
			name:   "api error 401 unauthorized",
			apiKey: "test-api-key-unauth",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "ApiKey test-api-key-unauth", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"error": "access denied"}`))
				assert.NoError(t, err)
			},
			expectError: true,
		},
		{
			name:   "success but missing id in response",
			apiKey: "test-api-key-no-id-resp",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"message": "Request processed, but no specific resource ID generated"}`))
				assert.NoError(t, err)
			},
			expectError:   false,
			expectedResID: "",
		},
		{
			name:   "empty api key",
			apiKey: "",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				assert.Equal(t, "ApiKey", authHeader, "Authorization header for empty API key mismatch")
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte(`{"error": "API key cannot be empty"}`))
				assert.NoError(t, err)
			},
			expectError: true,
		},
	}

	t.Cleanup(utils.ClearResourceID)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			utils.ClearResourceID()

			var server *httptest.Server

			if tc.serverHandler != nil {
				server = httptest.NewServer(http.HandlerFunc(tc.serverHandler))
				t.Setenv(CLOUD_CONNECTED_MODE_API_URL_NAME, server.URL)
				t.Cleanup(server.Close)
			} else {
				// For the client error case, point to a non-existent server
				t.Setenv(CLOUD_CONNECTED_MODE_API_URL_NAME, "http://localhost:12345")
			}

			err := registerCloudConnectedCluster(tc.apiKey, clusterInfo, lic)

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, utils.GetResourceID(), "Resource ID should not be set on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResID, utils.GetResourceID(), "Resource ID mismatch")
			}
		})
	}
}
