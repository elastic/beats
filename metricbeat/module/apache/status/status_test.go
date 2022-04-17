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

//go:build !integration
// +build !integration

package status

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/menderesk/beats/v7/metricbeat/module/apache"
)

// response is a raw response copied from an Apache web server.
const response = `apache
ServerVersion: Apache/2.4.18 (Unix)
ServerMPM: event
Server Built: Mar  2 2016 21:08:47
CurrentTime: Thursday, 12-May-2016 20:30:25 UTC
RestartTime: Saturday, 30-Apr-2016 23:17:22 UTC
ParentServerConfigGeneration: 1
ParentServerMPMGeneration: 0
ServerUptimeSeconds: 1026782
ServerUptime: 11 days 21 hours 13 minutes 2 seconds
Load1: 0.02
Load5: 0.01
Load15: 0.05
Total Accesses: 167
Total kBytes: 63
CPUUser: 14076.6
CPUSystem: 6750.8
CPUChildrenUser: 10.1
CPUChildrenSystem: 11.2
CPULoad: 2.02841
Uptime: 1026782
ReqPerSec: .000162644
BytesPerSec: .0628293
BytesPerReq: 386.299
BusyWorkers: 1
IdleWorkers: 99
ConnsTotal: 6
ConnsAsyncWriting: 1
ConnsAsyncKeepAlive: 2
ConnsAsyncClosing: 3
Scoreboard: __________________________________________________________________________________W_________________............................................................................................................................................................................................................................................................................................................`

// TestFetchEventContents verifies the contents of the returned event against
// the raw Apache response.
func TestFetchEventContents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "apache",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.Equal(t, 386.299, event["bytes_per_request"])
	assert.Equal(t, .0628293, event["bytes_per_sec"])

	workers := event["workers"].(common.MapStr)
	assert.EqualValues(t, 1, workers["busy"])
	assert.EqualValues(t, 99, workers["idle"])

	connections := event["connections"].(common.MapStr)
	async := connections["async"].(common.MapStr)
	assert.EqualValues(t, 3, async["closing"])
	assert.EqualValues(t, 2, async["keep_alive"])
	assert.EqualValues(t, 1, async["writing"])
	assert.EqualValues(t, 6, connections["total"])

	cpu := event["cpu"].(common.MapStr)
	assert.Equal(t, 11.2, cpu["children_system"])
	assert.Equal(t, 10.1, cpu["children_user"])
	assert.Equal(t, 2.02841, cpu["load"])
	assert.Equal(t, 6750.8, cpu["system"])
	assert.Equal(t, 14076.6, cpu["user"])

	assert.Equal(t, server.URL[7:], event["hostname"])

	load := event["load"].(common.MapStr)
	assert.Equal(t, .02, load["1"])
	assert.Equal(t, .05, load["15"])
	assert.Equal(t, .01, load["5"])

	assert.Equal(t, .000162644, event["requests_per_sec"])

	scoreboard := event["scoreboard"].(common.MapStr)
	assert.Equal(t, 0, scoreboard["closing_connection"])
	assert.Equal(t, 0, scoreboard["dns_lookup"])
	assert.Equal(t, 0, scoreboard["gracefully_finishing"])
	assert.Equal(t, 0, scoreboard["idle_cleanup"])
	assert.Equal(t, 0, scoreboard["keepalive"])
	assert.Equal(t, 0, scoreboard["logging"])
	assert.Equal(t, 300, scoreboard["open_slot"]) // Number of '.'
	assert.Equal(t, 0, scoreboard["reading_request"])
	assert.Equal(t, 1, scoreboard["sending_reply"])           // Number of 'W'
	assert.Equal(t, 400, scoreboard["total"])                 // Number of scorecard chars.
	assert.Equal(t, 99, scoreboard["waiting_for_connection"]) // Number of '_'

	assert.EqualValues(t, 167, event["total_accesses"])
	assert.EqualValues(t, 63, event["total_kbytes"])

	uptime := event["uptime"].(common.MapStr)
	assert.EqualValues(t, 1026782, uptime["uptime"])
	assert.EqualValues(t, 1026782, uptime["server_uptime"])
}

// TestFetchTimeout verifies that the HTTP request times out and an error is
// returned.
func TestFetchTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(response))
		<-r.Context().Done()
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "apache",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
		"timeout":    "50ms",
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	start := time.Now()
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) == 0 {
		t.Fatalf("Expected an error, had %d. %v\n", len(errs), errs)
	}
	assert.Empty(t, events)
	elapsed := time.Since(start)
	var found bool
	for _, err := range errs {
		if strings.Contains(err.Error(), "Client.Timeout exceeded") {
			found = true
		}
	}
	if !found {
		assert.Failf(t, "", "expected an error containing 'Client.Timeout exceeded'. Got %v", errs)
	}

	// Elapsed should be ~50ms, sometimes it can be up to 1s
	assert.True(t, elapsed < 5*time.Second, "elapsed time: %s", elapsed.String())
}

// TestMultipleFetches verifies that the server connection is reused when HTTP
// keep-alive is supported by the server.
func TestMultipleFetches(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(response))
	}))

	connLock := sync.Mutex{}
	conns := map[string]struct{}{}
	server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		connLock.Lock()
		conns[conn.RemoteAddr().String()] = struct{}{}
		connLock.Unlock()
	}

	server.Start()
	defer server.Close()

	config := map[string]interface{}{
		"module":     "apache",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)

	for i := 0; i < 20; i++ {
		_, errs := mbtest.ReportingFetchV2Error(f)
		if len(errs) > 0 {
			t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
		}
	}

	connLock.Lock()
	assert.Len(t, conns, 1,
		"only a single connection should exist because of keep-alives")
	connLock.Unlock()
}

func TestHostParser(t *testing.T) {
	var tests = []struct {
		host string
		url  string
		err  string
	}{
		{"", "", "empty host"},
		{":80", "", "empty host"},
		{"localhost", "http://localhost/server-status?auto=", ""},
		{"localhost/ServerStatus", "http://localhost/ServerStatus?auto=", ""},
		{"127.0.0.1", "http://127.0.0.1/server-status?auto=", ""},
		{"https://127.0.0.1", "https://127.0.0.1/server-status?auto=", ""},
		{"[2001:db8:0:1]:80", "http://[2001:db8:0:1]:80/server-status?auto=", ""},
		{"https://admin:secret@127.0.0.1", "https://admin:secret@127.0.0.1/server-status?auto=", ""},
	}

	for _, test := range tests {
		hostData, err := hostParser(mbtest.NewTestModule(t, map[string]interface{}{}), test.host)
		if err != nil && test.err != "" {
			assert.Contains(t, err.Error(), test.err)
		} else if assert.NoError(t, err, "unexpected error") {
			assert.Equal(t, test.url, hostData.URI)
		}
	}
}

// Test event mapping for different apache status outputs
func TestStatusOutputs(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/status_*")
	assert.NoError(t, err)

	for _, filename := range files {
		f, err := os.Open(filename)
		assert.NoError(t, err, "cannot open test file "+filename)
		scanner := bufio.NewScanner(f)

		_, err = eventMapping(scanner, "localhost")
		assert.NoError(t, err, "error mapping "+filename)
	}
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "apache", "status")
}
