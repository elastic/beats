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

const ilmESConfig = `
mockbeat:
name:
output:
  elasticsearch:
    hosts:
      - %s
    username: %s
    password: %s
    allow_older_versions: true
logging:
  level: debug
`

const ilmESConfigILMDisabled = `
mockbeat:
name:
output:
  elasticsearch:
    hosts:
      - %s
    username: %s
    password: %s
    allow_older_versions: true
setup.ilm:
  enabled: false
logging:
  level: debug
`

const ilmESConfigCustomPolicy = `
mockbeat:
name:
output:
  elasticsearch:
    hosts:
      - %s
    username: %s
    password: %s
    allow_older_versions: true
setup.ilm:
  enabled: true
  policy_name: %s
logging:
  level: debug
`

const ilmConsoleConfig = `
mockbeat:
output:
  console:
    enabled: true
logging:
  level: debug
`

// TestILMDefault verifies that running mockbeat with default settings creates
// the ILM policy, data stream, and writes events to the data stream.
func TestILMDefault(t *testing.T) {
	EnsureESIsRunning(t)

	const (
		dataStream = "mockbeat-9.9.9"
		policyName = "mockbeat"
	)

	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)

	// Clean up before and after
	_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
	_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
	_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	t.Cleanup(func() {
		_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
		_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
		_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	})

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(ilmESConfig, esURL.Host, user, pass))
	mockbeat.Start()
	mockbeat.WaitLogsContains("mockbeat start running.", 60*time.Second)
	mockbeat.WaitLogsContains("lifecycle policy", 30*time.Second)
	require.Eventually(t, func() bool {
		return mockbeat.LogMatch("doBulkRequest: [[:digit:]]+ events have been sent")
	}, 30*time.Second, 100*time.Millisecond, "waiting for events to be sent")
	mockbeat.Stop()

	// Assert data stream created
	status, body, err := HttpDo(t, http.MethodGet, dsURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "data stream should exist. body: %s", string(body))

	// Assert ILM policy created with expected settings
	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "ILM policy should exist. body: %s", string(body))
	require.Containsf(t, string(body), `max_primary_shard_size":"50gb`, "primary shard size not found in %s", string(body))
	require.Containsf(t, string(body), `max_age":"30d`, "max_age not found in %s", string(body))

	// Assert docs written to data stream
	refreshURL := FormatRefreshURL(t, esURL)
	_, _, err = HttpDo(t, http.MethodPost, refreshURL)
	require.NoError(t, err)
	searchURL, err := FormatDataStreamSearchURL(t, esURL, dataStream)
	require.NoError(t, err)
	_, searchBody, err := HttpDo(t, http.MethodGet, searchURL)
	require.NoError(t, err)
	var resp struct {
		Hits struct {
			Total struct{ Value int } `json:"total"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal(searchBody, &resp), "unmarshal search body: %s", string(searchBody))
	require.Greater(t, resp.Hits.Total.Value, 0, "no documents found in data stream: %s", string(searchBody))
}

// TestILMDisabled verifies that with ILM disabled:
//   - the index template is loaded
//   - the ILM policy is NOT created
//   - events are still written to the data stream
func TestILMDisabled(t *testing.T) {
	EnsureESIsRunning(t)

	const (
		dataStream = "mockbeat-9.9.9"
		policyName = "mockbeat"
	)

	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)

	_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
	_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
	_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	t.Cleanup(func() {
		_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
		_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
		_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	})

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(ilmESConfigILMDisabled, esURL.Host, user, pass))
	mockbeat.Start()
	mockbeat.WaitLogsContains("mockbeat start running.", 60*time.Second)
	require.Eventually(t, func() bool {
		return mockbeat.LogMatch("doBulkRequest: [[:digit:]]+ events have been sent")
	}, 30*time.Second, 100*time.Millisecond, "waiting for events to be sent")
	mockbeat.Stop()

	// Assert index template is loaded
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "index template should exist. body: %s", string(body))

	// Assert ILM policy is NOT created
	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusNotFound, status, "ILM policy should not exist when ILM is disabled. body: %s", string(body))

	// Assert docs written to data stream
	refreshURL := FormatRefreshURL(t, esURL)
	_, _, err = HttpDo(t, http.MethodPost, refreshURL)
	require.NoError(t, err)
	searchURL, err := FormatDataStreamSearchURL(t, esURL, dataStream)
	require.NoError(t, err)
	_, searchBody, err := HttpDo(t, http.MethodGet, searchURL)
	require.NoError(t, err)
	var resp struct {
		Hits struct {
			Total struct{ Value int } `json:"total"`
		} `json:"hits"`
	}
	require.NoError(t, json.Unmarshal(searchBody, &resp), "unmarshal search body: %s", string(searchBody))
	require.Greater(t, resp.Hits.Total.Value, 0, "no documents found in data stream: %s", string(searchBody))
}

// TestILMCustomPolicyName verifies that a custom ILM policy name can be
// configured when running mockbeat.
func TestILMCustomPolicyName(t *testing.T) {
	EnsureESIsRunning(t)

	const (
		dataStream = "mockbeat-9.9.9"
		policyName = "mockbeat_foo"
	)

	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)

	_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
	_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
	_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	t.Cleanup(func() {
		_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
		_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
		_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	})

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(ilmESConfigCustomPolicy, esURL.Host, user, pass, policyName))
	mockbeat.Start()
	mockbeat.WaitLogsContains("mockbeat start running.", 60*time.Second)
	mockbeat.WaitLogsContains("lifecycle policy", 30*time.Second)
	require.Eventually(t, func() bool {
		return mockbeat.LogMatch("doBulkRequest: [[:digit:]]+ events have been sent")
	}, 30*time.Second, 100*time.Millisecond, "waiting for events to be sent")
	mockbeat.Stop()

	// Assert index template is loaded
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "index template should exist. body: %s", string(body))

	// Assert custom policy created
	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "custom ILM policy should exist. body: %s", string(body))
	require.Containsf(t, string(body), `max_primary_shard_size":"50gb`, "primary shard size not found in %s", string(body))
	require.Containsf(t, string(body), `max_age":"30d`, "max_age not found in %s", string(body))
}

// TestSetupILMCustomPolicyName verifies that setup --index-management with a
// custom policy name creates the policy with the specified name.
func TestSetupILMCustomPolicyName(t *testing.T) {
	EnsureESIsRunning(t)

	const (
		dataStream = "mockbeat-9.9.9"
		policyName = "mockbeat_bar"
	)

	esURL := GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)
	templateURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)

	_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
	_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
	_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	t.Cleanup(func() {
		_, _, _ = HttpDo(t, http.MethodDelete, dsURL)
		_, _, _ = HttpDo(t, http.MethodDelete, templateURL)
		_, _, _ = HttpDo(t, http.MethodDelete, policyURL)
	})

	cfg := `
mockbeat:
name:
output:
  elasticsearch:
    hosts:
      - %s
    username: %s
    password: %s
    allow_older_versions: true
logging:
  level: debug
`
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(cfg, esURL.Host, user, pass))
	mockbeat.Start("setup", "--index-management", "-E", "setup.ilm.policy_name="+policyName)
	err = mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	// Assert index template is loaded
	status, body, err := HttpDo(t, http.MethodGet, templateURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "index template should exist. body: %s", string(body))

	// Assert custom policy created
	status, body, err = HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "custom ILM policy should exist. body: %s", string(body))
	require.Containsf(t, string(body), `max_primary_shard_size":"50gb`, "primary shard size not found in %s", string(body))
	require.Containsf(t, string(body), `max_age":"30d`, "max_age not found in %s", string(body))
}

// TestExportILMPolicy verifies that export ilm-policy outputs valid policy JSON
// containing the expected rollover settings.
func TestExportILMPolicy(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(ilmConsoleConfig)
	mockbeat.Start("export", "ilm-policy")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains(`"max_age": "30d"`, 5*time.Second)
	mockbeat.WaitStdOutContains(`"max_primary_shard_size": "50gb"`, 5*time.Second)
}

// TestExportILMPolicyILMDisabled verifies that export ilm-policy works and
// outputs valid policy JSON even when ILM is disabled in config.
func TestExportILMPolicyILMDisabled(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(ilmConsoleConfig)
	mockbeat.Start("export", "ilm-policy", "-E", "setup.ilm.enabled=false")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains(`"max_age": "30d"`, 5*time.Second)
	mockbeat.WaitStdOutContains(`"max_primary_shard_size": "50gb"`, 5*time.Second)
}

// TestExportILMPolicyCustomName verifies that export ilm-policy works when a
// custom policy name is configured.
func TestExportILMPolicyCustomName(t *testing.T) {
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(ilmConsoleConfig)
	mockbeat.Start("export", "ilm-policy", "-E", "setup.ilm.policy_name=foo")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	mockbeat.WaitStdOutContains(`"max_age": "30d"`, 5*time.Second)
	mockbeat.WaitStdOutContains(`"max_primary_shard_size": "50gb"`, 5*time.Second)
}

// TestExportILMPolicyToAbsoluteDir verifies that export ilm-policy writes the
// policy to a file when an absolute directory path is specified via --dir.
func TestExportILMPolicyToAbsoluteDir(t *testing.T) {
	const policyName = "mockbeat"

	exportDir := t.TempDir()
	policyFile := filepath.Join(exportDir, "policy", policyName+".json")

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(ilmConsoleConfig)
	mockbeat.Start("export", "ilm-policy", "--dir", exportDir)
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	data, err := os.ReadFile(policyFile)
	require.NoErrorf(t, err, "policy file should be created at %s", policyFile)
	require.Contains(t, string(data), `"max_primary_shard_size": "50gb"`)
	require.Contains(t, string(data), `"max_age": "30d"`)
}

// TestExportILMPolicyToRelativeDir verifies that export ilm-policy writes the
// policy to a file when a relative directory path is specified via --dir.
func TestExportILMPolicyToRelativeDir(t *testing.T) {
	const policyName = "mockbeat"

	cwd, err := os.Getwd()
	require.NoError(t, err)

	absExportDir := t.TempDir()
	relExportDir, err := filepath.Rel(cwd, absExportDir)
	require.NoError(t, err)

	policyFile := filepath.Join(absExportDir, "policy", policyName+".json")

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(ilmConsoleConfig)
	mockbeat.Start("export", "ilm-policy", "--dir", relExportDir)
	err = mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	data, err := os.ReadFile(policyFile)
	require.NoErrorf(t, err, "policy file should be created at %s", policyFile)
	require.Contains(t, string(data), `"max_primary_shard_size": "50gb"`)
	require.Contains(t, string(data), `"max_age": "30d"`)
}
