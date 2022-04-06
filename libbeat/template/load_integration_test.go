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

package template

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegtest"
	"github.com/elastic/beats/v7/libbeat/version"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type testTemplate struct {
	t      *testing.T
	client ESClient
	common.MapStr
}

type testSetup struct {
	t      *testing.T
	client ESClient
	loader *ESLoader
	config TemplateConfig
}

func newTestSetup(t *testing.T, cfg TemplateConfig) *testSetup {
	if cfg.Name == "" {
		cfg.Name = fmt.Sprintf("load-test-%+v", rand.Int())
	}
	client := getTestingElasticsearch(t)
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}
	s := testSetup{t: t, client: client, loader: NewESLoader(client), config: cfg}
	client.Request("DELETE", "/_data_stream/"+cfg.Name, "", nil, nil)
	s.requireDataStreamDoesNotExist("")
	client.Request("DELETE", "/_index_template/"+cfg.Name, "", nil, nil)
	s.requireTemplateDoesNotExist("")
	return &s
}

func newTestSetupWithESClient(t *testing.T, client ESClient, cfg TemplateConfig) *testSetup {
	t.Helper()
	if cfg.Name == "" {
		cfg.Name = fmt.Sprintf("load-test-%+v", rand.Int())
	}
	return &testSetup{t: t, client: client, loader: NewESLoader(client), config: cfg}
}

func (ts *testSetup) mustLoadTemplate(body map[string]interface{}) {
	ts.t.Helper()
	err := ts.loader.loadTemplate(ts.config.Name, body)
	require.NoError(ts.t, err)
	ts.requireTemplateExists("")
}

func (ts *testSetup) loadFromFile(fileElems []string) error {
	ts.config.Fields = path(ts.t, fileElems)
	beatInfo := beat.Info{Version: version.GetDefaultVersion()}
	return ts.loader.Load(ts.config, beatInfo, nil, false)
}

func (ts *testSetup) mustLoadFromFile(fileElems []string) {
	require.NoError(ts.t, ts.loadFromFile(fileElems))
	ts.requireTemplateExists("")
}

func (ts *testSetup) load(fields []byte) error {
	beatInfo := beat.Info{Version: version.GetDefaultVersion()}
	return ts.loader.Load(ts.config, beatInfo, fields, false)
}

func (ts *testSetup) mustLoad(fields []byte) {
	require.NoError(ts.t, ts.load(fields))
	ts.requireTemplateExists("")
}

func (ts *testSetup) requireTemplateExists(name string) {
	if name == "" {
		name = ts.config.Name
	}
	exists, err := ts.loader.checkExistsTemplate(name)
	require.NoError(ts.t, err, "failed to query template status")
	require.True(ts.t, exists, "template must exist: %s", name)
}

func (ts *testSetup) cleanupDataStream(name string) {
	ts.client.Request("DELETE", "/_data_stream/"+name, "", nil, nil)
	ts.requireDataStreamDoesNotExist(name)
}

func (ts *testSetup) cleanupTemplate(name string) {
	ts.client.Request("DELETE", "/_index_template/"+name, "", nil, nil)
	ts.requireTemplateDoesNotExist(name)
}

func (ts *testSetup) requireTemplateDoesNotExist(name string) {
	if name == "" {
		name = ts.config.Name
	}
	exists, err := ts.loader.checkExistsTemplate(name)
	require.NoError(ts.t, err, "failed to query template status")
	require.False(ts.t, exists, "template must not exist")
}

func (ts *testSetup) requireDataStreamDoesNotExist(name string) {
	if name == "" {
		name = ts.config.Name
	}
	exists, err := ts.loader.checkExistsDatastream(name)
	require.NoError(ts.t, err, "failed to query data stream status")
	require.False(ts.t, exists, "data stream must not exist")
}

func (ts *testSetup) sendTestEvent() {
	evt := map[string]interface{}{
		"@timestamp": "2099-11-15T13:12:00",
		"message":    "my super important message",
	}
	c, _, err := ts.client.Request(http.MethodPut, "/"+ts.config.Name+"/_create/1", "", nil, evt)
	require.NoError(ts.t, err)
	require.Equal(ts.t, c, http.StatusCreated, "document must be created with id 1")

	// refresh index so the event becomes available immediately
	_, _, err = ts.client.Request(http.MethodPost, "/"+ts.config.Name+"/_refresh", "", nil, nil)
	require.NoError(ts.t, err)
}

