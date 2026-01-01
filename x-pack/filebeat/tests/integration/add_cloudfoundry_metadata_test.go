// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

// mockCFAPIServer creates a mock Cloud Foundry API server that handles:
// - /v2/info: Returns API info with UAA endpoint
// - /oauth/token: Returns a mock OAuth token
// - /v2/apps/{guid}: Returns app metadata with inline space/org (cfclient format)
type mockCFAPIServer struct {
	server *httptest.Server
	apps   map[string]mockCFApp
}

type mockCFApp struct {
	GUID      string
	Name      string
	SpaceGUID string
	SpaceName string
	OrgGUID   string
	OrgName   string
}

func newMockCFAPIServer(apps []mockCFApp) *mockCFAPIServer {
	m := &mockCFAPIServer{
		apps: make(map[string]mockCFApp),
	}
	for _, app := range apps {
		m.apps[app.GUID] = app
	}

	mux := http.NewServeMux()

	// V2 Info endpoint - used by cfclient to discover other endpoints
	mux.HandleFunc("/v2/info", func(w http.ResponseWriter, r *http.Request) {
		serverURL := "http://" + r.Host
		info := map[string]any{
			"name":                     "mock-cf",
			"build":                    "mock-1.0.0",
			"support":                  serverURL,
			"version":                  2,
			"authorization_endpoint":   serverURL,
			"token_endpoint":           serverURL,
			"doppler_logging_endpoint": "wss://" + r.Host,
			"routing_endpoint":         serverURL + "/routing",
			"logging_endpoint":         "wss://" + r.Host,
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncode(w, info)
	})

	// OAuth token endpoint
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		token := map[string]any{
			"access_token":  "mock-access-token-" + uuid.Must(uuid.NewV4()).String(),
			"token_type":    "bearer",
			"expires_in":    3600,
			"refresh_token": "mock-refresh-token",
			"scope":         "cloud_controller.admin_read_only doppler.firehose",
			"jti":           uuid.Must(uuid.NewV4()).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncode(w, token)
	})

	// V2 Apps endpoint - cfclient uses ?inline-relations-depth=2 to get space and org inline
	mux.HandleFunc("/v2/apps/", func(w http.ResponseWriter, r *http.Request) {
		// Extract GUID from path /v2/apps/{guid}
		path := r.URL.Path
		guid := filepath.Base(path)

		app, ok := m.apps[guid]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			mustEncode(w, map[string]any{
				"error_code":  "CF-AppNotFound",
				"code":        100004,
				"description": "The app could not be found: " + guid,
			})
			return
		}

		// cfclient expects inline space and org data when inline-relations-depth=2
		response := map[string]any{
			"metadata": map[string]any{
				"guid":       app.GUID,
				"created_at": "2023-01-01T00:00:00Z",
				"updated_at": "2023-01-01T00:00:00Z",
			},
			"entity": map[string]any{
				"name":       app.Name,
				"space_guid": app.SpaceGUID,
				"space_url":  "/v2/spaces/" + app.SpaceGUID,
				// Inline space data (for inline-relations-depth >= 1)
				"space": map[string]any{
					"metadata": map[string]any{
						"guid":       app.SpaceGUID,
						"created_at": "2023-01-01T00:00:00Z",
						"updated_at": "2023-01-01T00:00:00Z",
					},
					"entity": map[string]any{
						"name":              app.SpaceName,
						"organization_guid": app.OrgGUID,
						"organization_url":  "/v2/organizations/" + app.OrgGUID,
						// Inline org data (for inline-relations-depth >= 2)
						"organization": map[string]any{
							"metadata": map[string]any{
								"guid":       app.OrgGUID,
								"created_at": "2023-01-01T00:00:00Z",
								"updated_at": "2023-01-01T00:00:00Z",
							},
							"entity": map[string]any{
								"name": app.OrgName,
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncode(w, response)
	})

	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockCFAPIServer) URL() string {
	return m.server.URL
}

func (m *mockCFAPIServer) Close() {
	m.server.Close()
}

// CFEvent represents the expected structure of enriched CF events
type CFEvent struct {
	Message      string `json:"message"`
	CloudFoundry struct {
		App struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"app"`
		Space struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"space"`
		Org struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"org"`
	} `json:"cloudfoundry"`
	Input struct {
		Type string `json:"type"`
	} `json:"input"`
}

func TestAddCloudFoundryMetadataProcessor(t *testing.T) {
	// Create test apps
	testApps := []mockCFApp{
		{
			GUID:      "app-guid-1",
			Name:      "web-frontend",
			SpaceGUID: "space-guid-1",
			SpaceName: "production",
			OrgGUID:   "org-guid-1",
			OrgName:   "acme-corp",
		},
		{
			GUID:      "app-guid-2",
			Name:      "api-backend",
			SpaceGUID: "space-guid-1",
			SpaceName: "production",
			OrgGUID:   "org-guid-1",
			OrgName:   "acme-corp",
		},
		{
			GUID:      "app-guid-3",
			Name:      "worker-service",
			SpaceGUID: "space-guid-2",
			SpaceName: "staging",
			OrgGUID:   "org-guid-1",
			OrgName:   "acme-corp",
		},
	}

	// Start mock CF API
	mockAPI := newMockCFAPIServer(testApps)
	defer mockAPI.Close()

	t.Logf("Mock CF API running at %s", mockAPI.URL())

	// Filebeat config template with add_cloudfoundry_metadata processor
	var tmplCfg = `
filebeat.inputs:
  - type: filestream
    id: cf-logs
    enabled: true
    paths:
      - {{.log_path}}
    parsers:
      - ndjson:
          keys_under_root: true
          add_error_key: true

processors:
  # Map app_guid to cloudfoundry.app.id (what the processor expects)
  - copy_fields:
      fields:
        - from: app_guid
          to: cloudfoundry.app.id
      fail_on_error: false
      ignore_missing: true

  # Apply the CF metadata processor
  - add_cloudfoundry_metadata:
      api_address: {{.api_address}}
      client_id: test-client
      client_secret: test-secret
      ssl:
        verification_mode: none

  # Clean up temporary field
  - drop_fields:
      fields: [app_guid]
      ignore_missing: true

path.home: {{.path_home}}

output.file:
  path: ${path.home}
  filename: output-file
  rotate_every_kb: 10000

logging.level: debug
`

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// Create log file with CF app GUIDs (needs to be > 1024 bytes for fingerprinting)
	logFilePath := filepath.Join(tempDir, "cf-logs.json")
	logEntries := []string{
		`{"message": "App started successfully", "app_guid": "app-guid-1", "timestamp": "2025-01-01T10:00:00Z", "padding": "` + strings.Repeat("x", 200) + `"}`,
		`{"message": "Handling HTTP request", "app_guid": "app-guid-2", "timestamp": "2025-01-01T10:00:01Z", "padding": "` + strings.Repeat("x", 200) + `"}`,
		`{"message": "Background job completed", "app_guid": "app-guid-3", "timestamp": "2025-01-01T10:00:02Z", "padding": "` + strings.Repeat("x", 200) + `"}`,
		`{"message": "Another log from app 1", "app_guid": "app-guid-1", "timestamp": "2025-01-01T10:00:03Z", "padding": "` + strings.Repeat("x", 200) + `"}`,
		`{"message": "Final log entry", "app_guid": "app-guid-2", "timestamp": "2025-01-01T10:00:04Z", "padding": "` + strings.Repeat("x", 200) + `"}`,
	}
	err := os.WriteFile(logFilePath, []byte(strings.Join(logEntries, "\n")+"\n"), 0644)
	require.NoError(t, err, "Failed to create log file")

	// Write configuration file
	cfgSB := strings.Builder{}
	tmpl, err := template.New("filebeatConfig").Parse(tmplCfg)
	require.NoError(t, err, "Failed to parse config template")

	require.NoError(t, tmpl.Execute(&cfgSB, map[string]string{
		"log_path":    logFilePath,
		"api_address": mockAPI.URL(),
		"path_home":   tempDir,
	}), "Failed to execute config template")

	filebeat.WriteConfigFile(cfgSB.String())
	filebeat.Start()

	// Wait for Filebeat to start scanning for files
	filebeat.WaitLogsContains(
		fmt.Sprintf("A new file %s has been found", logFilePath),
		30*time.Second,
		"Filebeat did not start looking for files to ingest")

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath),
		10*time.Second,
		"Filebeat did not finish reading the file")

	// Read events from output file
	events := integration.GetEventsFromFileOutput[CFEvent](filebeat, len(logEntries), true)

	// Build expected metadata by app GUID
	expectedByGUID := make(map[string]mockCFApp)
	for _, app := range testApps {
		expectedByGUID[app.GUID] = app
	}

	// Verify events are enriched with CF metadata
	for i, ev := range events {
		appID := ev.CloudFoundry.App.ID
		require.NotEmpty(t, appID, "Event %d missing cloudfoundry.app.id", i)

		expected, ok := expectedByGUID[appID]
		require.True(t, ok, "Event %d has unknown app ID: %s", i, appID)

		// Verify enrichment happened
		assert.Equal(t, expected.Name, ev.CloudFoundry.App.Name,
			"Event %d: app name mismatch", i)
		assert.Equal(t, expected.SpaceGUID, ev.CloudFoundry.Space.ID,
			"Event %d: space ID mismatch", i)
		assert.Equal(t, expected.SpaceName, ev.CloudFoundry.Space.Name,
			"Event %d: space name mismatch", i)
		assert.Equal(t, expected.OrgGUID, ev.CloudFoundry.Org.ID,
			"Event %d: org ID mismatch", i)
		assert.Equal(t, expected.OrgName, ev.CloudFoundry.Org.Name,
			"Event %d: org name mismatch", i)

		t.Logf("Event %d: app=%s enriched with name=%s, space=%s, org=%s",
			i, appID, ev.CloudFoundry.App.Name, ev.CloudFoundry.Space.Name, ev.CloudFoundry.Org.Name)
	}

	// Verify we got all expected events
	assert.Len(t, events, len(logEntries), "Expected %d events, got %d", len(logEntries), len(events))
}

func TestAddCloudFoundryMetadataProcessor_UnknownApp(t *testing.T) {
	// Start mock CF API with no apps (all lookups will return 404)
	mockAPI := newMockCFAPIServer(nil)
	defer mockAPI.Close()

	var tmplCfg = `
filebeat.inputs:
  - type: filestream
    id: cf-logs
    enabled: true
    paths:
      - {{.log_path}}
    parsers:
      - ndjson:
          keys_under_root: true
          add_error_key: true

processors:
  - copy_fields:
      fields:
        - from: app_guid
          to: cloudfoundry.app.id
      fail_on_error: false
      ignore_missing: true

  - add_cloudfoundry_metadata:
      api_address: {{.api_address}}
      client_id: test-client
      client_secret: test-secret
      ssl:
        verification_mode: none

  - drop_fields:
      fields: [app_guid]
      ignore_missing: true

path.home: {{.path_home}}

output.file:
  path: ${path.home}
  filename: output-file
  rotate_every_kb: 10000

logging.level: debug
`

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// Create log file with unknown app GUID
	logFilePath := filepath.Join(tempDir, "cf-logs.json")
	logEntries := []string{
		`{"message": "Log from unknown app", "app_guid": "unknown-app-guid", "padding": "` + strings.Repeat("x", 300) + `"}`,
	}
	err := os.WriteFile(logFilePath, []byte(strings.Join(logEntries, "\n")+"\n"), 0644)
	require.NoError(t, err)

	cfgSB := strings.Builder{}
	tmpl, err := template.New("filebeatConfig").Parse(tmplCfg)
	require.NoError(t, err)

	require.NoError(t, tmpl.Execute(&cfgSB, map[string]string{
		"log_path":    logFilePath,
		"api_address": mockAPI.URL(),
		"path_home":   tempDir,
	}))

	filebeat.WriteConfigFile(cfgSB.String())
	filebeat.Start()

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath),
		30*time.Second,
		"Filebeat did not finish reading the file")

	// Read events - should still be published, just without enrichment
	events := integration.GetEventsFromFileOutput[CFEvent](filebeat, 1, true)

	require.Len(t, events, 1)
	ev := events[0]

	// App ID should be present but no enrichment
	assert.Equal(t, "unknown-app-guid", ev.CloudFoundry.App.ID)
	assert.Empty(t, ev.CloudFoundry.App.Name, "Unknown app should not have name")
	assert.Empty(t, ev.CloudFoundry.Space.ID, "Unknown app should not have space.id")
	assert.Empty(t, ev.CloudFoundry.Org.ID, "Unknown app should not have org.id")
}

func TestAddCloudFoundryMetadataProcessor_Caching(t *testing.T) {
	// Track API calls to verify caching
	apiCallCount := 0
	testApp := mockCFApp{
		GUID:      "cached-app-guid",
		Name:      "cached-app",
		SpaceGUID: "space-guid-1",
		SpaceName: "production",
		OrgGUID:   "org-guid-1",
		OrgName:   "acme-corp",
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/v2/info", func(w http.ResponseWriter, r *http.Request) {
		serverURL := "http://" + r.Host
		info := map[string]any{
			"name":                   "mock-cf",
			"authorization_endpoint": serverURL,
			"token_endpoint":         serverURL,
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncode(w, info)
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		token := map[string]any{
			"access_token": "mock-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncode(w, token)
	})

	mux.HandleFunc("/v2/apps/", func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		t.Logf("API call #%d: %s", apiCallCount, r.URL.Path)

		response := map[string]any{
			"metadata": map[string]any{"guid": testApp.GUID},
			"entity": map[string]any{
				"name":       testApp.Name,
				"space_guid": testApp.SpaceGUID,
				"space": map[string]any{
					"metadata": map[string]any{"guid": testApp.SpaceGUID},
					"entity": map[string]any{
						"name":              testApp.SpaceName,
						"organization_guid": testApp.OrgGUID,
						"organization": map[string]any{
							"metadata": map[string]any{"guid": testApp.OrgGUID},
							"entity":   map[string]any{"name": testApp.OrgName},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		mustEncode(w, response)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	var tmplCfg = `
filebeat.inputs:
  - type: filestream
    id: cf-logs
    enabled: true
    paths:
      - {{.log_path}}
    parsers:
      - ndjson:
          keys_under_root: true
          add_error_key: true

processors:
  - copy_fields:
      fields:
        - from: app_guid
          to: cloudfoundry.app.id
      fail_on_error: false
      ignore_missing: true

  - add_cloudfoundry_metadata:
      api_address: {{.api_address}}
      client_id: test-client
      client_secret: test-secret
      ssl:
        verification_mode: none

  - drop_fields:
      fields: [app_guid]
      ignore_missing: true

path.home: {{.path_home}}

output.file:
  path: ${path.home}
  filename: output-file
  rotate_every_kb: 10000

logging.level: debug
`

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// Create log file with multiple entries for the SAME app (to test caching)
	logFilePath := filepath.Join(tempDir, "cf-logs.json")
	logEntries := []string{
		`{"message": "First log", "app_guid": "cached-app-guid", "padding": "` + strings.Repeat("x", 300) + `"}`,
		`{"message": "Second log", "app_guid": "cached-app-guid", "padding": "` + strings.Repeat("x", 300) + `"}`,
		`{"message": "Third log", "app_guid": "cached-app-guid", "padding": "` + strings.Repeat("x", 300) + `"}`,
		`{"message": "Fourth log", "app_guid": "cached-app-guid", "padding": "` + strings.Repeat("x", 300) + `"}`,
		`{"message": "Fifth log", "app_guid": "cached-app-guid", "padding": "` + strings.Repeat("x", 300) + `"}`,
	}
	err := os.WriteFile(logFilePath, []byte(strings.Join(logEntries, "\n")+"\n"), 0644)
	require.NoError(t, err)

	cfgSB := strings.Builder{}
	tmpl, err := template.New("filebeatConfig").Parse(tmplCfg)
	require.NoError(t, err)

	require.NoError(t, tmpl.Execute(&cfgSB, map[string]string{
		"log_path":    logFilePath,
		"api_address": server.URL,
		"path_home":   tempDir,
	}))

	filebeat.WriteConfigFile(cfgSB.String())
	filebeat.Start()

	filebeat.WaitLogsContains(
		fmt.Sprintf("End of file reached: %s; Backoff now.", logFilePath),
		30*time.Second,
		"Filebeat did not finish reading the file")

	// Read all events
	events := integration.GetEventsFromFileOutput[CFEvent](filebeat, len(logEntries), true)
	require.Len(t, events, len(logEntries))

	// Verify all events were enriched
	for i, ev := range events {
		assert.Equal(t, testApp.Name, ev.CloudFoundry.App.Name,
			"Event %d should have app name", i)
	}

	// The key assertion: API should only be called ONCE due to caching
	assert.Equal(t, 1, apiCallCount,
		"API should only be called once for 5 events with the same app GUID (caching)")
}

func mustEncode(w io.Writer, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(fmt.Sprintf("failed to encode JSON: %v", err))
	}
}
