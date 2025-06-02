// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/receiver"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewReceiver(t *testing.T) {
	monitorSocket := genSocketPath()
	var monitorHost string
	if runtime.GOOS == "windows" {
		monitorHost = "npipe:///" + filepath.Base(monitorSocket)
	} else {
		monitorHost = "unix://" + monitorSocket
	}
	config := Config{
		Beatconfig: map[string]any{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu", "memory", "network", "filesystem"},
					},
				},
			},
			"output": map[string]any{
				"otelconsumer": map[string]any{},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost,
		},
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: NewFactory(),
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			require.Conditionf(c, func() bool {
				return len(logs["r1"]) > 0
			}, "expected at least one ingest log, got logs: %v", logs["r1"])
			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "stats")
			}, "failed to connect to monitoring socket stats endpoint, last error was: %s", &lastError)
			assert.Conditionf(c, func() bool {
				return getFromSocket(t, &lastError, monitorSocket, "inputs")
			}, "failed to connect to monitoring socket inputs endpoint, last error was: %s", &lastError)
			assert.Condition(c, func() bool {
				processorsLoaded := zapLogs.FilterMessageSnippet("Generated new processors").
					FilterMessageSnippet("add_host_metadata").
					FilterMessageSnippet("add_cloud_metadata").
					FilterMessageSnippet("add_docker_metadata").
					FilterMessageSnippet("add_kubernetes_metadata").
					Len() == 1
				assert.True(c, processorsLoaded, "processors not loaded")
				// Check that add_host_metadata works, other processors are not guaranteed to add fields in all environments
				return assert.Contains(c, logs["r1"][0].Flatten(), "host.architecture")
			}, "failed to check processors loaded")

			assertSystemMetricFields(c, logs["r1"], "expected r1 to have system metric fields in ingested logs")
		},
	})
}

func TestMultipleReceivers(t *testing.T) {
	monitorSocket1 := genSocketPath()
	var monitorHost1 string
	if runtime.GOOS == "windows" {
		monitorHost1 = "npipe:///" + filepath.Base(monitorSocket1)
	} else {
		monitorHost1 = "unix://" + monitorSocket1
	}
	monitorSocket2 := genSocketPath()
	var monitorHost2 string
	if runtime.GOOS == "windows" {
		monitorHost2 = "npipe:///" + filepath.Base(monitorSocket2)
	} else {
		monitorHost2 = "unix://" + monitorSocket2
	}
	config1 := Config{
		Beatconfig: map[string]any{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu", "memory", "network", "filesystem"},
					},
				},
			},
			"output": map[string]any{
				"otelconsumer": map[string]any{},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost1,
		},
	}

	config2 := Config{
		Beatconfig: map[string]any{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu", "memory", "network", "filesystem"},
					},
				},
			},
			"output": map[string]any{
				"otelconsumer": map[string]any{},
			},
			"logging": map[string]any{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home":    t.TempDir(),
			"http.enabled": true,
			"http.host":    monitorHost2,
		},
	}

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config1,
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  &config2,
				Factory: factory,
			},
		},
		AssertFunc: func(c *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			assert.Conditionf(c, func() bool {
				return len(logs["r1"]) > 0 && len(logs["r2"]) > 0
			}, "expected at least one ingest log for each receiver, got logs: %v", logs)
			var lastError strings.Builder
			assert.Conditionf(c, func() bool {
				tests := []string{monitorSocket1, monitorSocket2}
				for _, tc := range tests {
					if ret := getFromSocket(t, &lastError, tc, "stats"); ret == false {
						return false
					}
					if ret := getFromSocket(t, &lastError, tc, "inputs"); ret == false {
						return false
					}
				}
				return true
			}, "failed to connect to monitoring socket, last error was: %s", &lastError)

			assertSystemMetricFields(c, logs["r1"], "expected r1 to have system metric fields in ingested logs")
			assertSystemMetricFields(c, logs["r2"], "expected r2 to have system metric fields in ingested logs")
		},
	})
}

func genSocketPath() string {
	randData := make([]byte, 16)
	for i := range len(randData) {
		randData[i] = uint8(rand.UintN(255)) //nolint:gosec // 0-255 fits in a uint8
	}
	socketName := base64.URLEncoding.EncodeToString(randData) + ".sock"
	socketDir := os.TempDir()
	return filepath.Join(socketDir, socketName)
}

func getFromSocket(t *testing.T, sb *strings.Builder, socketPath string, endpoint string) bool {
	// skip windows for now
	if runtime.GOOS == "windows" {
		return true
	}
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	url, err := url.JoinPath("http://unix", endpoint)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "JoinPath failed: %s", err)
		return false
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "error creating request: %s", err)
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "client.Get failed: %s", err)
		return false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sb.Reset()
		fmt.Fprintf(sb, "io.ReadAll of body failed: %s", err)
		return false
	}
	if len(body) <= 0 {
		sb.Reset()
		sb.WriteString("body too short")
		return false
	}
	if endpoint == "inputs" {
		var data []any
		if err := json.Unmarshal(body, &data); err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "json unmarshal of body failed: %s\n", err)
			fmt.Fprintf(sb, "body was %v\n", body)
			return false
		}

		if len(data) <= 0 {
			sb.Reset()
			sb.WriteString("json array didn't have any entries")
			return false
		}
	} else {
		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			sb.Reset()
			fmt.Fprintf(sb, "json unmarshal of body failed: %s\n", err)
			fmt.Fprintf(sb, "body was %v\n", body)
			return false
		}

		if len(data) <= 0 {
			sb.Reset()
			sb.WriteString("json didn't have any keys")
			return false
		}
	}
	return true
}

