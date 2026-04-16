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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var IdxMgmtCfg = `
mockbeat:
name:
logging:
  level: debug
queue.mem:
  events: 4096
  flush.min_events: 8
  flush.timeout: 0.1s
output.elasticsearch:
  hosts:
    - %s
  username: admin
  password: testing
  allow_older_versions: true
 `

func TestSetupIdxMgmt(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	deleteDataStream(t, esURL, dataStream)
	deleteIndexTemplate(t, esURL, dataStream)
	deleteILMPolicy(t, esURL, policy)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, dataStream)
		deleteIndexTemplate(t, esURL, dataStream)
		deleteILMPolicy(t, esURL, policy)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, dataStream))
	require.True(t, isIndexPatternSet(t, dataStream, dataStream))
	require.True(t, isPolicyCreated(t, policy))
}

func TestSetupTemplateDisabled(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	deleteDataStream(t, esURL, dataStream)
	deleteIndexTemplate(t, esURL, dataStream)
	deleteILMPolicy(t, esURL, policy)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, dataStream)
		deleteIndexTemplate(t, esURL, dataStream)
		deleteILMPolicy(t, esURL, policy)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.template.enabled=false")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateNotLoaded(t, dataStream))
	require.True(t, isPolicyCreated(t, policy))
}

func TestSetupILMDisabled(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	deleteDataStream(t, esURL, dataStream)
	deleteIndexTemplate(t, esURL, dataStream)
	deleteILMPolicy(t, esURL, policy)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, dataStream)
		deleteIndexTemplate(t, esURL, dataStream)
		deleteILMPolicy(t, esURL, policy)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.enabled=false")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, dataStream))
	require.True(t, isPolicyNotCreated(t, policy))
}

func TestSetupPolicyName(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	customPolicy := "mockbeat_bar"
	deleteDataStream(t, esURL, dataStream)
	deleteIndexTemplate(t, esURL, dataStream)
	deleteILMPolicy(t, esURL, customPolicy)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, dataStream)
		deleteIndexTemplate(t, esURL, dataStream)
		deleteILMPolicy(t, esURL, customPolicy)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.policy_name="+customPolicy)
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, dataStream))
	require.True(t, isPolicyCreated(t, customPolicy))
}

func TestSetupILMPolicyNoOverwrite(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	policyName := "mockbeat-test"
	deleteDataStream(t, esURL, dataStream)
	deleteIndexTemplate(t, esURL, dataStream)
	deleteILMPolicy(t, esURL, policyName)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, dataStream)
		deleteIndexTemplate(t, esURL, dataStream)
		deleteILMPolicy(t, esURL, policyName)
	})

	// Pre-condition: create a policy with only a delete phase (no hot phase).
	deleteOnlyPolicy := []byte(`{
		"policy": {
			"phases": {
				"delete": {
					"actions": {
						"delete": {}
					}
				}
			}
		}
	}`)
	putILMPolicy(t, esURL, policyName, deleteOnlyPolicy)

	phases := getILMPolicyPhases(t, esURL, policyName)
	require.Contains(t, phases, "delete", "expected delete phase before overwrite")
	require.NotContains(t, phases, "hot", "expected no hot phase before overwrite")

	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))

	// Run 1: overwrite=false — policy must not be overwritten.
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.enabled=true",
		"-E", "setup.ilm.overwrite=false",
		"-E", "setup.ilm.policy_name="+policyName)
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	phases = getILMPolicyPhases(t, esURL, policyName)
	require.Contains(t, phases, "delete", "expected delete phase after no-overwrite run")
	require.NotContains(t, phases, "hot", "expected no hot phase after no-overwrite run")

	// Run 2: overwrite=true — policy must be overwritten with hot phase.
	mockbeat2 := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat2.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat2.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.enabled=true",
		"-E", "setup.ilm.overwrite=true",
		"-E", "setup.ilm.policy_name="+policyName)
	err = mockbeat2.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat2.Cmd.ProcessState.ExitCode(), "incorrect exit code")

	phases = getILMPolicyPhases(t, esURL, policyName)
	require.NotContains(t, phases, "delete", "expected no delete phase after overwrite")
	require.Contains(t, phases, "hot", "expected hot phase after overwrite")
}

