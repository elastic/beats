// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package hbreceiver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap/zaptest"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// startReceiverWithConfig creates and starts a heartbeat receiver using the
// given Config, returning the receiver and a snapshot function that returns a
// copy of all events collected so far.
func startReceiverWithConfig(t *testing.T, cfg *Config) (receiver.Logs, func() []mapstr.M) {
	t.Helper()

	var mu sync.Mutex
	var events []mapstr.M

	logConsumer, err := consumer.NewLogs(func(_ context.Context, ld plog.Logs) error {
		mu.Lock()
		defer mu.Unlock()
		for _, rl := range ld.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					events = append(events, lr.Body().Map().AsRaw())
				}
			}
		}
		return nil
	})
	require.NoError(t, err)

	factory := NewFactoryWithSettings(Settings{Home: t.TempDir()})
	set := receiver.Settings{}
	set.ID = component.NewIDWithName(factory.Type(), "smoke")
	set.Logger = zaptest.NewLogger(t)

	rec, err := factory.CreateLogs(t.Context(), set, cfg, logConsumer)
	require.NoError(t, err)
	require.NoError(t, rec.Start(t.Context(), componenttest.NewNopHost()))

	return rec, func() []mapstr.M {
		mu.Lock()
		defer mu.Unlock()
		cp := make([]mapstr.M, len(events))
		copy(cp, events)
		return cp
	}
}

// staticMonitorConfig builds a Config for a set of static monitors.
func staticMonitorConfig(t *testing.T, monitors []map[string]any, extra map[string]any) *Config {
	t.Helper()
	hbSection := map[string]any{
		"monitors": monitors,
	}
	for k, v := range extra {
		hbSection[k] = v
	}
	return &Config{
		Beatconfig: map[string]any{
			"heartbeat":               hbSection,
			"queue.mem.flush.timeout": "0s",
			"path.home":               t.TempDir(),
		},
	}
}

// startSmokeReceiver is a convenience wrapper for tests with static monitors.
func startSmokeReceiver(t *testing.T, monitors []map[string]any) (receiver.Logs, func() []mapstr.M) {
	t.Helper()
	return startReceiverWithConfig(t, staticMonitorConfig(t, monitors, nil))
}

// assertMonitorEvent waits up to 2 minutes for the first collected event's
// flattened fields to all match want.
func assertMonitorEvent(t *testing.T, snapshot func() []mapstr.M, want map[string]any) {
	t.Helper()
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		events := snapshot()
		require.NotEmpty(ct, events, "no monitor events received yet")
		flat := events[0].Flatten()
		for k, v := range want {
			assert.Equalf(ct, v, flat[k], "field %q", k)
		}
	}, 2*time.Minute, 1*time.Second, "timed out waiting for monitor event")
}

// assertAnyEvent waits up to 2 minutes for at least one event whose flattened
// fields all match want. Unlike assertMonitorEvent it searches the entire
// accumulated slice, which is needed when events from multiple monitors are
// interleaved or when config is reloaded mid-test.
func assertAnyEvent(t *testing.T, snapshot func() []mapstr.M, want map[string]any) {
	t.Helper()
	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		events := snapshot()
		require.NotEmpty(ct, events, "no events received yet")
		found := false
		for _, ev := range events {
			flat := ev.Flatten()
			match := true
			for k, v := range want {
				if flat[k] != v {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
		assert.Truef(ct, found, "no event matching %v found in %d events", want, len(events))
	}, 2*time.Minute, 1*time.Second, "timed out waiting for matching event")
}

// ── HTTP monitor tests ────────────────────────────────────────────────────────

// TestSmokeHTTPMonitorUp verifies that an HTTP monitor reports "up" when the
// target server returns 200.
// Ported from test_monitor.py::test_http[200].
func TestSmokeHTTPMonitorUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls": []string{server.URL},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":              "http",
		"monitor.status":            "up",
		"http.response.status_code": int64(200),
	})
}

