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
	"io/ioutil"
	"net/http"
	"net/url"
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

func TestSetupIdxMgmt(t *testing.T) {
	EnsureESIsRunning(t)
	esURL := GetESURL(t, "http")
	dataStream := "mockbeat-9.9.9"
	policy := "mockbeat"
	t.Cleanup(func() {
		err := deleteDataStream(t, dataStream)
		if err != nil {
			t.Logf("error deleting data_stream %s: %s", dataStream, err)
		}
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
	t.Cleanup(func() {
		err := deleteDataStream(t, dataStream)
		if err != nil {
			t.Logf("error deleting data_stream %s: %s", dataStream, err)
		}
	})
	esURL := GetESURL(t, "http")
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

func isTemplateLoaded(t *testing.T, data_stream string) bool {
	esURL := GetESURL(t, "http")
	path, err := url.JoinPath("/_index_template", data_stream)
	require.NoError(t, err, "error building template path")
	esURL.Path = path
	esURL.User = url.UserPassword("admin", "testing")
	resp, err := http.Get(esURL.String())
	require.NoError(t, err, "error getting datastream")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "incorrect status code")

	body, _ := ioutil.ReadAll(resp.Body)
	var r IndexTemplateResult
	json.Unmarshal(body, &r)
	for _, t := range r.IndexTemplates {
		if t.Name == data_stream {
			return true
		}
	}
	return false
}

func isIndexPatternSet(t *testing.T, data_stream string) bool {
	esURL := GetESURL(t, "http")
	path, err := url.JoinPath("/_index_template", data_stream)
	require.NoError(t, err, "error building template path")
	esURL.Path = path
	esURL.User = url.UserPassword("admin", "testing")
	resp, err := http.Get(esURL.String())
	require.NoError(t, err, "error getting datastream")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "incorrect status code")

	body, _ := ioutil.ReadAll(resp.Body)
	var r IndexTemplateResult
	json.Unmarshal(body, &r)
	for _, t := range r.IndexTemplates {
		if t.Name == data_stream {
			for _, p := range t.IndexTemplate.IndexPatterns {
				if p == data_stream {
					return true
				}
			}
		}
	}
	return false
}

func isPolicyCreated(t *testing.T, policy string) bool {
	esURL := GetESURL(t, "http")
	path, err := url.JoinPath("/_ilm/policy/", policy)
	require.NoError(t, err, "error building policy path")
	esURL.Path = path
	esURL.User = url.UserPassword("admin", "testing")
	resp, err := http.Get(esURL.String())
	require.NoError(t, err, "error getting policy")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "incorrect status code")

	body, _ := ioutil.ReadAll(resp.Body)
	if !strings.Contains(string(body), "max_primary_shard_size\":\"50gb") {
		return false
	}
	if !strings.Contains(string(body), "max_age\":\"30d") {
		return false
	}
	return true
}

func deleteDataStream(t *testing.T, data_stream string) error {
	esURL := GetESURL(t, "http")
	path, err := url.JoinPath("/_data_stream", data_stream)
	if err != nil {
		return fmt.Errorf("error joining data_stream path: %w", err)
	}
	esURL.Path = path
	esURL.User = url.UserPassword("admin", "testing")
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", esURL.String(), nil)
	if err != nil {
		return fmt.Errorf("error making new delete request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error doing delete request: %w", err)
	}
	defer resp.Body.Close()
	if http.StatusOK != resp.StatusCode {
		return fmt.Errorf("status code was %d", resp.StatusCode)
	}
	return nil
}

// func deleteDataStream(t *testing.T, data_stream string) error {
// 	esURL := GetESURL(t, "http")
// 	path, err := url.JoinPath("/_data_stream", data_stream)
// 	if err != nil {
// 		return fmt.Errorf("error joining data_stream path: %w", err)
// 	}
// 	esURL.Path = path
// 	esURL.User = url.UserPassword("admin", "testing")
// 	client := &http.Client{}
// 	req, err := http.NewRequest("DELETE", esURL.String(), nil)
// 	if err != nil {
// 		return fmt.Errorf("error making new delete request: %w", err)
// 	}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return fmt.Errorf("error doing delete request: %w", err)
// 	}
// 	defer resp.Body.Close()
// 	if http.StatusOK != resp.StatusCode {
// 		return fmt.Errorf("status code was %d", resp.StatusCode)
// 	}
// 	return nil
// }
