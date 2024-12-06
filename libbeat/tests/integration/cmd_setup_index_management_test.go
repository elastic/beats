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
	"strings"
	"testing"

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
logging:
  level: debug
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
	t.Cleanup(func() {
		dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, dsURL)
		require.NoError(t, err)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, dataStream))
	require.True(t, isIndexPatternSet(t, "mockbeat-9.9.9"))
	require.True(t, isPolicyCreated(t, policy))
}

func TestSetupTemplateDisabled(t *testing.T) {
	EnsureESIsRunning(t)
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	esURL := GetESURL(t, "http")
	t.Cleanup(func() {
		dsURL, err := FormatDatastreamURL(t, esURL, dataStream)
		require.NoError(t, err)
		_, _, err = HttpDo(t, http.MethodDelete, dsURL)
		require.NoError(t, err)
	})
	mockbeat := NewBeat(t, "mockbeat", "../../libbeat.test")
	mockbeat.WriteConfigFile(fmt.Sprintf(IdxMgmtCfg, esURL.String()))
	mockbeat.Start("setup", "--index-management", "-v", "-e")
	procState, err := mockbeat.Process.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, procState.ExitCode(), "incorrect exit code")
	require.True(t, isTemplateLoaded(t, dataStream))
	require.True(t, isIndexPatternSet(t, "mockbeat-9.9.9"))
	require.True(t, isPolicyCreated(t, policy))
}

func isTemplateLoaded(t *testing.T, dataStream string) bool {
	esURL := GetESURL(t, "http")
	indexURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	var r IndexTemplateResult
	json.Unmarshal(body, &r)
	for _, t := range r.IndexTemplates {
		if t.Name == dataStream {
			return true
		}
	}
	return false
}

func isIndexPatternSet(t *testing.T, dataStream string) bool {
	esURL := GetESURL(t, "http")
	indexURL, err := FormatIndexTemplateURL(t, esURL, dataStream)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, indexURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	var r IndexTemplateResult
	json.Unmarshal(body, &r)
	for _, t := range r.IndexTemplates {
		if t.Name == dataStream {
			for _, p := range t.IndexTemplate.IndexPatterns {
				if p == dataStream {
					return true
				}
			}
		}
	}
	return false
}

func isPolicyCreated(t *testing.T, policy string) bool {
	esURL := GetESURL(t, "http")
	policyURL, err := FormatPolicyURL(t, esURL, policy)
	require.NoError(t, err)
	status, body, err := HttpDo(t, http.MethodGet, policyURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "incorrect status code")

	if !strings.Contains(string(body), "max_primary_shard_size\":\"50gb") {
		return false
	}
	if !strings.Contains(string(body), "max_age\":\"30d") {
		return false
	}
	return true
}
