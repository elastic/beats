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

// +build integration

package template

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/estest"
	"github.com/elastic/beats/libbeat/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTemplate struct {
	t      *testing.T
	client ESClient
	common.MapStr
}

func TestCheckTemplate(t *testing.T) {
	client := estest.GetTestingElasticsearch(t)
	err := client.Connect()
	require.NoError(t, err)

	loader := &ESLoader{
		client: client,
	}

	// Check for non existent template
	assert.False(t, loader.templateLoaded("libbeat-notexists"))
}

func TestESLoader_Load(t *testing.T) {

	data := []struct {
		name        string
		templateCfg Config
		ilmCfg      ilm.Config
		errMsg      string
	}{
		{
			name:        "disabled",
			templateCfg: Config{Name: "testbeat", Enabled: false},
			ilmCfg:      ilm.Config{},
		},
		{
			name:        "json file configured and update ilm",
			templateCfg: Config{Name: "testbeat", Enabled: true, JSON: JSON{Enabled: true, Path: ""}},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeEnabled},
			errMsg:      "mixing template.json and ilm",
		},
		{
			name:        "invalid json path",
			templateCfg: Config{Name: "testbeat", Enabled: true, JSON: JSON{Enabled: true, Path: ""}},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeDisabled},
			errMsg:      "error checking file",
		},
		{
			name:        "invalid json file",
			templateCfg: Config{Name: "testbeat", Enabled: true, JSON: JSON{Enabled: true, Path: "./testdata/invalid.json"}},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeDisabled},
			errMsg:      "could not unmarshal",
		}, {
			name:        "invalid fields file",
			templateCfg: Config{Name: "testbeat", Enabled: true, Fields: "./testdata/nonexisting.yml"},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeEnabled},
			errMsg:      "error creating template from file",
		},
	}
	for _, td := range data {

		client := estest.GetTestingElasticsearch(t)
		err := client.Connect()
		require.NoError(t, err)

		beatInfo := beat.Info{Version: "7.0.0", IndexPrefix: "testbeat"}

		//load to ES
		esLoader := ESLoader{
			client:     client,
			beatInfo:   beatInfo,
			ilmEnabled: true,
			esVersion:  client.GetVersion(),
		}

		t.Run(fmt.Sprintf("ES loader %s", td.name), func(t *testing.T) {
			templateName := getTemplateName(t, td.templateCfg.Name, beatInfo)

			// Delete template and make sure it was removed
			client.Request("DELETE", "/_template/"+templateName, "", nil, nil)
			assert.False(t, esLoader.templateLoaded(templateName))

			loaded, err := esLoader.Load(td.templateCfg, td.ilmCfg)
			assert.False(t, loaded)
			if td.errMsg == "" {
				assert.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), td.errMsg, fmt.Sprintf("Error `%s` doesn't contain expected error string", err.Error()))
			}
			assert.False(t, esLoader.templateLoaded(templateName))
		})

	}

}

func TestSuccesfullyLoaded(t *testing.T) {

	data := []struct {
		name        string
		templateCfg Config
		ilmCfg      ilm.Config
		ilmSection  interface{}
	}{
		{
			name:        "default",
			templateCfg: Config{Name: "testbeat", Enabled: true},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeDisabled},
		},
		{
			name:        "default with ilm",
			templateCfg: Config{Name: "testbeat", Enabled: true},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeEnabled, RolloverAlias: "testbeat-ilm", Policy: ilm.PolicyCfg{Name: "testpolicy"}},
			ilmSection:  map[string]interface{}(map[string]interface{}{"name": "testpolicy", "rollover_alias": "testbeat-ilm"}),
		},
		{
			name:        "json file",
			templateCfg: Config{Name: "testbeat", Enabled: true, JSON: JSON{Enabled: true, Path: "./testdata/template.json", Name: "testbeat"}},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeDisabled},
		},
		{
			name:        "fields file",
			templateCfg: Config{Name: "testbeat", Enabled: true, Fields: "./testdata/fields.yml"},
			ilmCfg:      ilm.Config{Enabled: ilm.ModeEnabled, RolloverAlias: "metricbeat-load", Policy: ilm.PolicyCfg{Name: "beatDefaultPolicy"}},
			ilmSection:  map[string]interface{}{"name": "beatDefaultPolicy", "rollover_alias": "metricbeat-load"},
		},
	}
	for _, d := range data {

		client := estest.GetTestingElasticsearch(t)
		err := client.Connect()
		require.NoError(t, err)

		beatInfo := beat.Info{Version: "7.0.0", IndexPrefix: "testbeat"}

		//load to ES
		esLoader := ESLoader{
			client:     client,
			beatInfo:   beatInfo,
			ilmEnabled: true,
			esVersion:  client.GetVersion(),
		}

		t.Run(d.name, func(t *testing.T) {
			templateName := getTemplateName(t, d.templateCfg.Name, beatInfo)

			// Delete template and make sure it was removed
			client.Request("DELETE", "/_template/"+templateName, "", nil, nil)
			assert.False(t, esLoader.templateLoaded(templateName))

			// load template
			loaded, err := esLoader.Load(d.templateCfg, d.ilmCfg)
			assert.True(t, loaded)
			assert.NoError(t, err)

			// don't load second time (overwrite is disabled by default)
			loaded, err = esLoader.Load(d.templateCfg, d.ilmCfg)
			assert.False(t, loaded)
			assert.NoError(t, err)

			// check ilm section
			templateJSON := getTemplate(t, client, templateName)
			val, err := templateJSON.GetValue("settings.index.lifecycle")

			if d.ilmSection != nil {
				require.NoError(t, err)
				assert.Equal(t, d.ilmSection, val)
			} else {
				require.Nil(t, val)
				require.Error(t, err)
			}

		})
	}

}

