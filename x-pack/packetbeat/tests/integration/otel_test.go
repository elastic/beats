// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/otel/oteltest"
	"github.com/elastic/beats/v7/x-pack/otel/oteltestcol"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing/estools"
)

// loopbackDevice returns the platform-specific loopback interface name.
func loopbackDevice() string {
	if runtime.GOOS == "darwin" {
		return "lo0"
	}
	return "lo"
}

// skipIfNotRoot skips the test when not running as root, since live packet
// capture requires elevated privileges on most platforms.
func skipIfNotRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("skipping: packet capture requires root privileges")
	}
}

// startHTTPServer starts an HTTP server bound to an ephemeral port on the
// loopback interface and registers cleanup to stop it when the test ends.
// Binding to an OS-assigned port avoids the time-of-check/time-of-use race of
// pre-allocating a port.
func startHTTPServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "packetbeat integration test")
	}))
	t.Cleanup(srv.Close)
	return srv
}

// serverPort returns the ephemeral TCP port the test HTTP server bound to.
func serverPort(t *testing.T, srv *httptest.Server) int {
	t.Helper()
	addr, ok := srv.Listener.Addr().(*net.TCPAddr)
	require.Truef(t, ok, "expected a TCP listener address, got %T", srv.Listener.Addr())
	return addr.Port
}

// sendHTTPRequests sends numRequests GET requests to url, ignoring errors.
func sendHTTPRequests(url string, numRequests int) {
	client := &http.Client{Timeout: 5 * time.Second}
	for i := 0; i < numRequests; i++ {
		resp, err := client.Get(url) //nolint:noctx // fine for tests
		if err == nil {
			resp.Body.Close()
		}
	}
}

// keepSendingHTTPRequests sends HTTP requests to url every interval until ctx
// is cancelled.  The packetbeat sniffer starts asynchronously, so traffic must
// be generated continuously rather than in a one-shot burst.
func keepSendingHTTPRequests(ctx context.Context, url string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sendHTTPRequests(url, 5)
		}
	}
}

func renderOtelConfig(tb testing.TB, cfgTemplate string, data any) string {
	tb.Helper()
	var buf bytes.Buffer
	require.NoError(tb, template.Must(template.New("config").Parse(cfgTemplate)).Execute(&buf, data))
	cfg := buf.String()
	tb.Cleanup(func() {
		if tb.Failed() {
			tb.Logf("OTel config:\n%s", cfg)
		}
	})
	return cfg
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

// TestPacketbeatOTelE2E verifies that a packetbeat OTel receiver ingests events
// from a pre-recorded pcap file and publishes them to Elasticsearch.
// No live packet-capture capability (root / npcap) is required.
func TestPacketbeatOTelE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace

	cfg := fmt.Sprintf(`receivers:
  packetbeatreceiver:
    packetbeat:
      interfaces:
        file: ../../../../packetbeat/tests/system/pcaps/http_x_forwarded_for.pcap
      protocols:
        - type: http
          ports: [80]
    logging:
      level: info
      selectors:
        - '*'
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
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: %s
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
        - packetbeatreceiver
      exporters:
        - elasticsearch/log
`, tmpdir, index)

	collector := oteltestcol.New(t, cfg)

	es := integration.GetESClient(t, "http")

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			docs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+index+"*")
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, docs.Hits.Total.Value, 1, "expected at least 1 event, got %d", docs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected packetbeat events in ES")

	assertMonitoring(t, collector.MonitoringPort(t))
}

type receiverConfig struct {
	PathHome string
	PcapFile string
	Protocol string
	Ports    []int
}

// TestPacketbeatOTelMultipleReceiversE2E verifies that multiple packetbeat
// OTel receivers run in isolation, each replaying a different pre-recorded
// pcap file and publishing events to Elasticsearch.
// No live packet-capture capability (root / npcap) is required.
func TestPacketbeatOTelMultipleReceiversE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace

	receivers := []receiverConfig{
		{
			PathHome: tmpdir + "/r0",
			PcapFile: "../../../../packetbeat/tests/system/pcaps/http_x_forwarded_for.pcap",
			Protocol: "http",
			Ports:    []int{80},
		},
		{
			PathHome: tmpdir + "/r1",
			PcapFile: "../../../../packetbeat/tests/system/pcaps/dns_google_com.pcap",
			Protocol: "dns",
			Ports:    []int{53},
		},
	}

	cfg := renderOtelConfig(t, `receivers:
{{range $i, $r := .Receivers}}
  packetbeatreceiver/{{$i}}:
    packetbeat:
      id: pbreceiver-{{$i}}
      interfaces:
        file: {{$r.PcapFile}}
      protocols:
        - type: {{$r.Protocol}}
          ports: [{{index $r.Ports 0}}]
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    setup.template.enabled: false
    path.home: {{$r.PathHome}}
    http.enabled: true
    http.host: localhost
    http.port: 0
    management.otel.enabled: true
{{end}}
exporters:
  elasticsearch/log:
    endpoints:
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: {{.Index}}
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
{{range $i, $r := .Receivers}}
        - packetbeatreceiver/{{$i}}
{{end}}
      exporters:
        - elasticsearch/log
`, map[string]any{
		"Index":     index,
		"Receivers": receivers,
	})

	collector := oteltestcol.New(t, cfg)

	es := integration.GetESClient(t, "http")

	wantEvents := len(receivers)
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			docs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+index+"*")
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, docs.Hits.Total.Value, wantEvents,
				"expected at least %d events, got %d", wantEvents, docs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected events from %d receivers in ES", len(receivers))

	for _, port := range collector.MonitoringPorts(t, len(receivers)) {
		assertMonitoring(t, port)
	}
}

