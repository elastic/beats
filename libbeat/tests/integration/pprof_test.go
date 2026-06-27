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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var pprofCfg = `
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

func TestHttpPProfIndex(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test",
		"-E", "http.enabled=true",
		"-E", "http.pprof.enabled=true",
	)
	mockbeat.WriteConfigFile(pprofCfg)
	mockbeat.Start()
	mockbeat.WaitLogsContains("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066/debug/pprof/") //nolint:noctx // fine for tests
	require.NoError(t, err)
	_ = r.Body.Close()
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")
}

func TestHttpPProfCmdline(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test",
		"-E", "http.enabled=true",
		"-E", "http.pprof.enabled=true",
	)
	mockbeat.WriteConfigFile(pprofCfg)
	mockbeat.Start()
	mockbeat.WaitLogsContains("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066/debug/pprof/cmdline") //nolint:noctx // fine for tests
	require.NoError(t, err)
	_ = r.Body.Close()
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")
}

func TestHttpPProfNotFound(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test",
		"-E", "http.enabled=true",
		"-E", "http.pprof.enabled=true",
	)
	mockbeat.WriteConfigFile(pprofCfg)
	mockbeat.Start()
	mockbeat.WaitLogsContains("Starting stats endpoint", 60*time.Second)

	r, err := http.Get("http://localhost:5066/debug/pprof/not-exist") //nolint:noctx // fine for tests
	require.NoError(t, err)
	_ = r.Body.Close()
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}