func TestLoadInvalidTemplate(t *testing.T) {
	// Setup ES
	client := estest.GetTestingElasticsearch(t)
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	templateName := "invalidtemplate"

	// Invalid Template
	template := map[string]interface{}{
		"json": "invalid",
	}

	loader := &ESLoader{client: client}

	// Try to load invalid template
	err := loader.loadTemplate(templateName, template)
	assert.Error(t, err)

	// Make sure template was not loaded
	assert.False(t, loader.templateLoaded(templateName))
}

// Tests loading the templates for each beat
func TestLoadBeatsTemplate(t *testing.T) {
	beats := []string{
		"auditbeat",
		"filebeat",
		"heartbeat",
		"journalbeat",
		"libbeat",
		"metricbeat",
		"packetbeat",
		"winlogbeat",
	}

	// Setup ES
	client := estest.GetTestingElasticsearch(t)
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	for _, beat := range beats {
		t.Run(beat, func(t *testing.T) {

			// Setup template configuration with fields.yml
			cfg := map[string]interface{}{"name": beat, "fields": fmt.Sprintf("../../%s/fields.yml", beat)}
			c, err := common.NewConfigFrom(cfg)
			require.NoError(t, err)

			var tmplCfg Config
			err = c.Unpack(&tmplCfg)
			require.NoError(t, err)

			// create new loader
			beatInfo := getBeatInfo(beat)
			loader := ESLoader{
				client:     client,
				beatInfo:   beatInfo,
				ilmEnabled: true,
				esVersion:  client.GetVersion(),
			}

			templateName := getTemplateName(t, tmplCfg.Name, beatInfo)

			// Delete template to ensure it isn't loaded
			client.Request("DELETE", "/_template/"+templateName, "", nil, nil)
			assert.False(t, loader.templateLoaded(templateName))

			// load template
			loaded, err := loader.Load(tmplCfg, ilm.Config{})
			assert.Nil(t, err)
			assert.True(t, loaded)

			// Make sure template was loaded
			assert.True(t, loader.templateLoaded(templateName))

		})
	}
}

func TestTemplateSettings(t *testing.T) {
	// Setup ES
	client := estest.GetTestingElasticsearch(t)
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	// Setup template configuration with fields.yml
	name := "testbeat"
	tmplCfg := Config{
		Name:    name,
		Enabled: true,
		Fields:  "../fields.yml",
		Settings: Settings{
			Index:  common.MapStr{"number_of_shards": 1},
			Source: common.MapStr{"enabled": false},
		},
	}

	beatInfo := getBeatInfo(name)
	templateName := getTemplateName(t, tmplCfg.Name, beatInfo)

	// Ensure template is not loaded
	client.Request("DELETE", "/_template/"+templateName, "", nil, nil)

	// create new loader
	loader := ESLoader{
		client:     client,
		beatInfo:   beatInfo,
		ilmEnabled: true,
		esVersion:  client.GetVersion(),
	}

	// Delete template
	client.Request("DELETE", "/_template/"+templateName, "", nil, nil)
	assert.False(t, loader.templateLoaded(templateName))

	// load template
	loaded, err := loader.Load(tmplCfg, ilm.Config{})
	assert.Nil(t, err)
	assert.True(t, loaded)

	// Check that it contains the mapping
	templateJSON := getTemplate(t, client, templateName)
	assert.Equal(t, 1, templateJSON.NumberOfShards())
	assert.Equal(t, false, templateJSON.SourceEnabled())

}

