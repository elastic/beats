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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test that beat stops in case elasticsearch index is modified and pattern not
func TestIndexModified(t *testing.T) {
	mockbeatConfigWithIndex := `
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
	mockbeatConfigWithES := `
mockbeat:
output:
  elasticsearch:
    hosts: %s
`
	esUrl := GetESURL(t, "http")
	cfg := fmt.Sprintf(mockbeatConfigWithES, esUrl.String())
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
}

// Test that beat stops in case elasticsearch index is modified and pattern not
func TestIndexModifiedNoPattern(t *testing.T) {
	cfg := `
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
	cfg := `
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
	mockbeatConfigWithES := `
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
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
}

// Test loading of json based template
func TestJsonTemplate(t *testing.T) {
	EnsureESIsRunning(t)
	_, err := os.Stat("../files/template.json")
	require.NoError(t, err)

	templateName := "bla"
	mockbeatConfigWithES := `
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
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	msg := "Loading json template from file"
	mockbeat.WaitForLogs(msg, 60*time.Second)
	msg = "Template with name \\\"bla\\\" loaded."
	mockbeat.WaitForLogs(msg, 60*time.Second)

	// check effective changes in ES
	indexURL, err := FormatIndexTemplateURL(t, esUrl, templateName)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	var m IndexTemplateResult
	err = json.Unmarshal(body, &m)
	require.NoError(t, err)
	require.Equal(t, len(m.IndexTemplates), 1)
}

// Test run cmd with default settings for template
func TestTemplateDefault(t *testing.T) {
	EnsureESIsRunning(t)

	mockbeatConfigWithES := `
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

	// make sure Datastream and Index aren't present
	dsURL, err := FormatDatastreamURL(t, esUrl, datastream)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, dsURL)
	require.NoError(t, err)

	indexURL, err := FormatIndexTemplateURL(t, esUrl, datastream)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, indexURL)
	require.NoError(t, err)

	// start mockbeat and wait for the relevant log lines
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	mockbeat.WaitForLogs("Template with name \\\"mockbeat-9.9.9\\\" loaded.", 20*time.Second)
	mockbeat.WaitForLogs("PublishEvents: 1 events have been published", 20*time.Second)

	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	var m IndexTemplateResult
	err = json.Unmarshal(body, &m)
	require.NoError(t, err)

	require.Equal(t, len(m.IndexTemplates), 1)
	require.Equal(t, datastream, m.IndexTemplates[0].Name)

	refreshURL := FormatRefreshURL(t, esUrl)
	require.NoError(t, err)
	status, body, err = HttpDo(t, http.MethodPost, refreshURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect http status")

	searchURL, err := FormatDataStreamSearchURL(t, esUrl, datastream)
	require.NoError(t, err)
	status, body, err = HttpDo(t, http.MethodGet, searchURL)
	require.NoError(t, err)
	var results SearchResult
	err = json.Unmarshal(body, &results)
	require.NoError(t, err)

	require.True(t, results.Hits.Total.Value > 0)
}

// Test run cmd does not load template when disabled in config
func TestTemplateDisabled(t *testing.T) {
	EnsureESIsRunning(t)

	mockbeatConfigWithES := `
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

	dsURL, err := FormatDatastreamURL(t, esUrl, datastream)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, dsURL)
	require.NoError(t, err)

	indexURL, err := FormatIndexTemplateURL(t, esUrl, datastream)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, indexURL)
	require.NoError(t, err)

	// start mockbeat and wait for the relevant log lines
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start()
	mockbeat.WaitForLogs("mockbeat start running.", 60*time.Second)
	mockbeat.WaitForLogs("PublishEvents: 1 events have been published", 20*time.Second)

	u := fmt.Sprintf("%s/_index_template/%s", esUrl.String(), datastream)
	r, _ := http.Get(u)
	require.Equal(t, 404, r.StatusCode, "incorrect status code")
}