// requireTestEventPresent validates that the event is available
// returns the backing index of the event
func (ts *testSetup) requireTestEventPresent() string {
	c, b, err := ts.client.Request("GET", "/"+ts.config.Name+"/_search", "", nil, nil)
	require.NoError(ts.t, err)
	require.Equal(ts.t, http.StatusOK, c)

	var resp eslegclient.SearchResults
	err = json.Unmarshal(b, &resp)
	require.Equal(ts.t, 1, resp.Hits.Total.Value, "the test event must be returned")

	idx := struct {
		Index string `json:"_index"`
	}{Index: ""}
	err = json.Unmarshal(resp.Hits.Hits[0], &idx)
	require.NoError(ts.t, err, "backing index name must be parsed")
	return idx.Index
}

func TestESLoader_Load(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		t.Run("loading disabled", func(t *testing.T) {
			setup := newTestSetup(t, TemplateConfig{Enabled: false})

			setup.load(nil)
			setup.requireTemplateDoesNotExist("")
		})

		t.Run("invalid version", func(t *testing.T) {
			setup := newTestSetup(t, TemplateConfig{Enabled: true})

			beatInfo := beat.Info{Version: "invalid"}
			err := setup.loader.Load(setup.config, beatInfo, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "version is not semver")
		})

		t.Run("no Elasticsearch client", func(t *testing.T) {
			setup := newTestSetupWithESClient(t, nil, TemplateConfig{Enabled: true})

			beatInfo := beat.Info{Version: "9.9.9"}
			err := setup.loader.Load(setup.config, beatInfo, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "can not load template without active Elasticsearch client")
		})

		t.Run("cannot check template", func(t *testing.T) {
			m := getMockElasticsearchClient(t, "HEAD", "/_index_template/", 500, []byte("cannot check template"))
			setup := newTestSetupWithESClient(t, m, TemplateConfig{Enabled: true})

			beatInfo := beat.Info{Version: "9.9.9"}
			err := setup.loader.Load(setup.config, beatInfo, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failure while checking if template exists", "Load must return error because template cannot be checked")
		})

		t.Run("cannot load template", func(t *testing.T) {
			m := getMockElasticsearchClient(t, "PUT", "/_index_template/", 503, []byte("cannot load template"))
			setup := newTestSetupWithESClient(t, m, TemplateConfig{Enabled: true, Overwrite: true})

			beatInfo := beat.Info{Version: "9.9.9"}
			err := setup.loader.Load(setup.config, beatInfo, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failed to load template", "Load must return error because we cannot load the index template")
		})

		t.Run("cannot check data stream", func(t *testing.T) {
			m := getMockElasticsearchClient(t, "GET", "/_data_stream/", 503, []byte("error checking data stream"))
			setup := newTestSetupWithESClient(t, m, TemplateConfig{Enabled: true, Overwrite: true})

			beatInfo := beat.Info{Version: "9.9.9"}
			err := setup.loader.Load(setup.config, beatInfo, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failed to check data stream", "Load must return error because data stream cannot be checked")
		})

		t.Run("cannot load data stream", func(t *testing.T) {
			m := getMockElasticsearchClient(t, "PUT", "/_data_stream/", 300, nil)
			setup := newTestSetupWithESClient(t, m, TemplateConfig{Enabled: true, Overwrite: true})

			beatInfo := beat.Info{Version: "9.9.9"}
			err := setup.loader.Load(setup.config, beatInfo, nil, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failed to put data stream", "Load must return error because data stream cannot be uploaded")
		})
	})

	t.Run("overwrite", func(t *testing.T) {
		// Setup create template with source enabled
		setup := newTestSetup(t, TemplateConfig{Enabled: true})
		setup.mustLoad(nil)

		// Add custom settings
		setup.config.Settings = TemplateSettings{Source: map[string]interface{}{"enabled": false}}

		t.Run("disabled", func(t *testing.T) {
			setup.load(nil)
			tmpl := getTemplate(t, setup.client, setup.config.Name)
			assert.Equal(t, true, tmpl.SourceEnabled())
		})

		t.Run("enabled", func(t *testing.T) {
			setup.config.Overwrite = true
			setup.load(nil)
			tmpl := getTemplate(t, setup.client, setup.config.Name)
			assert.Equal(t, false, tmpl.SourceEnabled())
		})

		t.Run("preserve existing data stream even if overwriting templates is allowed", func(t *testing.T) {
			fields, err := ioutil.ReadFile(path(t, []string{"testdata", "default_fields.yml"}))
			require.NoError(t, err)
			setup := newTestSetup(t, TemplateConfig{Enabled: true, Overwrite: true})
			setup.mustLoad(fields)

			exists, err := setup.loader.checkExistsDatastream(setup.config.Name)
			require.True(t, exists, "data stream must exits")

			// send test event before reloading the template
			setup.sendTestEvent()
			backingIdx := setup.requireTestEventPresent()

			setup.mustLoad(fields)

			newBackingIdx := setup.requireTestEventPresent()
			require.Equal(setup.t, backingIdx, newBackingIdx, "the event must be present in the same backing index")
		})
	})

	t.Run("json.name", func(t *testing.T) {
		nameJSON := "bar"

		setup := newTestSetup(t, TemplateConfig{Enabled: true})
		setup.mustLoad(nil)

		// Load Template with same name, but different JSON.name and ensure it is used
		setup.config.JSON = struct {
			Enabled      bool   `config:"enabled"`
			Path         string `config:"path"`
			Name         string `config:"name"`
			IsDataStream bool   `config:"data_stream"`
		}{Enabled: true, Path: path(t, []string{"testdata", "fields.json"}), Name: nameJSON, IsDataStream: false}
		setup.load(nil)
		setup.requireTemplateExists(nameJSON)
		setup.cleanupTemplate(nameJSON)
	})

	t.Run("load template successful", func(t *testing.T) {
		fields, err := ioutil.ReadFile(path(t, []string{"testdata", "default_fields.yml"}))
		require.NoError(t, err)
		for run, data := range map[string]struct {
			cfg        TemplateConfig
			fields     []byte
			fieldsPath string
			properties []string
		}{
			"default config with fields": {
				cfg:        TemplateConfig{Enabled: true},
				fields:     fields,
				properties: []string{"foo", "bar"},
			},
			"minimal template": {
				cfg:    TemplateConfig{Enabled: true},
				fields: nil,
			},
			"fields from file": {
				cfg:        TemplateConfig{Enabled: true, Fields: path(t, []string{"testdata", "fields.yml"})},
				fields:     fields,
				properties: []string{"object", "keyword", "alias", "migration_alias_false", "object_disabled", "@timestamp"},
			},
			"fields from json": {
				cfg: TemplateConfig{Enabled: true, JSON: struct {
					Enabled      bool   `config:"enabled"`
					Path         string `config:"path"`
					Name         string `config:"name"`
					IsDataStream bool   `config:"data_stream"`
				}{Enabled: true, Path: path(t, []string{"testdata", "fields.json"}), Name: "json-template", IsDataStream: false}},
				fields:     fields,
				properties: []string{"host_name"},
			},
			"fields from json with data stream": {
				cfg: TemplateConfig{Enabled: true, JSON: struct {
					Enabled      bool   `config:"enabled"`
					Path         string `config:"path"`
					Name         string `config:"name"`
					IsDataStream bool   `config:"data_stream"`
				}{Enabled: true, Path: path(t, []string{"testdata", "fields-data-stream.json"}), Name: "json-ds", IsDataStream: true}},
				fields:     fields,
				properties: []string{"host_name"},
			},
		} {
			t.Run(run, func(t *testing.T) {
				if data.cfg.JSON.Enabled {
					data.cfg.Name = data.cfg.JSON.Name
				}
				setup := newTestSetup(t, data.cfg)
				setup.mustLoad(data.fields)

				// Fetch properties
				tmpl := getTemplate(t, setup.client, setup.config.Name)
				val, err := tmpl.GetValue("template.mappings.properties")
				if data.properties == nil {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					p, ok := val.(map[string]interface{})
					require.True(t, ok)
					var properties []string
					for k := range p {
						properties = append(properties, k)
					}
					assert.ElementsMatch(t, properties, data.properties)
				}
				if !data.cfg.JSON.Enabled || data.cfg.JSON.IsDataStream {
					setup.cleanupDataStream(setup.config.Name)
				}
				setup.cleanupTemplate(setup.config.Name)
			})
		}
	})
}

func TestTemplate_LoadFile(t *testing.T) {
	setup := newTestSetup(t, TemplateConfig{Enabled: true})
	setup.mustLoadFromFile([]string{"..", "fields.yml"})
}

func TestLoadInvalidTemplate(t *testing.T) {
	setup := newTestSetup(t, TemplateConfig{})

	// Try to load invalid template
	template := map[string]interface{}{"json": "invalid"}
	err := setup.loader.loadTemplate(setup.config.Name, template)
	assert.Error(t, err)
	setup.requireTemplateDoesNotExist("")
}

// Tests loading the templates for each beat
func TestLoadBeatsTemplate_fromFile(t *testing.T) {
	beats := []string{
		"libbeat",
	}

	for _, beat := range beats {
		setup := newTestSetup(t, TemplateConfig{Name: beat, Enabled: true})
		setup.mustLoadFromFile([]string{"..", "..", beat, "fields.yml"})
	}
}

func TestTemplateSettings(t *testing.T) {
	settings := TemplateSettings{
		Index:  common.MapStr{"number_of_shards": 1},
		Source: common.MapStr{"enabled": false},
	}
	setup := newTestSetup(t, TemplateConfig{Settings: settings, Enabled: true})
	setup.mustLoadFromFile([]string{"..", "fields.yml"})

	// Check that it contains the mapping
	templateJSON := getTemplate(t, setup.client, setup.config.Name)
	assert.Equal(t, 1, templateJSON.NumberOfShards())
	assert.Equal(t, false, templateJSON.SourceEnabled())
}

var dataTests = []struct {
	data  common.MapStr
	error bool
}{
	{
		data: common.MapStr{
			"@timestamp": time.Now(),
			"keyword":    "test keyword",
			"array":      [...]int{1, 2, 3},
			"object": common.MapStr{
				"hello": "world",
			},
		},
		error: false,
	},
	{
		// Invalid array
		data: common.MapStr{
			"array": common.MapStr{
				"hello": "world",
			},
		},
		error: true,
	},
	{
		// Invalid object
		data: common.MapStr{
			"object": [...]int{1, 2, 3},
		},
		error: true,
	},
	{
		// tests enabled: false values
		data: common.MapStr{
			"@timestamp":     time.Now(),
			"array_disabled": [...]int{1, 2, 3},
			"object_disabled": common.MapStr{
				"hello": "world",
			},
		},
		error: false,
	},
}

// Tests if data can be loaded into elasticsearch with right types
func TestTemplateWithData(t *testing.T) {
	setup := newTestSetup(t, TemplateConfig{Enabled: true})
	setup.mustLoadFromFile([]string{"testdata", "fields.yml"})

	esClient := setup.client.(*eslegclient.Connection)
	for _, test := range dataTests {
		_, _, err := esClient.Index(setup.config.Name, "_doc", "", nil, test.data)
		if test.error {
			assert.Error(t, err)

		} else {
			assert.NoError(t, err)
		}
	}
}

func getTemplate(t *testing.T, client ESClient, templateName string) testTemplate {
	status, body, err := client.Request("GET", "/_index_template/"+templateName, "", nil, nil)
	require.NoError(t, err)
	require.Equal(t, status, 200)

	var response common.MapStr
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	require.NotNil(t, response)

	templates, _ := response.GetValue("index_templates")
	templatesList, _ := templates.([]interface{})
	templateElem := templatesList[0].(map[string]interface{})

	return testTemplate{
		t:      t,
		client: client,
		MapStr: common.MapStr(templateElem["index_template"].(map[string]interface{})),
	}
}

func (tt *testTemplate) SourceEnabled() bool {
	key := fmt.Sprintf("template.mappings._source.enabled")

	// _source.enabled is true if it's missing (default)
	b, _ := tt.HasKey(key)
	if !b {
		return true
	}

	val, err := tt.GetValue(key)
	if !assert.NoError(tt.t, err) {
		doc, _ := json.MarshalIndent(tt.MapStr, "", "    ")
		tt.t.Fatal(fmt.Sprintf("failed to read '%v' in %s", key, doc))
	}

	return val.(bool)
}

func (tt *testTemplate) NumberOfShards() int {
	val, err := tt.GetValue("template.settings.index.number_of_shards")
	require.NoError(tt.t, err)

	i, err := strconv.Atoi(val.(string))
	require.NoError(tt.t, err)
	return i
}

func path(t *testing.T, fileElems []string) string {
	fieldsPath, err := filepath.Abs(filepath.Join(fileElems...))
	require.NoError(t, err)
	return fieldsPath
}

func getTestingElasticsearch(t eslegtest.TestLogger) *eslegclient.Connection {
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:       eslegtest.GetURL(),
		Transport: httpcommon.DefaultHTTPTransportSettings(),
		Username:  eslegtest.GetUser(),
		Password:  eslegtest.GetPass(),
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

func getMockElasticsearchClient(t *testing.T, method, endpoint string, code int, body []byte) *eslegclient.Connection {
	server := esMock(t, method, endpoint, code, body)
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:       server.URL,
		Transport: httpcommon.DefaultHTTPTransportSettings(),
	})
	require.NoError(t, err)
	err = conn.Connect()
	require.NoError(t, err)
	return conn
}

func esMock(t *testing.T, method, endpoint string, code int, body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"version":{"number":"5.0.0"}}`))
			return
		}

		if r.Method == method && strings.HasPrefix(r.URL.Path, endpoint) {
			w.WriteHeader(code)
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
			return
		}

		c := 200
		// if we are checking if the data stream is available,
		// return 404 so the client will try to load it.
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/_data_stream") {
			c = 404
		}
		w.WriteHeader(c)
		if body != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(body)
		}
	}))
}
