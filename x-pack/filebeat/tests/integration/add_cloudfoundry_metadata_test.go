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

const (
	cfClientID        = "test-client"
	cfClientSecret    = "test-secret"
	fingerprintOffset = 0
	fingerprintLength = 64
)

const filebeatConfigTemplate = `
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
    prospector.scanner.fingerprint.offset: {{.fingerprint_offset}}
    prospector.scanner.fingerprint.length: {{.fingerprint_length}}

processors:
  - add_cloudfoundry_metadata:
      api_address: {{.api_address}}
      client_id: {{.client_id}}
      client_secret: {{.client_secret}}
      ssl:
        verification_mode: none

path.home: {{.path_home}}

output.file:
  path: ${path.home}
  filename: output-file
  rotate_every_kb: 10000

logging.level: debug
`

var filebeatConfigTmpl = template.Must(template.New("filebeatConfig").Parse(filebeatConfigTemplate))

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

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// Create log file with CF app GUIDs.
	logFilePath := filepath.Join(tempDir, "cf-logs.json")
	logEntries := []string{
		`{"message": "App started successfully", "cloudfoundry": {"app": {"id": "app-guid-1"}}, "timestamp": "2026-01-01T10:00:00Z"}`,
		`{"message": "Handling HTTP request", "cloudfoundry": {"app": {"id": "app-guid-2"}}, "timestamp": "2026-01-01T10:00:01Z"}`,
		`{"message": "Background job completed", "cloudfoundry": {"app": {"id": "app-guid-3"}}, "timestamp": "2026-01-01T10:00:02Z"}`,
		`{"message": "Another log from app 1", "cloudfoundry": {"app": {"id": "app-guid-1"}}, "timestamp": "2026-01-01T10:00:03Z"}`,
		`{"message": "Final log entry", "cloudfoundry": {"app": {"id": "app-guid-2"}}, "timestamp": "2026-01-01T10:00:04Z"}`,
	}
	writeNDJSONFile(t, logFilePath, logEntries)

	filebeat.WriteConfigFile(renderFilebeatConfig(t, logFilePath, mockAPI.URL(), tempDir))
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

	expectedByGUID := appsByGUID(testApps)

	// Verify events are enriched with CF metadata
	for i, ev := range events {
		appID := ev.CloudFoundry.App.ID
		require.NotEmpty(t, appID, "Event %d missing cloudfoundry.app.id", i)

		require.Containsf(t, expectedByGUID, appID, "Event %d has unknown app ID: %s", i, appID)
		expected := expectedByGUID[appID]

		assert.Equal(t, cloudFoundryForApp(expected), ev.CloudFoundry,
			"Event %d: cloudfoundry metadata mismatch", i)

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

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// Create log file with unknown app GUID.
	logFilePath := filepath.Join(tempDir, "cf-logs.json")
	logEntries := []string{
		`{"message": "Log from unknown app", "cloudfoundry": {"app": {"id": "unknown-app-guid"}}}`,
	}
	writeNDJSONFile(t, logFilePath, logEntries)

	filebeat.WriteConfigFile(renderFilebeatConfig(t, logFilePath, mockAPI.URL(), tempDir))
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

	filebeat := NewFilebeat(t)
	tempDir := filebeat.TempDir()

	// Create log file with multiple entries for the SAME app (to test caching)
	logFilePath := filepath.Join(tempDir, "cf-logs.json")
	logEntries := []string{
		`{"message": "First log", "cloudfoundry": {"app": {"id": "cached-app-guid"}}}`,
		`{"message": "Second log", "cloudfoundry": {"app": {"id": "cached-app-guid"}}}`,
		`{"message": "Third log", "cloudfoundry": {"app": {"id": "cached-app-guid"}}}`,
		`{"message": "Fourth log", "cloudfoundry": {"app": {"id": "cached-app-guid"}}}`,
		`{"message": "Fifth log", "cloudfoundry": {"app": {"id": "cached-app-guid"}}}`,
	}
	writeNDJSONFile(t, logFilePath, logEntries)

	filebeat.WriteConfigFile(renderFilebeatConfig(t, logFilePath, server.URL, tempDir))
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

type CFEvent struct {
	Message      string         `json:"message"`
	CloudFoundry CFCloudFoundry `json:"cloudfoundry"`
	Input        CFInput        `json:"input"`
}

type CFCloudFoundry struct {
	App   CFApp   `json:"app"`
	Space CFSpace `json:"space"`
	Org   CFOrg   `json:"org"`
}

type CFApp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CFSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CFOrg struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CFInput struct {
	Type string `json:"type"`
}

func mustEncode(w io.Writer, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(fmt.Sprintf("failed to encode JSON: %v", err))
	}
}

func renderFilebeatConfig(t *testing.T, logPath, apiAddress, pathHome string) string {
	t.Helper()
	cfgSB := strings.Builder{}
	require.NoError(t, filebeatConfigTmpl.Execute(&cfgSB, map[string]any{
		"log_path":           logPath,
		"api_address":        apiAddress,
		"path_home":          pathHome,
		"client_id":          cfClientID,
		"client_secret":      cfClientSecret,
		"fingerprint_offset": fingerprintOffset,
		"fingerprint_length": fingerprintLength,
	}))
	return cfgSB.String()
}

func appsByGUID(apps []mockCFApp) map[string]mockCFApp {
	m := make(map[string]mockCFApp, len(apps))
	for _, app := range apps {
		m[app.GUID] = app
	}
	return m
}

func cloudFoundryForApp(app mockCFApp) CFCloudFoundry {
	return CFCloudFoundry{
		App: CFApp{
			ID:   app.GUID,
			Name: app.Name,
		},
		Space: CFSpace{
			ID:   app.SpaceGUID,
			Name: app.SpaceName,
		},
		Org: CFOrg{
			ID:   app.OrgGUID,
			Name: app.OrgName,
		},
	}
}

func writeNDJSONFile(t *testing.T, path string, entries []string) {
	t.Helper()
	content := strings.Join(entries, "\n") + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
