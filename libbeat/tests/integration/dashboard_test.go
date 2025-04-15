// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDashboardLoadSkip(t *testing.T) {
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	kURL, _ := GetKibana(t)
	esURL := GetESURL(t, "http")
	mockbeat.Start("setup",
		"--dashboards",
		"-E", "setup.dashboards.file="+filepath.Join("./testdata", "testbeat-no-dashboards.zip"),
		"-E", "setup.dashboards.beat=testbeat",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "setup.kibana.username=beats",
		"-E", "setup.kibana.password=testing",
		"-E", "output.elasticsearch.hosts=['"+esURL.String()+"']",
		"-E", "output.elasticsearch.username=admin",
		"-E", "output.elasticsearch.password=testing",
		"-E", "output.file.enabled=false")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("Skipping loading dashboards", 10*time.Second)
}

func TestDashboardLoad(t *testing.T) {
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	kURL, _ := GetKibana(t)
	esURL := GetESURL(t, "http")
	mockbeat.Start("setup",
		"--dashboards",
		"-E", "setup.dashboards.file="+filepath.Join("./testdata", "testbeat-dashboards.zip"),
		"-E", "setup.dashboards.beat=testbeat",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "setup.kibana.username=beats",
		"-E", "setup.kibana.password=testing",
		"-E", "output.elasticsearch.hosts=['"+esURL.String()+"']",
		"-E", "output.elasticsearch.username=admin",
		"-E", "output.elasticsearch.password=testing",
		"-E", "output.file.enabled=false")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitForLogs("Kibana dashboards successfully loaded", 30*time.Second)
}

func TestDashboardLoadIndexOnly(t *testing.T) {
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	kURL, _ := GetKibana(t)
	esURL := GetESURL(t, "http")
	mockbeat.Start("setup",
		"--dashboards",
		"-E", "setup.dashboards.file="+filepath.Join("./testdata", "testbeat-dashboards.zip"),
		"-E", "setup.dashboards.beat=testbeat",
		"-E", "setup.dashboards.only_index=true",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "setup.kibana.username=beats",
		"-E", "setup.kibana.password=testing",
		"-E", "output.elasticsearch.hosts=['"+esURL.String()+"']",
		"-E", "output.elasticsearch.username=admin",
		"-E", "output.elasticsearch.password=testing",
		"-E", "output.file.enabled=false")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitForLogs("Kibana dashboards successfully loaded", 30*time.Second)
}

func TestDashboardExportById(t *testing.T) {
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	kURL, _ := GetKibana(t)
	esURL := GetESURL(t, "http")
	mockbeat.Start("setup",
		"--dashboards",
		"-E", "setup.dashboards.file="+filepath.Join("./testdata", "testbeat-dashboards.zip"),
		"-E", "setup.dashboards.beat=testbeat",
		"-E", "setup.dashboards.only_index=true",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "setup.kibana.username=beats",
		"-E", "setup.kibana.password=testing",
		"-E", "output.elasticsearch.hosts=['"+esURL.String()+"']",
		"-E", "output.elasticsearch.username=admin",
		"-E", "output.elasticsearch.password=testing",
		"-E", "output.file.enabled=false")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitForLogs("Kibana dashboards successfully loaded", 30*time.Second)

	mockbeat.Start("export",
		"dashboard",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "setup.kibana.username=beats",
		"-E", "setup.kibana.password=testing",
		"-id", "Metricbeat-system-overview",
		"-folder", filepath.Join(mockbeat.TempDir(), "system-overview"))
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	dbPath := filepath.Join(mockbeat.TempDir(), "system-overview", "_meta", "kibana", "8", "dashboard", "Metricbeat-system-overview.json")
	require.FileExists(t, dbPath, "dashboard file not exported")
	b, err := os.ReadFile(dbPath)
	require.NoError(t, err)
	require.Contains(t, string(b), "Metricbeat-system-overview")
}

func TestDashboardExportByUnknownId(t *testing.T) {
	cfg := `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	kURL, _ := GetKibana(t)
	mockbeat.Start("export",
		"dashboard",
		"-E", "setup.kibana.protocol=http",
		"-E", "setup.kibana.host="+kURL.Hostname(),
		"-E", "setup.kibana.port="+kURL.Port(),
		"-E", "setup.kibana.username=beats",
		"-E", "setup.kibana.password=testing",
		"-id", "No-such-dashboard",
		"-folder", filepath.Join(mockbeat.TempDir(), "system-overview"))
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
}
