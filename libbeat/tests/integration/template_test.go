//go:build integration

package integration

import (
	"github.com/stretchr/testify/require"
	// "os"
	"fmt"
	"testing"
	"time"
)

func TestIndexModified(t *testing.T) {
	var mockbeatConfigWithIndex = `
mockbeat:
output:
  elasticsearch:
    index: test
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(mockbeatConfigWithIndex)
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("setup.template.name and setup.template.pattern have to be set if index name is modified", 60*time.Second)
}

func TestIndexNotModified(t *testing.T) {
	EnsureESIsRunning(t)
	var mockbeatConfigWithES = `
mockbeat:
output:
  elasticsearch:
    hosts: %s
`
	esUrl := GetESURL(t, "http")
	cfg := fmt.Sprintf(mockbeatConfigWithES, esUrl.String())
	startMockBeat(t, "mockbeat start running.", cfg)
}

func TestIndexModifiedNoPattern(t *testing.T) {
	var cfg = `
mockbeat:
output:
  elasticsearch:
    index: test
setup.template:
  name: test
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("setup.template.name and setup.template.pattern have to be set if index name is modified", 60*time.Second)
}

func TestIndexModifiedNoName(t *testing.T) {
	var cfg = `
mockbeat:
output:
  elasticsearch:
    index: test
setup.template:
  pattern: test
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err, "error waiting for mockbeat to exit")
	require.Equal(t, 1, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdErrContains("setup.template.name and setup.template.pattern have to be set if index name is modified", 60*time.Second)
}

func TestIndexWithPatternName(t *testing.T) {
	EnsureESIsRunning(t)
	var mockbeatConfigWithES = `
mockbeat:
output:
  elasticsearch:
    hosts: %s
setup.template:
  name: test
  pattern: test-*
`

	esUrl := GetESURL(t, "http")
	cfg := fmt.Sprintf(mockbeatConfigWithES, esUrl.String())
	startMockBeat(t, "mockbeat start running.", cfg)
}
