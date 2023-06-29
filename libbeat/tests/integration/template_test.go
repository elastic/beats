//go:build integration

package integration

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test that beat stops in case elasticsearch index is modified and pattern not
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

// Test that beat starts running if elasticsearch output is set
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

// Test that beat stops in case elasticsearch index is modified and pattern not
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

// Test that beat stops in case elasticsearch index is modified and name not
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

// Test that beat starts running if elasticsearch output with modified index and pattern and name are set
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

// Test loading of json based template
func TestJsonTemplate(t *testing.T) {
	EnsureESIsRunning(t)
	_, err := os.Stat("../files/template.json")
	require.Equal(t, err, nil)

	templateName := "bla"
	var mockbeatConfigWithES = `
mockbeat:
output:
  elasticsearch:
    hosts: %s
    username: %s
    password: %s
    allow_older_versions: true
setup.template:
  name: test
  pattern: test-*
  overwrite: true
  json:
    enabled: true
    path: %s
    name: %s
logging:
  level: debug
`

	// prepare the config
	pwd, err := os.Getwd()
	path := filepath.Join(pwd, "../files/template.json")
	esUrl := GetESURL(t, "http")
	user := esUrl.User.Username()
	pass, _ := esUrl.User.Password()
	cfg := fmt.Sprintf(mockbeatConfigWithES, esUrl.String(), user, pass, path, templateName)

	// start mockbeat and wait for the relevant log lines
	mockbeat := startMockBeat(t, "mockbeat start running.", cfg)
	msg := "Loading json template from file"
	mockbeat.WaitForLogs(msg, 60*time.Second)
	msg = "Template with name \\\"bla\\\" loaded."
	mockbeat.WaitForLogs(msg, 60*time.Second)

	// check effective changes in ES
	m := ESGetIndexTemplate(t, esUrl, templateName)
	require.Equal(t, len(m.IndexTemplates), 1)
}

// Test run cmd with default settings for template
func TestTemplateDefault(t *testing.T) {
	EnsureESIsRunning(t)

	var mockbeatConfigWithES = `
mockbeat:
output:
  elasticsearch:
    hosts: %s
    username: %s
    password: %s
    allow_older_versions: true
logging:
  level: debug
`
	datastream := "mockbeat-9.9.9"

	// prepare the config
	esUrl := GetESURL(t, "http")
	user := esUrl.User.Username()
	pass, _ := esUrl.User.Password()
	cfg := fmt.Sprintf(mockbeatConfigWithES, esUrl.String(), user, pass)

	// Delete the existing index template
	ESDeleteIndexTemplate(t, esUrl, datastream)

	// start mockbeat and wait for the relevant log lines
	mockbeat := startMockBeat(t, "mockbeat start running.", cfg)
	mockbeat.WaitForLogs("Template with name \\\"mockbeat-9.9.9\\\" loaded.", 20*time.Second)
	mockbeat.WaitForLogs("PublishEvents: 1 events have been published", 20*time.Second)

	m := ESGetIndexTemplate(t, esUrl, datastream)
	require.Equal(t, len(m.IndexTemplates), 1)
	require.Equal(t, datastream, m.IndexTemplates[0].Name)

	ESPostRefresh(t, esUrl)
	docs := ESGetDocumentsFromDataStream(t, esUrl, datastream)
	require.True(t, docs.Hits.Total.Value > 0)
}

// Test run cmd does not load template when disabled in config
func TestTemplateDisabled(t *testing.T) {
	EnsureESIsRunning(t)

	var mockbeatConfigWithES = `
mockbeat:
output:
  elasticsearch:
    hosts: %s
    username: %s
    password: %s
    allow_older_versions: true
setup.template:
  enabled: false
logging:
  level: debug
`
	datastream := "mockbeat-9.9.9"

	// prepare the config
	esUrl := GetESURL(t, "http")
	user := esUrl.User.Username()
	pass, _ := esUrl.User.Password()
	cfg := fmt.Sprintf(mockbeatConfigWithES, esUrl.String(), user, pass)

	// Delete the existing index template
	ESDeleteIndexTemplate(t, esUrl, datastream)

	// start mockbeat and wait for the relevant log lines
	mockbeat := startMockBeat(t, "mockbeat start running.", cfg)
	mockbeat.WaitForLogs("PublishEvents: 1 events have been published", 20*time.Second)

	u := fmt.Sprintf("%s/_index_template/%s", esUrl.String(), datastream)
	r, _ := http.Get(u)
	require.Equal(t, 404, r.StatusCode, "incorrect status code")
}
