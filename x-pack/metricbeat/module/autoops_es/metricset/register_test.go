// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package metricset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/elastic-agent-libs/version"
)

func writeCloudConnectedResource(w http.ResponseWriter, id string, supported bool, minimumStackVersion string, validLicenses []string) error {
	return json.NewEncoder(w).Encode(cloudConnectedResource{
		ID: id,
		Services: cloudConnectedServices{
			AutoOps: cloudConnectedService{
				Support: cloudConnectedServiceSupport{
					Supported:           supported,
					MinimumStackVersion: minimumStackVersion,
					ValidLicenseTypes:   validLicenses,
				},
			},
		},
	})
}

func TestRegisterCloudConnectedCluster(t *testing.T) {
	clusterInfo := &utils.ClusterInfo{
		ClusterID:   "test-cluster-id",
		ClusterName: "test-cluster-name",
		Version: utils.ClusterInfoVersion{
			Number: version.MustNew("8.0.0"),
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
				assert.NoError(t, writeCloudConnectedResource(w, "registered-cluster-id", true, "123.4.5", []string{}))
			},
			expectError:   false,
			expectedResID: "registered-cluster-id",
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
				assert.NoError(t, writeCloudConnectedResource(w, "created-cluster-id", true, "123.4.5", []string{"unchecked"}))
			},
			expectError:   false,
			expectedResID: "created-cluster-id",
		},
		{
			name:   "success but unsupported because of minimum stack version",
			apiKey: "test-api-key-201",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key-201", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				assert.NoError(t, writeCloudConnectedResource(w, "unsupported-version", false, "9.0.0", []string{lic.Type}))
			},
			expectError: true,
		},
		{
			name:   "success but unsupported because of license type",
			apiKey: "test-api-key-201",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key-201", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				assert.NoError(t, writeCloudConnectedResource(w, "unsupported-license", false, "9.0.0", []string{"enterprise"}))
			},
			expectError: true,
		},
		{
			name:   "success but unsupported for unknown reason with malformed version",
			apiKey: "test-api-key-201",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key-201", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				assert.NoError(t, writeCloudConnectedResource(w, "unsupported-cluster-id", false, "xyz", []string{lic.Type}))
			},
			expectError: true,
		},
		{
			name:   "success but unsupported for unknown reason",
			apiKey: "test-api-key-201",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key-201", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				assert.NoError(t, writeCloudConnectedResource(w, "unsupported-cluster-id", false, "7.17.0", []string{lic.Type}))
			},
			expectError: true,
		},
		{
			name:   "api error",
			apiKey: "test-api-key",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(`{"errors":[{"code":"service_unavailable","message":"internal server error"}]`))
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
			name:   "'success' but missing id in response",
			apiKey: "test-api-key-no-id-resp",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"message": "Request processed, but no specific resource ID generated"}`))
				assert.NoError(t, err)
			},
			expectError: true,
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(utils.ClearResourceID)

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
				assert.NotEmpty(t, utils.GetResourceID(), "Resource ID should not be empty")
				assert.Equal(t, tc.expectedResID, utils.GetResourceID(), "Resource ID mismatch")
			}
		})
	}
}

func TestMaybeRegisterCloudConnectedCluster(t *testing.T) {
	t.Cleanup(utils.ClearResourceID)
	t.Cleanup(func() {
		checkedCloudConnectedMode = false
	})

	checkedCloudConnectedMode = true // do NOT lookup anything

	noDisplayName := &clusterSettingsResponse{}

	clusterSettings := func(persistent string, transient string) *clusterSettingsResponse {
		persistentSettings := clusterSettingsDisplayName{}
		transientSettings := clusterSettingsDisplayName{}

		// even if they're blank, that matches expected behavior
		persistentSettings.Cluster.Metadata.DisplayName = persistent
		transientSettings.Cluster.Metadata.DisplayName = transient

		return &clusterSettingsResponse{
			Persistent: persistentSettings,
			Transient:  transientSettings,
		}
	}

	explicitlyBlankDisplayName := clusterSettings("", "")

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
		name                      string
		clusterInfoStatusCode     int
		clusterInfo               *utils.ClusterInfo
		clusterSettingsStatusCode int
		clusterSettings           *clusterSettingsResponse
		licenseStatusCode         int
		license                   *licenseWrapper
		expectError               bool
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
			name:                      "failed license request",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("8.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         500,
			expectError:               true,
		},
		{
			name:                      "failed license request (7.x)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("7.17.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         500,
			expectError:               true,
		},
		{
			name:                      "inactive license",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("8.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("enterprise", "inactive"),
			expectError:               true,
		},
		{
			name:                      "inactive license (7.x)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("7.17.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("enterprise", "inactive"),
			expectError:               true,
		},
		{
			name:                      "success for basic license",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("8.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("basic", "active"),
			expectError:               false,
		},
		{
			name:                      "success for basic license (7.x)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("7.17.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("basic", "active"),
			expectError:               false,
		},
		{
			name:                      "success for platinum license",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("8.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("basic", "active"),
			expectError:               false,
		},
		{
			name:                      "success for platinum license (7.x)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("7.17.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("basic", "active"),
			expectError:               false,
		},
		{
			name:                      "success for enterprise",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("8.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           explicitlyBlankDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("enterprise", "active"),
			expectError:               false,
		},
		{
			name:                      "success for trial",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("8.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success for enterprise (7.x)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("7.17.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           explicitlyBlankDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("enterprise", "active"),
			expectError:               false,
		},
		{
			name:                      "success for trial (7.x)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("7.17.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           noDisplayName,
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success with custom persistent display name",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("9.0.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           clusterSettings("custom-persistent", ""),
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success with custom transient display name",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("9.1.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           clusterSettings("", "custom-transient"),
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success with custom persistent and transient display name",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("9.2.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           clusterSettings("custom-persistent", "custom-transient"),
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success with blank persistent and transient display name",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("9.2.0"),
			clusterSettingsStatusCode: 200,
			clusterSettings:           clusterSettings("", ""),
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success with display name failure (4xx)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("9.0.0"),
			clusterSettingsStatusCode: 401, // fail to get settings because of auth error
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
		{
			name:                      "success with display name failure (5xx)",
			clusterInfoStatusCode:     200,
			clusterInfo:               clusterInfoForVersion("9.0.0"),
			clusterSettingsStatusCode: 500, // unknown server error
			licenseStatusCode:         200,
			license:                   licenseForType("trial", "active"),
			expectError:               false,
		},
	}

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
					case clusterSettingsPath: // Cluster Settings
						w.WriteHeader(tc.clusterSettingsStatusCode)
						w.Header().Set("Content-Type", "application/json")

						if tc.clusterSettings != nil {
							if tc.clusterSettings == noDisplayName {
								// write empty object that behaves like if these values are truly unset
								_, err := w.Write([]byte(`{}`))
								assert.NoError(t, err)
							} else {
								err := json.NewEncoder(w).Encode(tc.clusterSettings)
								assert.NoError(t, err)
							}
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

				var cloudConnectedCluster cloudConnectedCluster

				if err := json.NewDecoder(r.Body).Decode(&cloudConnectedCluster); err != nil {
					http.Error(w, "cannot read body", http.StatusInternalServerError)
					return
				}

				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/cloud-connected/clusters", r.URL.Path)
				assert.Equal(t, "ApiKey test-api-key", r.Header.Get("Authorization"))

				expectedName := "my-cluster" // default unless overridden by settings

				if tc.clusterSettings != nil {
					if tc.clusterSettings.Transient.Cluster.Metadata.DisplayName != "" {
						expectedName = tc.clusterSettings.Transient.Cluster.Metadata.DisplayName
					} else if tc.clusterSettings.Persistent.Cluster.Metadata.DisplayName != "" {
						expectedName = tc.clusterSettings.Persistent.Cluster.Metadata.DisplayName
					}
				}

				assert.Equal(t, expectedName, cloudConnectedCluster.Cluster.Name)
				assert.Equal(t, "id123", cloudConnectedCluster.Cluster.ID)
				assert.Equal(t, tc.clusterInfo.Version.Number.String(), cloudConnectedCluster.Cluster.Version)
				assert.Equal(t, "id456", cloudConnectedCluster.License.UID)
				assert.Equal(t, tc.license.License.Type, cloudConnectedCluster.License.Type)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				_, err := w.Write([]byte(`{"id": "registered-cluster-id", "services": {"auto_ops": {"support": {"supported": true}}}}`))
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

// ccmFixture serves a self-managed cluster that can be registered, and counts the registrations that the CCM API gets.
type ccmFixture struct {
	esURL         string
	registrations *atomic.Int32
}

func newCCMFixture(t *testing.T, ccmStatus int) ccmFixture {
	t.Helper()

	t.Cleanup(utils.ClearResourceID)
	utils.ClearResourceID()

	for _, name := range []string{"AUTOOPS_DEPLOYMENT_ID", "AUTOOPS_PROJECT_ID", "AUTOOPS_RESOURCE_ID"} {
		t.Setenv(name, "")
	}

	previous := checkedCloudConnectedMode
	checkedCloudConnectedMode = false

	t.Cleanup(func() { checkedCloudConnectedMode = previous })

	registrations := &atomic.Int32{}

	ccm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		registrations.Add(1)

		if ccmStatus != http.StatusOK {
			w.WriteHeader(ccmStatus)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = writeCloudConnectedResource(w, "registered-cluster-id", true, "8.0.0", []string{"enterprise"})
	}))
	t.Cleanup(ccm.Close)

	es := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`{"cluster_name":"c","cluster_uuid":"abc","version":{"number":"8.15.3","build_flavor":"default"}}`))
		case "/_cluster/settings":
			_, _ = w.Write([]byte(`{}`))
		case "/_license":
			_, _ = w.Write([]byte(`{"license":{"uid":"u","type":"enterprise","status":"active"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(es.Close)

	t.Setenv(CLOUD_CONNECTED_MODE_API_KEY_NAME, "test-api-key")
	t.Setenv(CLOUD_CONNECTED_MODE_API_URL_NAME, ccm.URL)

	return ccmFixture{esURL: es.URL, registrations: registrations}
}

// newMetricSets creates the metricsets in the order that they are given, which are either "standard" or "serverless".
func (f ccmFixture) newMetricSets(t *testing.T, kinds ...string) []error {
	t.Helper()

	var errs []error

	for _, kind := range kinds {
		registry := mb.NewRegister()
		registry.MustAddMetricSet(MODULE_NAME, "ccm_test", func(base mb.BaseMetricSet) (mb.MetricSet, error) {
			if kind == "serverless" {
				return newServerlessAutoOpsMetricSet(base, "/", RequestOptions{}, func(_ mb.ReporterV2, _ *utils.ClusterInfo, _ *any) error { return nil })
			}

			return newAutoOpsMetricSet(base, "/", func(_ mb.ReporterV2, _ *utils.ClusterInfo, _ *any) error { return nil }, nil)
		},
			mb.WithHostParser(elasticsearch.HostParser),
			mb.DefaultMetricSet(),
		)

		// not mbtest.NewMetricSetsWithRegistry, which fails the test when the metricset cannot be created
		c, err := conf.NewConfigFrom(map[string]any{
			"module":     "autoops_es",
			"metricsets": []string{"ccm_test"},
			"hosts":      []string{f.esURL},
		})
		require.NoError(t, err)

		_, _, err = mb.NewModule(c, registry, beat.Info{Paths: paths.New(), UserAgent: "test", Logger: logptest.NewTestingLogger(t, "")})

		errs = append(errs, err)
	}

	return errs
}

func TestRegisterCloudConnectedClusterOnce(t *testing.T) {
	tests := []struct {
		name              string
		kinds             []string
		ccmStatus         int
		expectRegistered  int32
		expectResourceID  string
		expectErrorOnEach []bool
	}{
		{"standard registers once", []string{"standard", "standard", "standard"}, http.StatusOK, 1, "registered-cluster-id", []bool{false, false, false}},
		{"only serverless never registers", []string{"serverless", "serverless"}, http.StatusOK, 0, "", []bool{false, false}},
		// a failure is returned once, then the flag is set, so later metricsets are not blocked (as before the change)
		{"failure is returned once", []string{"standard", "standard"}, http.StatusInternalServerError, 1, "", []bool{true, false}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCCMFixture(t, test.ccmStatus)

			errs := fixture.newMetricSets(t, test.kinds...)

			for i, err := range errs {
				if test.expectErrorOnEach[i] {
					assert.ErrorContains(t, err, "failed to register Cloud Connected cluster", "metricset %d should fail with the wrapped error", i)
				} else {
					assert.NoError(t, err, "metricset %d should be created", i)
				}
			}

			assert.Equal(t, test.expectRegistered, fixture.registrations.Load(), "registrations that the CCM API got")
			assert.Equal(t, test.expectResourceID, utils.GetResourceID(), "resource ID")
		})
	}
}

func TestGetAndSetResourceIDFromTheEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		existing string
		expected string
	}{
		{"nothing set", nil, "", ""},
		{"project", map[string]string{"AUTOOPS_PROJECT_ID": "project"}, "", "project"},
		{"resource", map[string]string{"AUTOOPS_RESOURCE_ID": "resource"}, "", "resource"},
		{"deployment wins over project", map[string]string{"AUTOOPS_DEPLOYMENT_ID": "deployment", "AUTOOPS_PROJECT_ID": "project"}, "", "deployment"},
		{"project wins over resource", map[string]string{"AUTOOPS_PROJECT_ID": "project", "AUTOOPS_RESOURCE_ID": "resource"}, "", "project"},
		{"existing is not overwritten", map[string]string{"AUTOOPS_PROJECT_ID": "project"}, "existing", "existing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Cleanup(utils.ClearResourceID)
			utils.ClearResourceID()

			for _, name := range []string{"AUTOOPS_DEPLOYMENT_ID", "AUTOOPS_PROJECT_ID", "AUTOOPS_RESOURCE_ID"} {
				t.Setenv(name, test.env[name])
			}

			if test.existing != "" {
				utils.SetResourceID(test.existing)
			}

			assert.Equal(t, test.expected, utils.GetAndSetResourceID(), "returned resource ID")
			assert.Equal(t, test.expected, utils.GetResourceID(), "stored resource ID")
		})
	}
}
