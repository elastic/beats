// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

// ensureOsquerydAvailable skips the test if osqueryd is not available on the host.
// GoIntegTest sets OSQUERYBEAT_BINARY_DIR to the base install directory downloaded
// by FetchOsquerydForTesting (build/data/install/{os}/{arch}).
func ensureOsquerydAvailable(t *testing.T) {
	t.Helper()
	// Check the directory set by GoIntegTest first (accounts for platform-specific
	// binary paths, e.g. osquery.app/Contents/MacOS/osqueryd on darwin).
	if dir := filepath.Clean(os.Getenv("OSQUERYBEAT_BINARY_DIR")); dir != "" && dir != "." {
		candidate := osqd.OsquerydPathForPlatform(runtime.GOOS, dir)
		if _, err := os.Stat(candidate); err == nil {
			return
		}
	}
	// Fall back to common install locations.
	for _, candidate := range []string{
		"osqueryd",
		"/usr/bin/osqueryd",
		"/usr/local/bin/osqueryd",
		"/opt/osquery/bin/osqueryd",
	} {
		if _, err := exec.LookPath(candidate); err == nil {
			return
		}
	}
	t.Skip("osqueryd not found; skipping osquerybeat OTel integration test")
}

func assertMonitoring(t *testing.T, port int) {
	t.Helper()
	address := fmt.Sprintf("http://localhost:%d", port)

	r, err := http.Get(address) //nolint:noctx,bodyclose,gosec // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/stats") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/not-exist") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

func TestOsquerybeatOtelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	ensureOsquerydAvailable(t)

	host := integration.GetESURL(t, "http")
	esUser := host.User.Username()
	esPass, _ := host.User.Password()
	esURL := fmt.Sprintf("%s://%s", host.Scheme, host.Host)

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	otelIndex := "logs-integration-" + namespace
	osqIndex := "logs-osquery_manager.result-" + namespace

	otelMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))
	osqMonitoringPort := int(libbeattesting.MustAvailableTCP4Port(t))

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{otelIndex, osqIndex})
		require.NoError(t, err, "failed to delete data streams")
	})

	otelCfg := fmt.Sprintf(`receivers:
  osquerybeatreceiver:
    osquerybeat:
      inputs:
        - type: osquery
          osquery:
            schedule:
              osquery_info:
                query: "SELECT * FROM osquery_info"
                interval: 60
    logging:
      level: info
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
    http.enabled: true
    http.host: localhost
    http.port: %d
    management.otel.enabled: true
exporters:
  elasticsearch/log:
    endpoints:
      - %s
    compression: none
    user: %s
    password: %s
    logs_index: %s
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
        - osquerybeatreceiver
      exporters:
        - elasticsearch/log
`, t.TempDir(), otelMonitoringPort, esURL, esUser, esPass, otelIndex)

	oteltestcol.New(t, otelCfg)

	osqBeatCfg := fmt.Sprintf(`osquerybeat:
  inputs:
    - type: osquery
      data_stream:
        namespace: %s
      osquery:
        schedule:
          osquery_info:
            query: "SELECT * FROM osquery_info"
            interval: 60
output:
  elasticsearch:
    hosts:
      - %s
    username: %s
    password: %s
queue.mem.flush.timeout: 0s
setup.template.enabled: false
http.enabled: true
http.host: localhost
http.port: %d
`, namespace, host.Host, esUser, esPass, osqMonitoringPort)

	osquerybeat := integration.NewBeat(t, "osquerybeat", "../../osquerybeat.test")
	osquerybeat.WriteConfigFile(osqBeatCfg)
	osquerybeat.Start()
	defer osquerybeat.Stop()

	// Sort by @timestamp ascending so Hits[0] is always the earliest document,
	// giving a deterministic result even if the query fires more than once.
	sortByTimestamp := map[string]any{
		"sort": []map[string]any{
			{"@timestamp": map[string]any{"order": "asc"}},
		},
	}

	var otelDocs estools.Documents
	var osqDocs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			otelDocs, err = estools.PerformQueryForRawQuery(findCtx, sortByTimestamp, ".ds-"+otelIndex+"*", es)
			assert.NoError(ct, err)

			osqDocs, err = estools.PerformQueryForRawQuery(findCtx, sortByTimestamp, ".ds-"+osqIndex+"*", es)
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 otel event, got %d", otelDocs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, osqDocs.Hits.Total.Value, 1, "expected at least 1 osquerybeat event, got %d", osqDocs.Hits.Total.Value)
		},
		3*time.Minute, 5*time.Second, "expected at least 1 event from both\nosquerybeat and otel receiver")

	var otelDoc, osqDoc mapstr.M
	otelDoc = otelDocs.Hits.Hits[0].Source
	osqDoc = osqDocs.Hits.Hits[0].Source

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		// Instance-specific osquery_info fields that differ between two osqueryd processes
		"osquery.instance_id",
		"osquery.pid",
		"osquery.start_time",
		// watcher is the PID of the osquery watchdog process; differs between the two
		// independent osqueryd instances even if both have watchdog enabled.
		"osquery.watcher",
		// Timing/scheduling fields that differ based on when each instance ran
		"osquery_meta.calendar_type",
		"osquery_meta.planned_schedule_time",
		"osquery_meta.unix_time",
		// schedule_execution_count is 0 when start_date is absent (as in this test),
		// but would diverge if start_date were set, since the two processes run at
		// different unix times.
		"osquery_meta.schedule_execution_count",
		// Per-result UUID
		"response_id",
	}

	oteltest.AssertMapsEqual(t, osqDoc, otelDoc, ignoredFields, "expected documents to be equal")

	assert.Equal(t, "osquerybeat", otelDoc.Flatten()["agent.type"], "expected agent.type to be 'osquerybeat' in otel docs")
	assert.Equal(t, "osquerybeat", osqDoc.Flatten()["agent.type"], "expected agent.type to be 'osquerybeat' in osquerybeat docs")

	assertMonitoring(t, otelMonitoringPort)
	assertMonitoring(t, osqMonitoringPort)
}