// TestSmokeHTTPMonitor404 verifies that an HTTP monitor reports "down" when
// the server returns 404; heartbeat treats non-2xx responses as failures.
// Ported from test_monitor.py::test_http[404].
func TestSmokeHTTPMonitor404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls": []string{server.URL},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":              "http",
		"monitor.status":            "down",
		"http.response.status_code": int64(404),
	})
}

// TestHTTPMonitorHostsConfigUp verifies that the "hosts" config alias works
// identically to "urls" for an HTTP monitor reporting "up" on 200.
// Ported from test_monitor.py::test_http_with_hosts_config[200].
func TestHTTPMonitorHostsConfigUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"hosts": []string{server.URL},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":              "http",
		"monitor.status":            "up",
		"http.response.status_code": int64(200),
	})
}

// TestHTTPMonitorHostsConfig404 verifies that the "hosts" alias also correctly
// surfaces a 404 as "down".
// Ported from test_monitor.py::test_http_with_hosts_config[404].
func TestHTTPMonitorHostsConfig404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"hosts": []string{server.URL},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":              "http",
		"monitor.status":            "down",
		"http.response.status_code": int64(404),
	})
}

// TestHTTPMonitorOptionsEnabled verifies that an HTTP monitor using the OPTIONS
// method reports "up" when the server responds 200 and includes the
// Access-Control-Allow-Methods header.
// Ported from test_monitor.py::test_http_check_with_options_method[enable=True].
func TestHTTPMonitorOptionsEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"hosts": []string{server.URL},
		"check": map[string]any{
			"request": map[string]any{"method": "OPTIONS"},
		},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":              "http",
		"monitor.status":            "up",
		"http.response.status_code": int64(200),
	})
}

// TestHTTPMonitorOptionsDisabled verifies that an HTTP monitor using the
// OPTIONS method reports "down" when the server returns 501.
// Ported from test_monitor.py::test_http_check_with_options_method[enable=False].
func TestHTTPMonitorOptionsDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"hosts": []string{server.URL},
		"check": map[string]any{
			"request": map[string]any{"method": "OPTIONS"},
		},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":              "http",
		"monitor.status":            "down",
		"http.response.status_code": int64(501),
	})
}

// TestHTTPMonitorDelayed verifies that a slow response body is reflected in the
// RTT metrics. The server delays the body by 1 second; http.rtt.total.us must
// be at least 800 000 µs (800 ms) to account for scheduling jitter.
// Ported from test_monitor.py::test_http_delayed.
func TestHTTPMonitorDelayed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush() // send headers immediately so content read time is measured separately
		}
		time.Sleep(1 * time.Second)
		_, _ = w.Write([]byte("slow body"))
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 5s", "timeout": "10s",
		"urls": []string{server.URL},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		events := snapshot()
		require.NotEmpty(ct, events, "no events yet")
		flat := events[0].Flatten()
		val, ok := flat["http.rtt.total.us"].(int64)
		assert.True(ct, ok, "http.rtt.total.us should be int64, got %T", flat["http.rtt.total.us"])
		assert.GreaterOrEqual(ct, val, int64(800_000), "expected RTT to reflect 1-second body delay")
	}, 2*time.Minute, 1*time.Second, "timed out waiting for delayed monitor event")
}

// ── TCP monitor tests ─────────────────────────────────────────────────────────

// TestSmokeTCPMonitorUp verifies that a TCP monitor reports "up" when the
// target port is reachable.
// Ported from test_monitor.py::test_tcp[up].
func TestSmokeTCPMonitorUp(t *testing.T) {
	// A bare listener is sufficient: heartbeat only checks that the TCP
	// handshake completes; it does not exchange application data.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "localhost:0")
	require.NoError(t, err)
	defer ln.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "tcp", "id": "test-tcp",
		"schedule": "@every 1s", "timeout": "3s",
		"hosts": []string{ln.Addr().String()},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":   "tcp",
		"monitor.status": "up",
	})
}

// TestSmokeTCPMonitorDown verifies that a TCP monitor reports "down" when the
// target is unreachable. 203.0.113.1 is TEST-NET-3 (RFC 5737), a non-routable
// address guaranteed to be unreachable.
// Ported from test_monitor.py::test_tcp[down].
func TestSmokeTCPMonitorDown(t *testing.T) {
	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "tcp", "id": "test-tcp",
		"schedule": "@every 1s", "timeout": "3s",
		"hosts": []string{"203.0.113.1:1233"},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":   "tcp",
		"monitor.status": "down",
	})
}

