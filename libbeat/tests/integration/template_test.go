//go:build integration

package integration

import (
	"github.com/stretchr/testify/require"
	// "os"
	"testing"
	"time"
)

var mockbeatConfigWithIndex = `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output:
  elasticsearch:
    index: test
`

var mockbeatConfigWithES = `
mockbeat:
name:
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output:
  elasticsearch:
    host: http://localhost:9200
`

func TestIndexModified(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(mockbeatConfigWithIndex)
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("setup.template.name and setup.template.pattern have to be set if index name is modified", 60*time.Second)
}