func TestOverwrite(t *testing.T) {
	// Setup ES
	client := estest.GetTestingElasticsearch(t)
	if err := client.Connect(); err != nil {
		t.Fatal(err)
	}

	beatInfo := beat.Info{
		Beat:        "testbeat",
		IndexPrefix: "testbeatidx",
		Version:     version.GetDefaultVersion(),
	}
	templateName := "testbeatidx-" + version.GetDefaultVersion()

	absPath, err := filepath.Abs("../")
	assert.NotNil(t, absPath)
	assert.Nil(t, err)

	loader := ESLoader{
		client:     client,
		beatInfo:   beatInfo,
		ilmEnabled: true,
		esVersion:  client.GetVersion(),
	}

	// Delete template to ensure it isn't loaded
	client.Request("DELETE", "/_template/"+templateName, "", nil, nil)
	assert.False(t, loader.templateLoaded(templateName))

	// Load template
	config := Config{
		Name:    templateName,
		Enabled: true,
		Fields:  absPath + "/fields.yml",
	}

	loaded, err := loader.Load(config, ilm.Config{})
	assert.NoError(t, err)
	assert.True(t, loaded)

	// Load template again, this time with custom settings
	config.Settings = Settings{
		Source: map[string]interface{}{
			"enabled": false,
		},
	}

	loaded, err = loader.Load(config, ilm.Config{})
	assert.NoError(t, err)
	assert.False(t, loaded)

	// Overwrite was not enabled, so the first version should still be there
	templateJSON := getTemplate(t, client, templateName)
	assert.Equal(t, true, templateJSON.SourceEnabled())

	// Load template again, this time with custom settings AND overwrite: true
	config = Config{
		Name:      templateName,
		Enabled:   true,
		Overwrite: true,
		Fields:    absPath + "/fields.yml",
		Settings: Settings{
			Source: map[string]interface{}{
				"enabled": false,
			},
		},
	}
	loaded, err = loader.Load(config, ilm.Config{})
	assert.NoError(t, err)
	assert.True(t, loaded)

	// Overwrite was enabled, so the custom setting should be there
	templateJSON = getTemplate(t, client, templateName)
	assert.Equal(t, false, templateJSON.SourceEnabled())
}

// Tests if data can be loaded into elasticsearch with right types
func TestTemplateWithData(t *testing.T) {
	var dataTests = []struct {
		data  common.MapStr
		error bool
	}{
		{
			data: common.MapStr{
				"keyword": "test keyword",
				"array":   [...]int{1, 2, 3},
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
				"array_disabled": [...]int{1, 2, 3},
				"object_disabled": common.MapStr{
					"hello": "world",
				},
			},
			error: false,
		},
	}

	fieldsPath, err := filepath.Abs("./testdata/fields.yml")
	assert.NotNil(t, fieldsPath)
	assert.Nil(t, err)

	// Setup ES
	client := estest.GetTestingElasticsearch(t)

	tmpl, err := New(version.GetDefaultVersion(), "testindex", client.GetVersion(), Config{Name: "testbeat"})
	assert.NoError(t, err)
	content, err := tmpl.LoadFile(fieldsPath)
	assert.NoError(t, err)

	loader := &ESLoader{client: client}

	// Delete template to ensure it isn't loaded
	client.Request("DELETE", "/_template/"+tmpl.GetName(), "", nil, nil)
	assert.False(t, loader.templateLoaded(tmpl.GetName()))

	// Load template
	err = loader.loadTemplate(tmpl.GetName(), content)
	assert.Nil(t, err)

	// Make sure template was loaded
	assert.True(t, loader.templateLoaded(tmpl.GetName()))

	for _, test := range dataTests {
		_, _, err = client.Index(tmpl.GetName(), "_doc", "", nil, test.data)
		if test.error {
			assert.NotNil(t, err)

		} else {
			assert.Nil(t, err)
		}
	}
}

// helper functions

func getTemplate(t *testing.T, client ESClient, templateName string) testTemplate {
	status, body, err := client.Request("GET", "/_template/"+templateName, "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, status, 200)

	var response common.MapStr
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	return testTemplate{
		t:      t,
		client: client,
		MapStr: common.MapStr(response[templateName].(map[string]interface{})),
	}
}

func (tt *testTemplate) SourceEnabled() bool {
	key := fmt.Sprintf("mappings._source.enabled")

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
	val, err := tt.GetValue("settings.index.number_of_shards")
	require.NoError(tt.t, err)

	i, err := strconv.Atoi(val.(string))
	require.NoError(tt.t, err)
	return i
}

func getBeatInfo(index string) beat.Info {
	return beat.Info{Version: version.GetDefaultVersion(), IndexPrefix: index}
}

//duplicate logic from template for convenience to get template name
func getTemplateName(t *testing.T, name string, info beat.Info) string {
	bv, err := common.NewVersion(info.Version)
	require.NoError(t, err)

	if name == "" {
		name = fmt.Sprintf("%s-%s", info.IndexPrefix, bv.String())
	}
	name, err = runFormatter(name, event(info.IndexPrefix, bv.String()))
	require.NoError(t, err)

	return name
}