// TestPacketbeatOTelBeatE2E verifies that the packetbeat OTel receiver and
// standalone packetbeat produce equivalent documents in Elasticsearch when
// both replay the same pre-recorded pcap file.
// No live packet-capture capability (root / npcap) is required.
func TestPacketbeatOTelBeatE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	pbOtelIndex := "logs-integration-" + namespace
	pbIndex := "logs-packetbeat-" + namespace

	otelCfg := fmt.Sprintf(`receivers:
  packetbeatreceiver:
    packetbeat:
      interfaces:
        file: ../../../../packetbeat/tests/system/pcaps/http_x_forwarded_for.pcap
      protocols:
        - type: http
          ports: [80]
    logging:
      level: info
      selectors:
        - '*'
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
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: %s
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
        - packetbeatreceiver
      exporters:
        - elasticsearch/log
`, tmpdir, pbOtelIndex)

	collector := oteltestcol.New(t, otelCfg)

	standaloneCfg := fmt.Sprintf(`
packetbeat.interfaces.file: ../../../../packetbeat/tests/system/pcaps/http_x_forwarded_for.pcap
packetbeat.protocols:
  - type: http
    ports: [80]
output.elasticsearch:
  hosts:
    - localhost:9200
  username: admin
  password: testing
  index: %s
setup.template.enabled: false
queue.mem.flush.timeout: 0s
`, pbIndex)

	pb := integration.NewBeat(t, "packetbeat", "../../packetbeat.test")
	pb.WriteConfigFile(standaloneCfg)
	pb.Start()
	defer pb.Stop()

	es := integration.GetESClient(t, "http")

	var pbDocs, otelDocs estools.Documents

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			var err error
			otelDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+pbOtelIndex+"*")
			assert.NoError(ct, err)
			pbDocs, err = estools.GetAllLogsForIndexWithContext(findCtx, es, pbIndex+"*")
			assert.NoError(ct, err)

			assert.GreaterOrEqual(ct, otelDocs.Hits.Total.Value, 1, "expected at least 1 otel event")
			assert.GreaterOrEqual(ct, pbDocs.Hits.Total.Value, 1, "expected at least 1 standalone event")
		},
		2*time.Minute, 1*time.Second, "expected events from both packetbeat and otel receiver in ES")

	otelDoc := mapstr.M(otelDocs.Hits.Hits[0].Source)
	pbDoc := mapstr.M(pbDocs.Hits.Hits[0].Source)

	ignoredFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"agent.name",
		"event.start",
		"event.end",
		"event.duration",
	}
	oteltest.AssertMapsEqual(t, pbDoc, otelDoc, ignoredFields, "expected standalone and otel documents to be equal")

	assert.Equal(t, "packetbeat", otelDoc.Flatten()["agent.type"], "expected agent.type to be 'packetbeat' in otel doc")
	assert.Equal(t, "packetbeat", pbDoc.Flatten()["agent.type"], "expected agent.type to be 'packetbeat' in standalone doc")

	assertMonitoring(t, collector.MonitoringPort(t))
}

