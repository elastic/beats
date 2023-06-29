//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type IndexTemplateResult struct {
	IndexTemplates []IndexTemplateEntry `json:"index_templates"`
}

type IndexTemplateEntry struct {
	Name          string        `json:"name"`
	IndexTemplate IndexTemplate `json:"index_template"`
}

type IndexTemplate struct {
	IndexPatterns []string `json:"index_patterns"`
}

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
	mockbeat.WaitForLogs(msg, 60*time.Second, fmt.Sprintf("error waiting for log: %s", msg))
	msg = "Template with name \\\"bla\\\" loaded."
	mockbeat.WaitForLogs(msg, 10*time.Second, fmt.Sprintf("error waiting for log: %s", msg))

	// check effective changes in ES
	req := fmt.Sprintf("%s/_index_template/%s", esUrl.String(), templateName)
	r, _ := http.Get(req)
	require.Equal(t, 200, r.StatusCode, "incorrect status code")
	body, _ := ioutil.ReadAll(r.Body)
	var m IndexTemplateResult
	json.Unmarshal(body, &m)
	require.Equal(t, len(m.IndexTemplates), 1)
}