// ── ICMP monitor tests ────────────────────────────────────────────────────────

// TestICMPMonitor verifies that an ICMP monitor produces an event whose status
// is either "up" or "down" — the exact result depends on whether the test
// environment allows unprivileged ICMP.
// Ported from test_icmp.py::Test.test_base.
func TestICMPMonitor(t *testing.T) {
	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type":     "icmp",
		"id":       "test-icmp",
		"schedule": "@every 1s",
		"hosts":    []string{"127.0.0.1"},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		events := snapshot()
		require.NotEmpty(ct, events, "no ICMP events yet")
		flat := events[0].Flatten()
		assert.Equal(ct, "icmp", flat["monitor.type"])
		status, _ := flat["monitor.status"].(string)
		assert.Truef(ct, status == "up" || status == "down",
			"expected monitor.status to be up or down, got %q", status)
	}, 2*time.Minute, 1*time.Second, "timed out waiting for ICMP monitor event")
}

// ── Monitor lifecycle tests ───────────────────────────────────────────────────

// TestMonitorDisabled verifies that a monitor with enabled:false produces no
// events.
// Ported from test_base.py::Test.test_disabled.
func TestMonitorDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls":    []string{server.URL},
		"enabled": false,
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	// Let the scheduler run for a few cycles; a disabled monitor must be silent.
	time.Sleep(3 * time.Second)
	assert.Empty(t, snapshot(), "disabled monitor must not produce events")
}

// TestRunOnce verifies that heartbeat in run_once mode fires monitors once,
// produces events, and exits cleanly.
// Ported from test_base.py::Test.test_run_once.
func TestRunOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := staticMonitorConfig(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls": []string{server.URL},
	}}, map[string]any{"run_once": true})

	rec, snapshot := startReceiverWithConfig(t, cfg)
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":   "http",
		"monitor.status": "up",
	})
}

// ── Event field tests ─────────────────────────────────────────────────────────

// TestEventDataset verifies that event.dataset equals monitor.type for both
// HTTP and TCP monitors.
// Ported from test_base.py::Test.test_dataset.
func TestEventDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "localhost:0")
	require.NoError(t, err)
	defer ln.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{
		{
			"type": "http", "id": "test-http",
			"schedule": "@every 1s", "timeout": "3s",
			"urls": []string{server.URL},
		},
		{
			"type": "tcp", "id": "test-tcp",
			"schedule": "@every 1s", "timeout": "3s",
			"hosts": []string{ln.Addr().String()},
		},
	})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		events := snapshot()
		foundHTTP, foundTCP := false, false
		for _, ev := range events {
			flat := ev.Flatten()
			monType, _ := flat["monitor.type"].(string)
			dataset, _ := flat["event.dataset"].(string)
			if monType == "http" && dataset == monType {
				foundHTTP = true
			}
			if monType == "tcp" && dataset == monType {
				foundTCP = true
			}
		}
		assert.True(ct, foundHTTP, "no HTTP event with event.dataset=http")
		assert.True(ct, foundTCP, "no TCP event with event.dataset=tcp")
	}, 2*time.Minute, 1*time.Second, "timed out waiting for dataset events")
}

// TestMonitorFieldsUnderRoot verifies that custom fields configured with
// fields_under_root:true appear at the top level of events.
// Ported from test_base.py::Test.test_fields_under_root.
func TestMonitorFieldsUnderRoot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls":              []string{server.URL},
		"fields_under_root": true,
		"fields":            map[string]any{"custom_env": "staging", "custom_team": "ops"},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type": "http",
		"custom_env":   "staging",
		"custom_team":  "ops",
	})
}

// TestMonitorFieldsNotUnderRoot verifies that custom fields without
// fields_under_root are namespaced under the "fields" key.
// Ported from test_base.py::Test.test_fields_not_under_root.
func TestMonitorFieldsNotUnderRoot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls":   []string{server.URL},
		"fields": map[string]any{"custom_env": "staging"},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	assertMonitorEvent(t, snapshot, map[string]any{
		"monitor.type":      "http",
		"fields.custom_env": "staging",
	})
}

