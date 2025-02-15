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
	"strings"
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
	require.Eventually(t, func() bool {
		return mockbeat.LogMatch("doBulkRequest: [[:digit:]]+ events have been sent")
	}, 20*time.Second, 100*time.Millisecond, "looking for PublishEvents")

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
	require.Eventually(t, func() bool {
		return mockbeat.LogMatch("doBulkRequest: [[:digit:]]+ events have been sent")
	}, 20*time.Second, 100*time.Millisecond, "looking for PublishEvents")

	u := fmt.Sprintf("%s/_index_template/%s", esUrl.String(), datastream)
	r, _ := http.Get(u)
	require.Equal(t, 404, r.StatusCode, "incorrect status code")
}

func TestSetupCmd(t *testing.T) {
	EnsureESIsRunning(t)

	cfg := `
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
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	dataStreamURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, templateURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, policyURL)
		require.NoError(t, err)
	})
	// Make sure datastream, template and policy don't exist
	_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, templateURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, policyURL)
	require.NoError(t, err)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.String(), user, pass))
	mockbeat.Start("setup", "--index-management")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	// check template loaded
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	var r IndexTemplateResult
	err = json.Unmarshal(body, &r)
	require.NoError(t, err)
	var found bool
	for _, t := range r.IndexTemplates {
		if t.Name == dataStream {
			found = true
		}
	}
	require.Truef(t, found, "data stream should be in: %v", r.IndexTemplates)

	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	require.Truef(t, strings.Contains(string(body), "max_primary_shard_size\":\"50gb"), "primary shard not found in %s", string(body))

	require.Truef(t, strings.Contains(string(body), "max_age\":\"30d"), "max_age not found in %s", string(body))
}

func TestSetupCmdTemplateDisabled(t *testing.T) {
	EnsureESIsRunning(t)

	cfg := `
mockbeat:
output:
  elasticsearch:
    hosts: %s
    username: %s
    password: %s
    allow_older_versions: true
logging:
  level: debug
setup:
  template:
    enabled: false
`
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	dataStreamURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, templateURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, policyURL)
		require.NoError(t, err)
	})
	// Make sure datastream, template and policy don't exist
	_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, templateURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, policyURL)
	require.NoError(t, err)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.String(), user, pass))
	mockbeat.Start("setup", "--index-management")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	// check template didn't load
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status, "incorrect status code")

	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	require.Truef(t, strings.Contains(string(body), "max_primary_shard_size\":\"50gb"), "primary shard not found in %s", string(body))

	require.Truef(t, strings.Contains(string(body), "max_age\":\"30d"), "max_age not found in %s", string(body))
}

func TestSetupCmdTemplateWithOpts(t *testing.T) {
	EnsureESIsRunning(t)

	cfg := `
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
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	dataStreamURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, templateURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, policyURL)
		require.NoError(t, err)
	})
	// Make sure datastream, template and policy don't exist
	_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, templateURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, policyURL)
	require.NoError(t, err)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.String(), user, pass))
	mockbeat.Start("setup", "--index-management", "-E", "setup.ilm.enabled=false", "-E", "setup.template.settings.index.number_of_shards=2")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	// check template loaded
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "incorrect status code for :%s", templateURL.String())
	require.Truef(t, strings.Contains(string(body), "number_of_shards\":\"2"), "number of shards not found in %s", string(body))
}

func TestTemplateCreatedOnIlmPolicyCreated(t *testing.T) {
	EnsureESIsRunning(t)

	cfg := `
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
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()
	dataStreamURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, templateURL)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, policyURL)
		require.NoError(t, err)
	})
	// Make sure datastream, template and policy don't exist
	_, _, err = HttpDo(t, http.MethodDelete, dataStreamURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, templateURL)
	require.NoError(t, err)
	_, _, err = HttpDo(t, http.MethodDelete, policyURL)
	require.NoError(t, err)

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.String(), user, pass))
	mockbeat.Start("setup", "--index-management", "-E", "setup.ilm.enabled=false")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	// check template loaded
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	var r IndexTemplateResult
	err = json.Unmarshal(body, &r)
	require.NoError(t, err)
	var found bool
	for _, t := range r.IndexTemplates {
		if t.Name == dataStream {
			found = true
		}
	}
	require.Truef(t, found, "data stream should be in: %v", r.IndexTemplates)

	// check policy not created
	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusNotFound, status, "incorrect status code for: %s", policyURL.String())

	mockbeat.Start("setup", "--index-management", "-E", "setup.template.overwrite=false", "-E", "setup.template.settings.index.number_of_shards=2")
	procState, err = mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")

	// check policy created
	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	require.Truef(t, strings.Contains(string(body), "max_primary_shard_size\":\"50gb"), "primary shard not found in %s", string(body))

	require.Truef(t, strings.Contains(string(body), "max_age\":\"30d"), "max_age not found in %s", string(body))
}

func TestExportTemplate(t *testing.T) {
	cfg := `
mockbeat:
output:
  console:
    enabled: true
logging:
  level: debug
`

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start("export", "template")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("mockbeat-9.9.9", 5*time.Second)
}

func TestExportTemplateDisabled(t *testing.T) {
	cfg := `
mockbeat:
output:
  console:
    enabled: true
logging:
  level: debug
`

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start("export", "template", "-E", "setup.template.enabled=false")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("mockbeat-9.9.9", 5*time.Second)
}

func TestExportAbsolutePath(t *testing.T) {
	cfg := `
mockbeat:
output:
  console:
    enabled: true
logging:
  level: debug
`

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	output := filepath.Join(mockbeat.TempDir(), "template", "mockbeat-9.9.9.json")
	t.Cleanup(func() {
		os.Remove(output)
	})
	mockbeat.WriteConfigFile(cfg)
	mockbeat.Start("export", "template", "--dir", mockbeat.TempDir())
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains("Writing to", 5*time.Second)
	mockbeat.WaitFileContains(output, "mockbeat-9.9.9", 5*time.Second)
}