// TestPacketbeatOTelLiveInterfaceE2E verifies that a packetbeat OTel receiver
// can capture live HTTP traffic from a network interface and publish events to
// Elasticsearch.  Requires root privileges for packet capture.
func TestPacketbeatOTelLiveInterfaceE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	skipIfNotRoot(t)

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace

	// The HTTP server binds an ephemeral port; packetbeat is configured to
	// sniff that resolved port. http.port is 0 so the monitoring server also
	// binds an ephemeral port, discovered from the collector logs below.
	srv := startHTTPServer(t)
	httpPort := serverPort(t, srv)

	cfg := renderOtelConfig(t, `receivers:
  packetbeatreceiver:
    packetbeat:
      interfaces:
        device: {{.Device}}
      protocols:
        - type: http
          ports: [{{.HTTPPort}}]
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{.PathHome}}
    http.enabled: true
    http.host: localhost
    http.port: 0
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: {{.Index}}
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
        - packetbeatreceiver
      exporters:
        - debug
        - elasticsearch/log
  telemetry:
    logs:
      level: DEBUG
`, map[string]any{
		"Device":   loopbackDevice(),
		"HTTPPort": httpPort,
		"PathHome": tmpdir,
		"Index":    index,
	})

	collector := oteltestcol.New(t, cfg)

	// Generate HTTP traffic continuously: the packetbeat sniffer starts
	// asynchronously, so a one-shot burst would race with capture startup.
	go keepSendingHTTPRequests(t.Context(), srv.URL, 500*time.Millisecond)

	es := integration.GetESClient(t, "http")

	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			docs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+index+"*")
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, docs.Hits.Total.Value, 1, "expected at least 1 event, got %d", docs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected packetbeat events in ES")

	assertMonitoring(t, collector.MonitoringPort(t))
}

// TestPacketbeatOTelLiveInterfaceMultipleReceiversE2E verifies that multiple
// packetbeat OTel receivers each capture live traffic on separate ports from a
// network interface and publish events to Elasticsearch.
// Requires root privileges for packet capture.
func TestPacketbeatOTelLiveInterfaceMultipleReceiversE2E(t *testing.T) {
	integration.EnsureESIsRunning(t)
	skipIfNotRoot(t)

	tmpdir := t.TempDir()
	namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
	index := "logs-integration-" + namespace

	// Each HTTP server binds an ephemeral port; packetbeat sniffs the resolved
	// ports. http.port is 0 so each monitoring server binds an ephemeral port
	// too, discovered from the collector logs below.
	type liveReceiverConfig struct {
		PathHome string
		HTTPPort int
	}
	servers := []*httptest.Server{startHTTPServer(t), startHTTPServer(t)}
	liveReceivers := []liveReceiverConfig{
		{PathHome: tmpdir + "/r0", HTTPPort: serverPort(t, servers[0])},
		{PathHome: tmpdir + "/r1", HTTPPort: serverPort(t, servers[1])},
	}

	cfg := renderOtelConfig(t, `receivers:
{{range $i, $r := .Receivers}}
  packetbeatreceiver/{{$i}}:
    packetbeat:
      id: pbreceiver-{{$i}}
      interfaces:
        device: {{$.Device}}
      protocols:
        - type: http
          ports: [{{$r.HTTPPort}}]
    logging:
      level: info
      selectors:
        - '*'
    queue.mem.flush.timeout: 0s
    path.home: {{$r.PathHome}}
    http.enabled: true
    http.host: localhost
    http.port: 0
{{end}}
exporters:
  debug:
    use_internal_logger: false
    verbosity: detailed
  elasticsearch/log:
    endpoints:
      - http://localhost:9200
    compression: none
    user: admin
    password: testing
    logs_index: {{.Index}}
    sending_queue:
      enabled: true
      batch:
        flush_timeout: 1s
service:
  pipelines:
    logs:
      receivers:
{{range $i, $r := .Receivers}}
        - packetbeatreceiver/{{$i}}
{{end}}
      exporters:
        - debug
        - elasticsearch/log
  telemetry:
    logs:
      level: DEBUG
`, map[string]any{
		"Device":    loopbackDevice(),
		"Index":     index,
		"Receivers": liveReceivers,
	})

	collector := oteltestcol.New(t, cfg)

	// Generate HTTP traffic continuously on each server: the packetbeat sniffer
	// starts asynchronously, so a one-shot burst would race with capture startup.
	for _, srv := range servers {
		go keepSendingHTTPRequests(t.Context(), srv.URL, 500*time.Millisecond)
	}

	es := integration.GetESClient(t, "http")

	wantEvents := len(liveReceivers)
	require.EventuallyWithTf(t,
		func(ct *assert.CollectT) {
			findCtx, findCancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer findCancel()

			docs, err := estools.GetAllLogsForIndexWithContext(findCtx, es, ".ds-"+index+"*")
			assert.NoError(ct, err)
			assert.GreaterOrEqual(ct, docs.Hits.Total.Value, wantEvents,
				"expected at least %d events, got %d", wantEvents, docs.Hits.Total.Value)
		},
		2*time.Minute, 1*time.Second, "expected events from %d receivers in ES", len(liveReceivers))

	for _, port := range collector.MonitoringPorts(t, len(liveReceivers)) {
		assertMonitoring(t, port)
	}
}
