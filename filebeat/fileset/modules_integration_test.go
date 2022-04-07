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
// +build integration

package fileset

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v8/libbeat/esleg/eslegtest"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func makeTestInfo(version string) beat.Info {
	return beat.Info{
		IndexPrefix: "filebeat",
		Version:     version,
	}
}

func TestLoadPipeline(t *testing.T) {
	client := getTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}

	client.Request("DELETE", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)

	content := map[string]interface{}{
		"description": "describe pipeline",
		"processors": []interface{}{
			map[string]interface{}{
				"set": map[string]interface{}{
					"field": "foo",
					"value": "bar",
				},
			},
		},
	}

	log := logp.NewLogger(logName)
	err := LoadPipeline(client, "my-pipeline-id", content, false, log)
	require.NoError(t, err)

	status, _, err := client.Request("GET", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)

	// loading again shouldn't actually update the pipeline
	content["description"] = "describe pipeline 2"
	err = LoadPipeline(client, "my-pipeline-id", content, false, log)
	require.NoError(t, err)
	checkUploadedPipeline(t, client, "describe pipeline")

	// loading again updates the pipeline
	err = LoadPipeline(client, "my-pipeline-id", content, true, log)
	require.NoError(t, err)
	checkUploadedPipeline(t, client, "describe pipeline 2")
}

func checkUploadedPipeline(t *testing.T, client *eslegclient.Connection, expectedDescription string) {
	status, response, err := client.Request("GET", "/_ingest/pipeline/my-pipeline-id", "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, status)

	var res map[string]interface{}
	err = json.Unmarshal(response, &res)
	if assert.NoError(t, err) {
		assert.Equal(t, expectedDescription, res["my-pipeline-id"].(map[string]interface{})["description"], string(response))
	}
}

func TestSetupNginx(t *testing.T) {
	client := getTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}

	client.Request("DELETE", "/_ingest/pipeline/filebeat-5.2.0-nginx-access-default", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/filebeat-5.2.0-nginx-error-pipeline", "", nil, nil)

	modulesPath, err := filepath.Abs("../module")
	require.NoError(t, err)

	configs := []*ModuleConfig{
		{
			Module: "nginx",
			Filesets: map[string]*FilesetConfig{
				"error":  {},
				"access": {},
			},
		},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, makeTestInfo("5.2.0"))
	if err != nil {
		t.Fatal(err)
	}

	err = reg.LoadPipelines(client, false)
	if err != nil {
		t.Fatal(err)
	}

	status, _, _ := client.Request("GET", "/_ingest/pipeline/filebeat-5.2.0-nginx-access-pipeline", "", nil, nil)
	assert.Equal(t, 200, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/filebeat-5.2.0-nginx-error-pipeline", "", nil, nil)
	assert.Equal(t, 200, status)
}

func TestAvailableProcessors(t *testing.T) {
	client := getTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}
	// these exists on our integration test setup
	requiredProcessors := []ProcessorRequirement{
		{Name: "user_agent", Plugin: "ingest-user-agent"},
		{Name: "geoip", Plugin: "ingest-geoip"},
	}

	err := checkAvailableProcessors(client, requiredProcessors)
	require.NoError(t, err)

	// these don't exists on our integration test setup
	requiredProcessors = []ProcessorRequirement{
		{Name: "test", Plugin: "ingest-test"},
		{Name: "hello", Plugin: "ingest-hello"},
	}

	err = checkAvailableProcessors(client, requiredProcessors)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ingest-test")
	assert.Contains(t, err.Error(), "ingest-hello")
}

func hasIngest(client *eslegclient.Connection) bool {
	v := client.GetVersion()
	return v.Major >= 5
}

func hasIngestPipelineProcessor(client *eslegclient.Connection) bool {
	v := client.GetVersion()
	return v.Major > 6 || (v.Major == 6 && v.Minor >= 5)
}

func TestLoadMultiplePipelines(t *testing.T) {
	client := getTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}

	if !hasIngestPipelineProcessor(client) {
		t.Skip("Skip tests because ingest is missing the pipeline processor: ", client.GetVersion())
	}

	client.Request("DELETE", "/_ingest/pipeline/filebeat-6.6.0-foo-multi-pipeline", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/filebeat-6.6.0-foo-multi-json_logs", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/filebeat-6.6.0-foo-multi-plain_logs", "", nil, nil)

	modulesPath, err := filepath.Abs("../_meta/test/module")
	require.NoError(t, err)

	enabled := true
	disabled := false
	filesetConfigs := map[string]*FilesetConfig{
		"multi":    {Enabled: &enabled},
		"multibad": {Enabled: &disabled},
	}
	configs := []*ModuleConfig{
		{"foo", &enabled, filesetConfigs},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, makeTestInfo("6.6.0"))
	if err != nil {
		t.Fatal(err)
	}

	err = reg.LoadPipelines(client, false)
	if err != nil {
		t.Fatal(err)
	}

	status, _, _ := client.Request("GET", "/_ingest/pipeline/filebeat-6.6.0-foo-multi-pipeline", "", nil, nil)
	assert.Equal(t, 200, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/filebeat-6.6.0-foo-multi-json_logs", "", nil, nil)
	assert.Equal(t, 200, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/filebeat-6.6.0-foo-multi-plain_logs", "", nil, nil)
	assert.Equal(t, 200, status)
}

func TestLoadMultiplePipelinesWithRollback(t *testing.T) {
	client := getTestingElasticsearch(t)
	if !hasIngest(client) {
		t.Skip("Skip tests because ingest is missing in this elasticsearch version: ", client.GetVersion())
	}

	if !hasIngestPipelineProcessor(client) {
		t.Skip("Skip tests because ingest is missing the pipeline processor: ", client.GetVersion())
	}

	client.Request("DELETE", "/_ingest/pipeline/filebeat-6.6.0-foo-multibad-pipeline", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/filebeat-6.6.0-foo-multibad-json_logs", "", nil, nil)
	client.Request("DELETE", "/_ingest/pipeline/filebeat-6.6.0-foo-multibad-plain_logs_bad", "", nil, nil)

	modulesPath, err := filepath.Abs("../_meta/test/module")
	require.NoError(t, err)

	enabled := true
	disabled := false
	filesetConfigs := map[string]*FilesetConfig{
		"multi":    {Enabled: &disabled},
		"multibad": {Enabled: &enabled},
	}
	configs := []*ModuleConfig{
		{"foo", &enabled, filesetConfigs},
	}

	reg, err := newModuleRegistry(modulesPath, configs, nil, makeTestInfo("6.6.0"))
	if err != nil {
		t.Fatal(err)
	}

	err = reg.LoadPipelines(client, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_processor")

	status, _, _ := client.Request("GET", "/_ingest/pipeline/filebeat-6.6.0-foo-multibad-pipeline", "", nil, nil)
	assert.Equal(t, 404, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/filebeat-6.6.0-foo-multibad-json_logs", "", nil, nil)
	assert.Equal(t, 404, status)
	status, _, _ = client.Request("GET", "/_ingest/pipeline/filebeat-6.6.0-foo-multibad-plain_logs_bad", "", nil, nil)
	assert.Equal(t, 404, status)
}

func getTestingElasticsearch(t eslegtest.TestLogger) *eslegclient.Connection {
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:      eslegtest.GetURL(),
		Username: eslegtest.GetUser(),
		Password: eslegtest.GetPass(),
	})
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	conn.Encoder = eslegclient.NewJSONEncoder(nil, false)

	err = conn.Connect()
	if err != nil {
		t.Fatal(err)
		panic(err) // panic in case TestLogger did not stop test
	}

	return conn
}