func TestSetupTemplateNameAndPatternOnILMDisabled(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	customTemplate := "mockbeat_foobar"
	policy := "mockbeat"
	deleteDataStream(t, esURL, customTemplate)
	deleteIndexTemplate(t, esURL, customTemplate)
	deleteILMPolicy(t, esURL, policy)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, customTemplate)
		deleteIndexTemplate(t, esURL, customTemplate)
		deleteILMPolicy(t, esURL, policy)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.enabled=false",
		"-E", "setup.template.name="+customTemplate,
		"-E", "setup.template.pattern="+customTemplate+"*")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, customTemplate))
	require.True(t, isIndexPatternSet(t, customTemplate, customTemplate+"*"))
	require.True(t, isPolicyNotCreated(t, policy))
}

func TestSetupTemplateWithOpts(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	deleteDataStream(t, esURL, dataStream)
	deleteIndexTemplate(t, esURL, dataStream)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, dataStream)
		deleteIndexTemplate(t, esURL, dataStream)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.enabled=false",
		"-E", "setup.template.settings.index.number_of_shards=2")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, dataStream))

	settings := getIndexTemplateSettings(t, esURL, dataStream)
	require.Equal(t, "2", settings["number_of_shards"], "unexpected number_of_shards")
}

func TestSetupOverwriteTemplateOnILMPolicyCreated(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	customTemplate := "mockbeat_foobar"
	policy := "mockbeat"
	deleteDataStream(t, esURL, customTemplate)
	deleteIndexTemplate(t, esURL, customTemplate)
	deleteILMPolicy(t, esURL, policy)
	t.Cleanup(func() {
		deleteDataStream(t, esURL, customTemplate)
		deleteIndexTemplate(t, esURL, customTemplate)
		deleteILMPolicy(t, esURL, policy)
	})

	// Run 1: ILM disabled — create template without ILM policy.
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.ilm.enabled=false",
		"-E", "setup.template.priority=160",
		"-E", "setup.template.name="+customTemplate,
		"-E", "setup.template.pattern="+customTemplate+"*")
	err := mockbeat.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, customTemplate))
	require.True(t, isPolicyNotCreated(t, policy))

	// Run 2: ILM enabled with overwrite=true — overwrites template and creates policy.
	mockbeat2 := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat2.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat2.Start("setup", "--index-management", "-v", "-e",
		"-E", "setup.template.overwrite=true",
		"-E", "setup.template.name="+customTemplate,
		"-E", "setup.template.pattern="+customTemplate+"*",
		"-E", "setup.template.settings.index.number_of_shards=2")
	err = mockbeat2.Cmd.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, mockbeat2.Cmd.ProcessState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, customTemplate))
	require.True(t, isPolicyCreated(t, policy))

	settings := getIndexTemplateSettings(t, esURL, customTemplate)
	require.Equal(t, "2", settings["number_of_shards"], "unexpected number_of_shards after overwrite")
}

// --- helpers ---

func isTemplateLoaded(t *testing.T, template string) bool {
	t.Helper()
	esURL := GetESURL(t, "http")
	indexURL, err := FormatIndexTemplateURL(t, esURL, template)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	if status == http.StatusNotFound {
		return false
	}
	require.Equalf(t, http.StatusOK, status, "unexpected status checking template %s, body: %s", template, string(body))

	var r IndexTemplateResult
	require.NoError(t, json.Unmarshal(body, &r))
	for _, entry := range r.IndexTemplates {
		if entry.Name == template {
			return true
		}
	}
	return false
}

func isTemplateNotLoaded(t *testing.T, template string) bool {
	t.Helper()
	esURL := GetESURL(t, "http")
	indexURL, err := FormatIndexTemplateURL(t, esURL, template)
	require.NoError(t, err)
	status, _, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	return status == http.StatusNotFound
}

func isIndexPatternSet(t *testing.T, template string, expectedPattern string) bool {
	t.Helper()
	esURL := GetESURL(t, "http")
	indexURL, err := FormatIndexTemplateURL(t, esURL, template)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "incorrect status code %d, body: %s", status, string(body))

	var r IndexTemplateResult
	require.NoError(t, json.Unmarshal(body, &r))
	for _, entry := range r.IndexTemplates {
		if entry.Name == template {
			for _, p := range entry.IndexTemplate.IndexPatterns {
				if p == expectedPattern {
					return true
				}
			}
		}
	}
	return false
}

