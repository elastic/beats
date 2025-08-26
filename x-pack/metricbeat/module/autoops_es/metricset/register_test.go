// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package metricset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/version"
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

			err := registerCloudConnectedCluster(tc.apiKey, clusterInfo, lic, logptest.NewTestingLogger(t, tc.name))

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

func TestMaybeRegisterCloudConnectedCluster(t *testing.T) {
	checkedCloudConnectedMode = true // do NOT lookup anything
	clusterInfoForVersion := func(v string) *utils.ClusterInfo {
		return &utils.ClusterInfo{
			ClusterName: "my-cluster",
			ClusterID:   "id123",
			Version: utils.ClusterInfoVersion{
				Number: version.MustNew(v),
			},
		}
	}
	licenseForType := func(licenseType string, status string) *licenseWrapper {
		return &licenseWrapper{
			License: license{
				UID:    "id456",
				Status: status,
				Type:   licenseType,
			},
		}
	}

	testCases := []struct {
		name                  string
		clusterInfoStatusCode int
		clusterInfo           *utils.ClusterInfo
		licenseStatusCode     int
		license               *licenseWrapper
		expectError           bool
	}{
		{
			name:        "client error no ES",
			expectError: true,
		},
		{
			name:                  "invalid ES auth",
			clusterInfoStatusCode: 401,
			expectError:           true,
		},
		{
			name:                  "failed cluster info request",
			clusterInfoStatusCode: 500,
			expectError:           true,
		},
		{
			name:                  "failed license request",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("8.0.0"),
			licenseStatusCode:     500,
			expectError:           true,
		},
		{
			name:                  "failed license request (7.x)",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("7.17.0"),
			licenseStatusCode:     500,
			expectError:           true,
		},
		{
			name:                  "inactive license",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("8.0.0"),
			licenseStatusCode:     200,
			license:               licenseForType("enterprise", "inactive"),
			expectError:           true,
		},
		{
			name:                  "inactive license (7.x)",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("7.17.0"),
			licenseStatusCode:     200,
			license:               licenseForType("enterprise", "inactive"),
			expectError:           true,
		},
		{
			name:                  "unsupported license",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("8.0.0"),
			licenseStatusCode:     200,
			license:               licenseForType("basic", "active"),
			expectError:           true,
		},
		{
			name:                  "unsupported license (7.x)",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("7.17.0"),
			licenseStatusCode:     200,
			license:               licenseForType("basic", "active"),
			expectError:           true,
		},
		{
			name:                  "success for enterprise",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("8.0.0"),
			licenseStatusCode:     200,
			license:               licenseForType("enterprise", "active"),
			expectError:           false,
		},
		{
			name:                  "success for trial",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("8.0.0"),
			licenseStatusCode:     200,
			license:               licenseForType("trial", "active"),
			expectError:           false,
		},
		{
			name:                  "success for enterprise (7.x)",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("7.17.0"),
			licenseStatusCode:     200,
			license:               licenseForType("enterprise", "active"),
			expectError:           false,
		},
		{
			name:                  "success for trial (7.x)",
			clusterInfoStatusCode: 200,
			clusterInfo:           clusterInfoForVersion("7.17.0"),
			licenseStatusCode:     200,
			license:               licenseForType("trial", "active"),
			expectError:           false,
		},
	}

	t.Cleanup(utils.ClearResourceID)
	t.Cleanup(func() {
		checkedCloudConnectedMode = false
	})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			utils.ClearResourceID()

			// mocked ES responses
			var esServer *httptest.Server

			// if the server does not exist, then we are testing that it reacts properly to no server existing
			if tc.clusterInfoStatusCode != 0 {
				esServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.RequestURI {
					case "/": // ClusterInfo
						w.WriteHeader(tc.clusterInfoStatusCode)
						w.Header().Set("Content-Type", "application/json")
						if tc.clusterInfo != nil {
							fmt.Fprintf(w, `{"cluster_name": "%s", "cluster_uuid": "%s", "version": { "number": "%s" }}`, tc.clusterInfo.ClusterName, tc.clusterInfo.ClusterID, tc.clusterInfo.Version.Number)
						}
					case licensePath: // License for non-7.x versions
						if tc.clusterInfo == nil || tc.clusterInfo.Version.Number.Major == 7 {
							t.Fatalf("Unexpected request to %v", r.RequestURI)
						}

						w.WriteHeader(tc.licenseStatusCode)
						w.Header().Set("Content-Type", "application/json")

						if tc.license != nil {
							err := json.NewEncoder(w).Encode(tc.license)
							assert.NoError(t, err)
						}
					case licensePathV7: // License for 7.x versions
						if tc.clusterInfo == nil || tc.clusterInfo.Version.Number.Major != 7 {
							t.Fatalf("Unexpected request to %v", r.RequestURI)
						}

						w.WriteHeader(tc.licenseStatusCode)
						w.Header().Set("Content-Type", "application/json")

						if tc.license != nil {
							err := json.NewEncoder(w).Encode(tc.license)
							assert.NoError(t, err)
						}
					default:
						t.Fatalf("Unrecognized request %v", r.RequestURI)
					}
				}))
				t.Cleanup(esServer.Close)
			}

			ccmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer r.Body.Close()

				if _, err := io.ReadAll(r.Body); err != nil {
					http.Error(w, "cannot read body", http.StatusInternalServerError)
					return
				}

				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key", r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, err := w.Write([]byte(`{"id": "registered-cluster-id"}`))
				assert.NoError(t, err)
			}))

			t.Setenv(CLOUD_CONNECTED_MODE_API_KEY_NAME, "test-api-key")
			t.Setenv(CLOUD_CONNECTED_MODE_API_URL_NAME, ccmServer.URL)
			t.Cleanup(ccmServer.Close)

			config := map[string]any{
				"module":     "autoops_es",
				"metricsets": []string{"mock_metricset"},
				"hosts":      []string{"http://example.invalid:9200"}, // https://www.rfc-editor.org/rfc/rfc2606
			}

			if esServer != nil {
				config["hosts"] = []string{esServer.URL}
			}

			mockRegistry := mb.NewRegister()
			mockRegistry.MustAddMetricSet(MODULE_NAME, "mock_metricset", func(base mb.BaseMetricSet) (mb.MetricSet, error) {
				return newAutoOpsMetricSet(base, "/", func(_ mb.ReporterV2, _ *utils.ClusterInfo, _ *any) error { return nil }, nil)
			},
				mb.WithHostParser(elasticsearch.HostParser),
				mb.DefaultMetricSet(),
			)

			m := mbtest.NewMetricSetWithRegistry(t, config, mockRegistry).(*AutoOpsMetricSet[any])

			err := maybeRegisterCloudConnectedCluster(m.MetricSet, func(ms *elasticsearch.MetricSet) (*utils.ClusterInfo, error) {
				return utils.FetchAPIData[utils.ClusterInfo](ms, "/")
			})

			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, utils.GetResourceID(), "Resource ID should not be set on error")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "registered-cluster-id", utils.GetResourceID(), "Resource ID mismatch")
			}
		})
	}
}
