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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

// TemplateLoader is a subset of the Elasticsearch client API capable of
// loading the template.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

type Loader struct {
	config   TemplateConfig
	client   ESClient
	beatInfo beat.Info
	fields   []byte
}

// NewLoader creates a new template loader
func NewLoader(cfg *common.Config, client ESClient, beatInfo beat.Info, fields []byte) (*Loader, error) {
	config := DefaultConfig

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	return &Loader{
		config:   config,
		client:   client,
		beatInfo: beatInfo,
		fields:   fields,
	}, nil
}

// Load checks if the index mapping template should be loaded
// In case the template is not already loaded or overwriting is enabled, the
// template is written to index
func (l *Loader) Load() error {
	tmpl, err := New(l.beatInfo.Version, l.beatInfo.IndexPrefix, l.client.GetVersion(), l.config)
	if err != nil {
		return fmt.Errorf("error creating template instance: %v", err)
	}

	templateName := tmpl.GetName()
	if l.config.JSON.Enabled {
		templateName = l.config.JSON.Name
	}
	// Check if template already exist or should be overwritten
	exists := l.CheckTemplate(templateName)
	if !exists || l.config.Overwrite {

		version := l.client.GetVersion()
		logp.Info("Loading template for Elasticsearch version: %s", version.String())
		if l.config.Overwrite {
			logp.Info("Existing template will be overwritten, as overwrite is enabled.")
		}

		var template map[string]interface{}
		if l.config.JSON.Enabled {
			jsonPath := paths.Resolve(paths.Config, l.config.JSON.Path)
			if _, err := os.Stat(jsonPath); err != nil {
				return fmt.Errorf("error checking for json template: %s", err)
			}

			logp.Info("Loading json template from file %s", jsonPath)

			content, err := ioutil.ReadFile(jsonPath)
			if err != nil {
				return fmt.Errorf("error reading file. Path: %s, Error: %s", jsonPath, err)

			}
			err = json.Unmarshal(content, &template)
			if err != nil {
				return fmt.Errorf("could not unmarshal json template: %s", err)
			}
			// Load fields from path
		} else if l.config.Fields != "" {
			logp.Debug("template", "Load fields.yml from file: %s", l.config.Fields)

			fieldsPath := paths.Resolve(paths.Config, l.config.Fields)

			template, err = tmpl.LoadFile(fieldsPath)
			if err != nil {
				return fmt.Errorf("error creating template from file %s: %v", fieldsPath, err)
			}
		} else {
			logp.Debug("template", "Load default fields.yml")
			template, err = tmpl.LoadBytes(l.fields)
			if err != nil {
				return fmt.Errorf("error creating template: %v", err)
			}
		}

		err = l.LoadTemplate(templateName, template)
		if err != nil {
			return fmt.Errorf("could not load template. Elasticsearch returned: %v. Template is: %s", err, template)
		}

	} else {
		logp.Info("Template already exists and will not be overwritten.")
	}

	return nil
}

// LoadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use CheckTemplate prior to calling this method.
func (l *Loader) LoadTemplate(templateName string, template map[string]interface{}) error {
	logp.Debug("template", "Try loading template with name: %s", templateName)
	path := "/_template/" + templateName
	body, err := loadJSON(l.client, path, template)
	if err != nil {
		return fmt.Errorf("couldn't load template: %v. Response body: %s", err, body)
	}
	logp.Info("Elasticsearch template with name '%s' loaded", templateName)
	return nil
}

// CheckTemplate checks if a given template already exist. It returns true if
// and only if Elasticsearch returns with HTTP status code 200.
func (l *Loader) CheckTemplate(templateName string) bool {
	status, _, _ := l.client.Request("HEAD", "/_template/"+templateName, "", nil, nil)
	return status == 200
}

func loadJSON(client ESClient, path string, json map[string]interface{}) ([]byte, error) {
	params := esVersionParams(client.GetVersion())
	status, body, err := client.Request("PUT", path, "", params, json)
	if err != nil {
		return body, fmt.Errorf("couldn't load json. Error: %s", err)
	}
	if status > 300 {
		return body, fmt.Errorf("couldn't load json. Status: %v", status)
	}

	return body, nil
}

func esVersionParams(ver common.Version) map[string]string {
	if ver.Major == 6 && ver.Minor == 7 {
		return map[string]string{
			"include_type_name": "true",
		}
	}

	return nil
}