// TestMonitorTags verifies that tags configured on a monitor are propagated to
// the event.
func TestMonitorTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rec, snapshot := startSmokeReceiver(t, []map[string]any{{
		"type": "http", "id": "test-http",
		"schedule": "@every 1s", "timeout": "3s",
		"urls": []string{server.URL},
		"tags": []string{"smoke", "ci"},
	}})
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	require.EventuallyWithT(t, func(ct *assert.CollectT) {
		events := snapshot()
		require.NotEmpty(ct, events, "no events yet")
		flat := events[0].Flatten()
		tags, ok := flat["tags"].([]any)
		require.True(ct, ok, "tags should be []any, got %T", flat["tags"])
		assert.Contains(ct, tags, "smoke")
		assert.Contains(ct, tags, "ci")
	}, 2*time.Minute, 1*time.Second, "timed out waiting for tagged event")
}

// ── Dynamic config reload tests ───────────────────────────────────────────────

// dynamicMonitorConfig builds a Config with heartbeat.config.monitors pointed
// at configDir/*.yml and reload.period set to 100ms for fast test cycles.
func dynamicMonitorConfig(t *testing.T, configDir string) *Config {
	t.Helper()
	return &Config{
		Beatconfig: map[string]any{
			"heartbeat": map[string]any{
				"config": map[string]any{
					"monitors": map[string]any{
						"path": filepath.Join(configDir, "*.yml"),
						"reload": map[string]any{
							"period":  "100ms",
							"enabled": true,
						},
					},
				},
			},
			"queue.mem.flush.timeout": "0s",
			"path.home":               t.TempDir(),
		},
	}
}

// writeMonitorYAML writes a YAML list of monitor configs to a file in dir.
func writeMonitorYAML(t *testing.T, dir, filename, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644))
}

// TestConfigAdd verifies that writing a monitor config file after the receiver
// starts causes the monitor to begin producing events.
// Ported from test_reload.py::Test.test_config_add.
func TestConfigAdd(t *testing.T) {
	configDir := t.TempDir()
	rec, snapshot := startReceiverWithConfig(t, dynamicMonitorConfig(t, configDir))
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Give the reloader time to scan the (initially empty) config dir before
	// adding the file so we exercise the "add" path rather than "initial load".
	time.Sleep(300 * time.Millisecond)

	writeMonitorYAML(t, configDir, "test.yml", fmt.Sprintf(`
- type: http
  id: dynamic-http
  schedule: "@every 1s"
  timeout: 3s
  urls:
    - %s
`, server.URL))

	assertAnyEvent(t, snapshot, map[string]any{
		"monitor.type":   "http",
		"monitor.status": "up",
	})
}

// TestConfigReload verifies that overwriting a monitor config file causes
// heartbeat to reload and produce events for the new target.
// Ported from test_reload.py::Test.test_config_reload.
func TestConfigReload(t *testing.T) {
	configDir := t.TempDir()
	rec, snapshot := startReceiverWithConfig(t, dynamicMonitorConfig(t, configDir))
	defer func() { require.NoError(t, rec.Shutdown(context.Background())) }()

	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer serverB.Close()

	monitorYAML := func(url string) string {
		return fmt.Sprintf(`
- type: http
  id: dynamic-http
  schedule: "@every 1s"
  timeout: 3s
  urls:
    - %s
`, url)
	}

	// Start with serverA and wait for its events.
	writeMonitorYAML(t, configDir, "test.yml", monitorYAML(serverA.URL))
	assertAnyEvent(t, snapshot, map[string]any{
		"monitor.type": "http",
		"url.full":     serverA.URL,
	})

	// Overwrite with serverB; wait for the reloaded monitor to fire.
	writeMonitorYAML(t, configDir, "test.yml", monitorYAML(serverB.URL))
	assertAnyEvent(t, snapshot, map[string]any{
		"monitor.type": "http",
		"url.full":     serverB.URL,
	})
}
