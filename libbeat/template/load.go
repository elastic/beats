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

package template

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/asset"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/ilm"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

// ESClient is a subset of the Elasticsearch client API capable of
// loading the template.
type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

//Loader interface for loading templates
type Loader interface {
	Load(config Config, ilmConfig ilm.Config) (bool, error)
}

//ESLoader holds all information necessary to write given templates to the configured output.
type ESLoader struct {
	client     ESClient
	beatInfo   beat.Info
	esVersion  common.Version
	ilmEnabled bool
	migration  bool
}

//StdoutLoader holds all information necessary to write given templates to the configured output.
type StdoutLoader struct {
	beatInfo  beat.Info
	migration bool
}

// NewESLoader creates a new template loader to ES
func NewESLoader(client ESClient, beatInfo beat.Info, migration bool) (Loader, error) {
	return &ESLoader{
		client:     client,
		beatInfo:   beatInfo,
		esVersion:  client.GetVersion(),
		ilmEnabled: ilm.EnabledFor(client),
		migration:  migration,
	}, nil
}

//NewStdoutLoader creates a new template loader to stdout
func NewStdoutLoader(beatInfo beat.Info, migration bool) (Loader, error) {
	return &StdoutLoader{beatInfo: beatInfo, migration: migration}, nil
}

// Load checks if the index mapping template should be loaded
// In case the template is not already loaded or overwriting is enabled, the
// template is written to the configured ES output.
func (l *ESLoader) Load(config Config, ilmConfig ilm.Config) (bool, error) {
	tmpl, err := template(l.beatInfo, l.esVersion, l.ilmEnabled, l.migration, config, ilmConfig)
	if err != nil || tmpl == nil {
		logp.Info("template not created")
		return false, err
	}

	// Check if template already exist or should be overwritten
	templateName := tmpl.GetName()
	if l.templateLoaded(templateName) && !config.Overwrite {
		logp.Info("Template %s already exists and will not be overwritten.", templateName)
		return false, nil
	}

	//loading template to ES
	body, err := buildBody(l.beatInfo, tmpl, config)
	if err != nil {
		return false, err
	}
	logp.Info("Loading template %s for Elasticsearch version: %s", templateName, l.esVersion.String())
	if err := l.loadTemplate(templateName, body); err != nil {
		return false, err
	}
	logp.Info("Template %s successfully loaded.", templateName)

	return true, nil
}

// loadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use templateLoaded prior to calling this method.
func (l *ESLoader) loadTemplate(templateName string, template common.MapStr) error {
	path := "/_template/" + templateName
	resp, err := l.client.LoadJSON(path, template)
	if err != nil {
		return fmt.Errorf("couldn't load template: %v. Response body: %s", err, resp)
	}
	return nil
}

// templateLoaded checks if a given template already exist. It returns true if
// and only if Elasticsearch returns with HTTP status code 200.
func (l *ESLoader) templateLoaded(templateName string) bool {
	if l.client == nil {
		return false
	}
	status, _, _ := l.client.Request("HEAD", "/_template/"+templateName, "", nil, nil)

	if status != 200 {
		return false
	}

	return true
}

//Load loads the configured templates to stdout
func (l *StdoutLoader) Load(config Config, ilmConfig ilm.Config) (bool, error) {
	//build template from config
	tmpl, err := template(l.beatInfo, common.Version{}, true, l.migration, config, ilmConfig)
	if err != nil || tmpl == nil {
		return false, err
	}

	//create body to print
	body, err := buildBody(l.beatInfo, tmpl, config)
	if err != nil {
		return false, err
	}

	str := body.StringToPrint()
	if _, err := os.Stdout.WriteString(str); err != nil {
		return false, fmt.Errorf("Error printing template: %v", err)
	}
	return true, nil
}

func template(info beat.Info, esVersion common.Version, ilmEnabled bool, migration bool, config Config, ilmConfig ilm.Config) (*Template, error) {
	if !config.Enabled {
		return nil, nil
	}

	//check if ilm related information needs to be updated
	var err error
	config, err = updateILM(ilmEnabled, config, ilmConfig)
	if err != nil {
		return nil, err
	}

	tmpl, err := New(info.Version, info.IndexPrefix, esVersion, config, migration)
	if err != nil {
		return nil, fmt.Errorf("error creating template instance: %v", err)
	}
	return tmpl, nil
}

func updateILM(ilmEnabled bool, config Config, ilmConfig ilm.Config) (Config, error) {
	if !ilmEnabled || ilmConfig.Enabled == ilm.ModeDisabled {
		return config, nil
	}

	if config.JSON.Enabled {
		return config, errors.Errorf("mixing template.json and ilm is not allowed %s", config.Name)
	}
	config.Pattern = fmt.Sprintf("%s*", ilmConfig.RolloverAlias)
	if config.Settings.Index == nil {
		config.Settings.Index = map[string]interface{}{}
	}
	config.Settings.Index["lifecycle"] = map[string]interface{}{
		"rollover_alias": ilmConfig.RolloverAlias,
		"name":           ilmConfig.Policy.Name,
	}
	return config, nil
}

func buildBody(info beat.Info, tmpl *Template, config Config) (common.MapStr, error) {

	if config.Overwrite {
		logp.Info("Existing template will be overwritten, as overwrite is enabled.")
	}

	var err error
	var body map[string]interface{}
	if config.JSON.Enabled {
		jsonPath := paths.Resolve(paths.Config, config.JSON.Path)
		if _, err = os.Stat(jsonPath); err != nil {
			return nil, fmt.Errorf("error checking file %s for template: %v", jsonPath, err)
		}

		logp.Info("Loading json template from file %s", jsonPath)

		content, err := ioutil.ReadFile(jsonPath)
		if err != nil {
			return nil, fmt.Errorf("error reading file %s for template: %v", jsonPath, err)

		}
		err = json.Unmarshal(content, &body)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal json template: %s", err)
		}

		// Load fields from path
	} else if config.Fields != "" {
		logp.Debug("template", "Load fields.yml from file: %s", config.Fields)

		fieldsPath := paths.Resolve(paths.Config, config.Fields)

		body, err = tmpl.LoadFile(fieldsPath)
		if err != nil {
			return nil, fmt.Errorf("error creating template from file %s: %v", fieldsPath, err)
		}

		// Load default fields
	} else {
		logp.Debug("template", "Load default fields.yml")
		fields, err := asset.GetFields(info.Beat)
		if err != nil {
			return nil, err
		}
		body, err = tmpl.LoadBytes(fields)
		if err != nil {
			return nil, fmt.Errorf("error creating template: %v", err)
		}
	}
	return body, nil
}