func BenchmarkFactory(b *testing.B) {
	tmpDir := b.TempDir()

	cfg := &Config{
		Beatconfig: map[string]interface{}{
			"metricbeat": map[string]any{
				"modules": []map[string]any{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
					},
				},
			},
			"output": map[string]any{
				"otelconsumer": map[string]any{},
			},
			"logging": map[string]any{
				"level": "info",
				"selectors": []string{
					"*",
				},
			},
			"path.home": tmpDir,
		},
	}

	var zapLogs bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(&zapLogs)),
		zapcore.InfoLevel)

	factory := NewFactory()

	receiverSettings := receiver.Settings{}
	receiverSettings.Logger = zap.New(core)
	receiverSettings.ID = component.NewIDWithName(factory.Type(), "r1")

	b.ResetTimer()
	for b.Loop() {
		_, err := factory.CreateLogs(b.Context(), receiverSettings, cfg, nil)
		require.NoError(b, err)
	}
}

func assertSystemMetricFields(c *assert.CollectT, logs []mapstr.M, msg string) {
	metricsetLogs := func(l []mapstr.M, mset string) []mapstr.M {
		var filtered []mapstr.M
		for _, log := range l {
			flat := log.Flatten()
			if flat["event.dataset"] == mset {
				filtered = append(filtered, flat)
			}
		}
		return filtered
	}

	commonFields := []string{
		"@timestamp",
		"agent.ephemeral_id",
		"agent.id",
		"agent.name",
		"agent.type",
		"agent.version",
		"ecs.version",
		"event.dataset",
		"event.duration",
		"event.module",
		"metricset.name",
		"metricset.period",
		"service.type",
		// add_host_metadata
		"host.architecture",
		"host.hostname",
		"host.id",
		"host.ip",
		"host.mac",
		"host.name",
		"host.os.family",
		"host.os.kernel",
		"host.os.name",
		"host.os.platform",
		"host.os.type",
		"host.os.version",
	}

	// Optional fields that may or may not be present depending on the environment or OS.
	optionalFields := []string{
		// not every OS provide this
		"host.os.build",
		"host.os.codename",

		// not available on windows
		"host.containerized",

		// depends on add_cloud_metadata
		// not available locally
		"cloud.account.id",
		"cloud.availability_zone",
		"cloud.instance.id",
		"cloud.instance.name",
		"cloud.machine.type",
		"cloud.project.id",
		"cloud.provider",
		"cloud.region",
		"cloud.service.name",
	}

	testCases := map[string][]string{
		"cpu": {
			"host.cpu.usage",
			"system.cpu.cores",
			"system.cpu.idle.norm.pct",
			"system.cpu.idle.pct",
			"system.cpu.iowait.norm.pct",
			"system.cpu.iowait.pct",
			"system.cpu.irq.norm.pct",
			"system.cpu.irq.pct",
			"system.cpu.nice.norm.pct",
			"system.cpu.nice.pct",
			"system.cpu.softirq.norm.pct",
			"system.cpu.softirq.pct",
			"system.cpu.steal.norm.pct",
			"system.cpu.steal.pct",
			"system.cpu.system.norm.pct",
			"system.cpu.system.pct",
			"system.cpu.total.norm.pct",
			"system.cpu.total.pct",
			"system.cpu.user.norm.pct",
			"system.cpu.user.pct",
		},
		"memory": {
			"system.memory.actual.free",
			"system.memory.actual.used.bytes",
			"system.memory.actual.used.pct",
			"system.memory.cached",
			"system.memory.free",
			"system.memory.swap.free",
			"system.memory.swap.total",
			"system.memory.swap.used.bytes",
			"system.memory.swap.used.pct",
			"system.memory.total",
			"system.memory.used.bytes",
			"system.memory.used.pct",
		},
		"network/interface-card": {
			"system.network.in.bytes",
			"system.network.in.dropped",
			"system.network.in.errors",
			"system.network.in.packets",
			"system.network.name",
			"system.network.out.bytes",
			"system.network.out.dropped",
			"system.network.out.errors",
			"system.network.out.packets",
		},
		"network/host": {
			"host.network.egress.bytes",
			"host.network.egress.packets",
			"host.network.ingress.bytes",
			"host.network.ingress.packets",
		},
		"filesystem": {
			"system.filesystem.available",
			"system.filesystem.device_name",
			"system.filesystem.files",
			"system.filesystem.free",
			"system.filesystem.free_files",
			"system.filesystem.mount_point",
			"system.filesystem.options",
			"system.filesystem.total",
			"system.filesystem.type",
			"system.filesystem.used.bytes",
			"system.filesystem.used.pct",
		},
	}

	for testName, wantFields := range testCases {
		parts := strings.Split(testName, "/")
		mset := parts[0]

		var msetLogs []mapstr.M
		require.Conditionf(c, func() bool {
			msetLogs = metricsetLogs(logs, fmt.Sprintf("system.%s", mset))
			return len(msetLogs) > 0
		}, msg+": expected at least one ingest log for metricset %s, got 0: %v", testName, logs)

		for _, doc := range msetLogs {
			if _, ok := doc[wantFields[0]]; !ok {
				continue
			}

			fields := *doc.FlattenKeys()
			slices.Sort(fields)
			wantFields := append(wantFields, commonFields...)
			// Remove optional fields before comparing, in case they cause a mismatch.
			if diff := cmp.Diff(fields, wantFields); diff != "" {
				// Filter out optional fields
				var filtered []string
				for _, field := range fields {
					if !slices.Contains(optionalFields, field) {
						filtered = append(filtered, field)
					}
				}
				fields = filtered
			}
			slices.Sort(wantFields)
			assert.Equal(c, wantFields, fields, msg+": unexpected fields for metricset %s: got %v, want %v", testName, fields, wantFields)
		}
	}
}
