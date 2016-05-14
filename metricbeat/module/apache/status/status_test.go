// +build !integration

package status

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
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

	f := mbtest.NewEventFetcher(t, config)
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.Equal(t, 1, event["busyWorkers"])
	assert.Equal(t, 386.299, event["bytesPerReq"])
	assert.Equal(t, .0628293, event["bytesPerSec"])

	connections := event["connections"].(common.MapStr)
	assert.Equal(t, 3, connections["connsAsyncClosing"])
	assert.Equal(t, 2, connections["connsAsyncKeepAlive"])
	assert.Equal(t, 1, connections["connsAsyncWriting"])
	assert.Equal(t, 6, connections["connsTotal"])

	cpu := event["cpu"].(common.MapStr)
	assert.Equal(t, 11.2, cpu["cpuChildrenSystem"])
	assert.Equal(t, 10.1, cpu["cpuChildrenUser"])
	assert.Equal(t, 2.02841, cpu["cpuLoad"])
	assert.Equal(t, 6750.8, cpu["cpuSystem"])
	assert.Equal(t, 14076.6, cpu["cpuUser"])

	assert.Equal(t, server.URL, event["hostname"])
	assert.Equal(t, 99, event["idleWorkers"])

	load := event["load"].(common.MapStr)
	assert.Equal(t, .02, load["load1"])
	assert.Equal(t, .05, load["load15"])
	assert.Equal(t, .01, load["load5"])

	assert.Equal(t, .000162644, event["reqPerSec"])

	scoreboard := event["scoreboard"].(common.MapStr)
	assert.Equal(t, 0, scoreboard["closingConnection"])
	assert.Equal(t, 0, scoreboard["dnsLookup"])
	assert.Equal(t, 0, scoreboard["gracefullyFinishing"])
	assert.Equal(t, 0, scoreboard["idleCleanup"])
	assert.Equal(t, 0, scoreboard["keepalive"])
	assert.Equal(t, 0, scoreboard["logging"])
	assert.Equal(t, 300, scoreboard["openSlot"]) // Number of '.'
	assert.Equal(t, 0, scoreboard["readingRequest"])
	assert.Equal(t, 1, scoreboard["sendingReply"])          // Number of 'W'
	assert.Equal(t, 400, scoreboard["total"])               // Number of scorecard chars.
	assert.Equal(t, 99, scoreboard["waitingForConnection"]) // Number of '_'

	assert.Equal(t, 167, event["totalAccesses"])
	assert.Equal(t, 63, event["totalKBytes"])

	uptime := event["uptime"].(common.MapStr)
	assert.Equal(t, 1026782, uptime["serverUptimeSeconds"])
	assert.Equal(t, 1026782, uptime["uptime"])
}

// TestFetchTimeout verifies that the HTTP request times out and an error is
// returned.
func TestFetchTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(response))
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "apache",
		"metricsets": []string{"status"},
		"hosts":      []string{server.URL},
		"timeout":    "50ms",
	}

	f := mbtest.NewEventFetcher(t, config)

	start := time.Now()
	_, err := f.Fetch()
	elapsed := time.Since(start)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "request canceled (Client.Timeout exceeded")
	}

	// Elapsed should be ~50ms.
	assert.True(t, elapsed < 100*time.Millisecond, "elapsed time: %s", elapsed.String())
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

	f := mbtest.NewEventFetcher(t, config)

	for i := 0; i < 20; i++ {
		_, err := f.Fetch()
		if !assert.NoError(t, err) {
			t.FailNow()
		}
	}

	connLock.Lock()
	assert.Len(t, conns, 1,
		"only a single connection should exist because of keep-alives")
	connLock.Unlock()
}

func TestHostParse(t *testing.T) {
	var tests = []struct {
		host string
		url  string
		err  string
	}{
		{"", "", "error parsing apache host: empty host"},
		{":80", "", "error parsing apache host: parse :80: missing protocol scheme"},
		{"localhost", "http://localhost/server-status?auto=", ""},
		{"localhost/ServerStatus", "http://localhost/ServerStatus?auto=", ""},
		{"127.0.0.1", "http://127.0.0.1/server-status?auto=", ""},
		{"https://127.0.0.1", "https://127.0.0.1/server-status?auto=", ""},
		{"[2001:db8:0:1]:80", "http://[2001:db8:0:1]:80/server-status?auto=", ""},
		{"https://admin:secret@127.0.0.1", "https://admin:secret@127.0.0.1/server-status?auto=", ""},
	}

	for _, test := range tests {
		u, err := getURL("", "", defaultPath, test.host)
		if err != nil && test.err != "" {
			assert.Equal(t, test.err, err.Error())
		} else if assert.NoError(t, err, "unexpected error") {
			assert.Equal(t, test.url, u.String())
		}
	}
}

func TestRedactPassword(t *testing.T) {
	rawURL := "https://admin:secret@127.0.0.1"
	u, err := url.Parse(rawURL)
	if assert.NoError(t, err) {
		assert.Equal(t, "https://admin@127.0.0.1", redactPassword(*u))
		// redactPassword shall not modify the URL.
		assert.Equal(t, rawURL, u.String())
	}
}
