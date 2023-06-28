//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cfg = `
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

func startMockBeat(t *testing.T, msg string, args ...string) BeatProc {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", append([]string{"-E", "http.enabled=true"}, args...)...)
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs(msg, 60*time.Second, fmt.Sprintf("error waiting for log: %s", msg))
	return mockbeat
}

func TestLoggingConsoleECS(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", "-E", "http.enabled=true", "-e")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	line := mockbeat.WaitStdErrContains("ecs.version", 60*time.Second)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &m), "Unmarshaling log line as json")

	_, ok := m["log.level"]
	assert.True(t, ok)

	_, ok = m["@timestamp"]
	assert.True(t, ok)

	_, ok = m["message"]
	assert.True(t, ok)
}

// func TestLoggingFileDefault(t *testing.T) {
// 	startMockBeat(t, "Mockbeat is alive!")
// }
