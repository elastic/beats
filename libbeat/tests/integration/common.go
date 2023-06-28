package integration

import (
	"fmt"
	"testing"
	"time"
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
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test", args...)
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs(msg, 60*time.Second, fmt.Sprintf("error waiting for log: %s", msg))
	return mockbeat
}
