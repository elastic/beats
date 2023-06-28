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

func startMockBeat(t *testing.T) {
	cfg := `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.console:
  code.json:
    pretty: true
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-E", "http.enabled=true")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("Starting stats endpoint", 60*time.Second, "error waiting for stats endpoint to start")
}

func TestHttpRoot(t *testing.T) {
	startMockBeat(t)
	r, _ := http.Get("http://localhost:5066")
	require.Equal(t, 200, r.StatusCode, "incorrect status code")

	body, _ := ioutil.ReadAll(r.Body)
	var m map[string]interface{}
	json.Unmarshal(body, &m)

	require.Equal(t, "mockbeat", m["beat"])
	require.Equal(t, "9.9.9", m["version"])
}

func TestHttpStats(t *testing.T) {
	startMockBeat(t)
	r, _ := http.Get("http://localhost:5066/stats")
	require.Equal(t, 200, r.StatusCode, "incorrect status code")

	body, _ := ioutil.ReadAll(r.Body)
	var m Stats

	// Setting the value to 1 to make sure 'body' does have 0 in it
	m.Libbeat.Config.Scans = 1
	json.Unmarshal(body, &m)

	require.Equal(t, 0, m.Libbeat.Config.Scans)
}

func TestHttpError(t *testing.T) {
	startMockBeat(t)
	r, _ := http.Get("http://localhost:5066/not-exist")
	require.Equal(t, 404, r.StatusCode, "incorrect status code")
}

func TestHttpPProfDisabled(t *testing.T) {
	startMockBeat(t)
	r, _ := http.Get("http://localhost:5066/debug/pprof/")
	require.Equal(t, 404, r.StatusCode, "incorrect status code")
}
