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
	// Use DisableKeepAlives so each GET closes the TCP connection immediately.
	// Persistent connections are kept in the transport pool and their readLoop/
	// writeLoop goroutines would outlive this test and trip VerifyNoLeaks in
	// subsequent tests.
	client := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	address := fmt.Sprintf("http://localhost:%d", port)

	r, err := client.Get(address) //nolint:noctx // fine for tests
	require.NoError(t, err)
	r.Body.Close()
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = client.Get(address + "/stats") //nolint:noctx // fine for tests
	require.NoError(t, err)
	r.Body.Close()
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = client.Get(address + "/not-exist") //nolint:noctx // fine for tests
	require.NoError(t, err)
	r.Body.Close()
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

func TestOsquerybeatOtelE2E(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Elasticsearch Docker image has no Windows manifest; skipping ES-dependent test on Windows")
	}
	integration.EnsureESIsRunning(t)
	ensureOsquerydAvailable(t)

	host := integration.GetESURL(t, "http")
	esUser := host.User.Username()
	esPass, _ := host.User.Password()
	esURL := fmt.Sprintf("%s://%s", host.Scheme, host.Host)

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	otelIndex := "logs-integration-" + namespace
	osqIndex := "logs-osquery_manager.result-" + namespace

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
    http.port: 0
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
`, t.TempDir(), esURL, esUser, esPass, otelIndex)

	collector := oteltestcol.New(t, otelCfg)
	otelMonitoringPort := collector.MonitoringPort(t)

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
http.port: 0
`, namespace, host.Host, esUser, esPass)

	osquerybeat := integration.NewBeat(t, "osquerybeat", "../../osquerybeat.test")
	osquerybeat.WriteConfigFile(osqBeatCfg)
	osquerybeat.Start()
	defer osquerybeat.Stop()

	osqMonitoringPort := osquerybeat.MonitoringPort(30 * time.Second)

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
		// On Windows osquery derives uuid per-process rather than from hardware, so it
		// differs between the two independent osqueryd instances.
		"osquery.uuid",
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

func TestOsquerybeatOtelMultipleReceiversE2E(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Elasticsearch Docker image has no Windows manifest; skipping ES-dependent test on Windows")
	}
	integration.EnsureESIsRunning(t)
	ensureOsquerydAvailable(t)

	host := integration.GetESURL(t, "http")
	esUser := host.User.Username()
	esPass, _ := host.User.Password()
	esURL := fmt.Sprintf("%s://%s", host.Scheme, host.Host)

	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace

	type receiverEntry struct {
		id       int
		pathHome string
	}

	tmpDir := t.TempDir()
	receivers := []receiverEntry{
		{id: 0, pathHome: filepath.Join(tmpDir, "r0")},
		{id: 1, pathHome: filepath.Join(tmpDir, "r1")},
	}

	es := integration.GetESClient(t, "http")
	t.Cleanup(func() {
		_, err := es.Indices.DeleteDataStream([]string{index})
		require.NoError(t, err, "failed to delete data streams")
	})

	// Build a config with two named receiver instances. Each stamps its events
	// with a unique receiverid field so ES queries can distinguish them.
	otelCfg := fmt.Sprintf(`receivers:
  osquerybeatreceiver/0:
    osquerybeat:
      inputs:
        - type: osquery
          osquery:
            schedule:
              osquery_info:
                query: "SELECT * FROM osquery_info"
                interval: 10
    processors:
      - add_fields:
          target: ''
          fields:
            receiverid: "0"
    logging:
      level: info
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
    http.enabled: true
    http.host: localhost
    http.port: 0
    management.otel.enabled: true
  osquerybeatreceiver/1:
    osquerybeat:
      inputs:
        - type: osquery
          osquery:
            schedule:
              osquery_info:
                query: "SELECT * FROM osquery_info"
                interval: 10
    processors:
      - add_fields:
          target: ''
          fields:
            receiverid: "1"
    logging:
      level: info
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: %s
    http.enabled: true
    http.host: localhost
    http.port: 0
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
        - osquerybeatreceiver/0
        - osquerybeatreceiver/1
      exporters:
        - elasticsearch/log
`,
		receivers[0].pathHome,
		receivers[1].pathHome,
		esURL, esUser, esPass, index)

	collector := oteltestcol.New(t, otelCfg)

	queryForReceiver := func(id string) map[string]any {
		return map[string]any{
			"query": map[string]any{
				"match": map[string]any{"receiverid": id},
			},
			"sort": []map[string]any{
				{"@timestamp": map[string]any{"order": "asc"}},
			},
		}
	}

	var r0Docs, r1Docs estools.Documents
	var err error

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer findCancel()

			r0Docs, err = estools.PerformQueryForRawQuery(findCtx, queryForReceiver("0"), ".ds-"+index+"*", es)
			assert.NoError(ct, err)

			r1Docs, err = estools.PerformQueryForRawQuery(findCtx, queryForReceiver("1"), ".ds-"+index+"*", es)
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, r0Docs.Hits.Total.Value, 1,
				"expected at least 1 event from receiver 0, got %d", r0Docs.Hits.Total.Value)
			assert.GreaterOrEqual(ct, r1Docs.Hits.Total.Value, 1,
				"expected at least 1 event from receiver 1, got %d", r1Docs.Hits.Total.Value)
		},
		5*time.Minute, 5*time.Second,
		"expected at least 1 event from each osquerybeatreceiver instance")

	// Both receiver instances should produce structurally identical documents
	// (same field set, differing only in instance-specific values).
	r0Doc := r0Docs.Hits.Hits[0].Source
	r1Doc := r1Docs.Hits.Hits[0].Source

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"osquery.instance_id",
		"osquery.pid",
		"osquery.start_time",
		"osquery.watcher",
		// On Linux osquery derives uuid per-process rather than from hardware when
		// multiple independent osqueryd instances run concurrently, so it differs
		// between the two receivers.
		"osquery.uuid",
		"osquery_meta.calendar_type",
		"osquery_meta.planned_schedule_time",
		"osquery_meta.unix_time",
		"osquery_meta.schedule_execution_count",
		"response_id",
		// receiverid is intentionally different between the two instances
		"receiverid",
	}
	oteltest.AssertMapsEqual(t, r0Doc, r1Doc, ignoredFields,
		"expected documents from both receivers to have the same schema")

	for _, port := range collector.MonitoringPorts(t, 2) {
		assertMonitoring(t, port)
	}
}
