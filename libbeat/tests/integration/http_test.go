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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type Stats struct {
	Libbeat Libbeat `json:"libbeat"`
}

type Libbeat struct {
	Config Config `json:"config"`
}

type Config struct {
	Scans int `json:"scans"`
}

func TestHttpRoot(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-E", "http.enabled=true")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)
	var m map[string]interface{}
	err = json.Unmarshal(body, &m)

	require.NoError(t, err)
	require.Equal(t, "mockbeat", m["beat"])
	require.Equal(t, "9.9.9", m["version"])
}

func TestHttpStats(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-E", "http.enabled=true")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066/stats")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	body, err := ioutil.ReadAll(r.Body)
	require.NoError(t, err)
	var m Stats

	// Setting the value to 1 to make sure 'body' does have 0 in it
	m.Libbeat.Config.Scans = 1
	err = json.Unmarshal(body, &m)

	require.NoError(t, err)
	require.Equal(t, 0, m.Libbeat.Config.Scans)
}

func TestHttpError(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-E", "http.enabled=true")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066/not-exist")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

func TestHttpPProfDisabled(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: false
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-E", "http.enabled=true")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066/debug/pprof/")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}