func isPolicyCreated(t *testing.T, policy string) bool {
	t.Helper()
	esURL := GetESURL(t, "http")
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	if status == http.StatusNotFound {
		return false
	}
	require.Equalf(t, http.StatusOK, status, "unexpected status checking policy %s, status: %d, body: %s", policy, status, string(body))

	if !strings.Contains(string(body), `"max_primary_shard_size":"50gb"`) {
		return false
	}
	if !strings.Contains(string(body), `"max_age":"30d"`) {
		return false
	}
	return true
}

func isPolicyNotCreated(t *testing.T, policy string) bool {
	t.Helper()
	esURL := GetESURL(t, "http")
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	status, _, err := HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	return status == http.StatusNotFound
}

// putILMPolicy creates or updates an ILM policy via PUT.
func putILMPolicy(t *testing.T, esURL url.URL, policyName string, body []byte) {
	t.Helper()
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(30*time.Second))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, policyURL.String(), bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode, "failed to PUT ILM policy: %s, resp: %d, body: %s", policyName, resp.StatusCode, string(bodyBytes))
}

// getILMPolicyPhases returns the phases map for the named ILM policy.
func getILMPolicyPhases(t *testing.T, esURL url.URL, policyName string) map[string]any {
	t.Helper()
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "failed to GET ILM policy %s, body: %s", policyName, string(body))

	var result map[string]any
	require.NoError(t, json.Unmarshal(body, &result))
	policyObj, ok := result[policyName].(map[string]any)
	require.True(t, ok, "policy %s not found in response", policyName)
	policy, ok := policyObj["policy"].(map[string]any)
	require.True(t, ok, "policy.policy not found in response")
	phases, ok := policy["phases"].(map[string]any)
	require.True(t, ok, "policy.policy.phases not found in response")
	return phases
}

// deleteILMPolicy deletes an ILM policy, ignoring 404.
func deleteILMPolicy(t *testing.T, esURL url.URL, policyName string) {
	t.Helper()
	policyURL, err := FormatPolicyURL(t, esURL, policyName)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodDelete, policyURL)
	require.NoError(t, err)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("unexpected status %d deleting ILM policy %s, body: %s", status, policyName, string(body))
	}
}

// deleteIndexTemplate deletes an index template, ignoring 404.
func deleteIndexTemplate(t *testing.T, esURL url.URL, template string) {
	t.Helper()
	templateURL, err := FormatIndexTemplateURL(t, esURL, template)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodDelete, templateURL)
	require.NoError(t, err)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("unexpected status %d deleting index template %s, body: %s", status, template, string(body))
	}
}

// deleteDataStream deletes a data stream, ignoring 404.
func deleteDataStream(t *testing.T, esURL url.URL, dataStream string) {
	t.Helper()
	dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodDelete, dsURL)
	require.NoError(t, err)
	if status != http.StatusOK && status != http.StatusNotFound {
		t.Errorf("unexpected status %d deleting data stream %s, body: %s", status, dataStream, string(body))
	}
}

// getIndexTemplateSettings returns the settings.index map for the named template.
func getIndexTemplateSettings(t *testing.T, esURL url.URL, template string) map[string]any {
	t.Helper()
	indexURL, err := FormatIndexTemplateURL(t, esURL, template)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, status, "failed to GET index template %s, status: %d, body: %s", template, status, string(body))

	var r struct {
		IndexTemplates []struct {
			Name          string `json:"name"`
			IndexTemplate struct {
				Template struct {
					Settings struct {
						Index map[string]any `json:"index"`
					} `json:"settings"`
				} `json:"template"`
			} `json:"index_template"`
		} `json:"index_templates"`
	}
	require.NoError(t, json.Unmarshal(body, &r))
	for _, entry := range r.IndexTemplates {
		if entry.Name == template {
			return entry.IndexTemplate.Template.Settings.Index
		}
	}
	t.Fatalf("template %s not found in GET response", template)
	return nil
}
