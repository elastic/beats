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
	"path/filepath"
	"strconv"
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
	client.Request("DELETE", "/_index_template/"+cfg.Name, "", nil, nil)
	s.requireTemplateDoesNotExist("")
	return &s
}

func (ts *testSetup) mustLoadTemplate(body map[string]interface{}) {
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
	})

	t.Run("json.name", func(t *testing.T) {
		nameJSON := "bar"

		setup := newTestSetup(t, TemplateConfig{Enabled: true})
		setup.mustLoad(nil)

		// Load Template with same name, but different JSON.name and ensure it is used
		setup.config.JSON = struct {
			Enabled bool   `config:"enabled"`
			Path    string `config:"path"`
			Name    string `config:"name"`
		}{Enabled: true, Path: path(t, []string{"testdata", "fields.json"}), Name: nameJSON}
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
					Enabled bool   `config:"enabled"`
					Path    string `config:"path"`
					Name    string `config:"name"`
				}{Enabled: true, Path: path(t, []string{"testdata", "fields.json"}), Name: "json-template"}},
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
