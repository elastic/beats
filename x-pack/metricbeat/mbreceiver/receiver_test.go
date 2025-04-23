// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"

	"go.uber.org/zap/zaptest/observer"
)

func TestNewReceiver(t *testing.T) {
	monitorSocket := genSocketPath()
	monitorHost := "unix://" + monitorSocket
	config := Config{
		Beatconfig: map[string]any{
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
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			assert.Conditionf(t, func() bool {
				return len(logs["r1"]) > 0
			}, "expected at least one ingest log, got logs: %v", logs["r1"])
			var lastError strings.Builder
			assert.Condition(t, func() bool {
				// skip windows for now
				if runtime.GOOS == "windows" {
					return true
				}
				client := http.Client{
					Transport: &http.Transport{
						DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
							return net.Dial("unix", monitorSocket)
						},
					},
				}
				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://unix/stats", nil)
				if err != nil {
					lastError.Reset()
					lastError.WriteString(fmt.Sprintf("error creating request: %s", err))
					return false
				}
				resp, err := client.Do(req)
				if err != nil {
					lastError.Reset()
					lastError.WriteString(fmt.Sprintf("client.Get failed: %s", err))
					return false
				}
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					lastError.Reset()
					lastError.WriteString(fmt.Sprintf("io.ReadAll of body failed: %s", err))
					return false
				}
				if len(body) <= 0 {
					lastError.Reset()
					lastError.WriteString("body too short")
					return false
				}
				return true
			}, "failed to connect to monitoring socket, last error was: %s", &lastError)
		},
	})
}

func TestMultipleReceivers(t *testing.T) {
	config := Config{
		Beatconfig: map[string]any{
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
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

	factory := NewFactory()
	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T: t,
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:    "r1",
				Config:  &config,
				Factory: factory,
			},
			{
				Name:    "r2",
				Config:  &config,
				Factory: factory,
			},
		},
		AssertFunc: func(t *assert.CollectT, logs map[string][]mapstr.M, zapLogs *observer.ObservedLogs) {
			_ = zapLogs
			assert.Conditionf(t, func() bool {
				return len(logs["r1"]) > 0 && len(logs["r2"]) > 0
			}, "expected at least one ingest log for each receiver, got logs: %v", logs)
		},
	})
}

func genSocketPath() string {
	randData := make([]byte, 16)
	for i := range len(randData) {
		randData[i] = uint8(rand.UintN(255)) //nolint:gosec // 0-255 fits in a unint8
	}
	socketName := base64.URLEncoding.EncodeToString(randData) + ".sock"
	socketDir := os.TempDir()
	return filepath.Join(socketDir, socketName)
}
