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

// ESClient is a subset of the Elasticsearch client API capable of
// loading the template.
type Client interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

//Loader interface for loading templates
type Loader interface {
	Load(config TemplateConfig) error
}

// NewLoader creates a new template loader
func NewLoader(
	client Client,
	info beat.Info,
	fields []byte,
	migration bool,
) (Loader, error) {
	if client == nil {
		return &stdoutLoader{info: info, fields: fields, migration: migration}, nil
	}
	return &esLoader{
		client:    client,
		info:      info,
		fields:    fields,
		migration: migration,
	}, nil
}

type esLoader struct {
	client    Client
	info      beat.Info
	fields    []byte
	migration bool
}

// Load checks if the index mapping template should be loaded
// In case the template is not already loaded or overwriting is enabled, the
// template is written to index
func (l *esLoader) Load(config TemplateConfig) error {
	//build template from config
	tmpl, err := template(config, l.info, l.client.GetVersion(), l.migration)
	if err != nil || tmpl == nil {
		return err
	}

	// Check if template already exist or should be overwritten
	templateName := tmpl.GetName()
	if config.JSON.Enabled {
		templateName = config.JSON.Name
	}

	if l.templateExists(templateName) && !config.Overwrite {
		logp.Info("Template %s already exists and will not be overwritten.", templateName)
		return nil
	}

	//loading template to ES
	body, err := buildBody(l.info, tmpl, config, l.fields)
	if err != nil {
		return err
	}
	if err := l.loadTemplate(templateName, body); err != nil {
		return fmt.Errorf("could not load template. Elasticsearch returned: %v. Template is: %s", err, body.StringToPrint())
	}
	logp.Info("template with name '%s' loaded.", templateName)
	return nil
}

// LoadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use CheckTemplate prior to calling this method.
func (l *esLoader) loadTemplate(templateName string, template map[string]interface{}) error {
	logp.Info("Try loading template %s to Elasticsearch", templateName)
	path := "/_template/" + templateName
	resp, err := loadJSON(l.client, path, template)
	if err != nil {
		return fmt.Errorf("couldn't load template: %v. Response body: %s", err, resp)
	}
	return nil
}

// exists checks if a given template already exist. It returns true if
// and only if Elasticsearch returns with HTTP status code 200.
func (l *esLoader) templateExists(templateName string) bool {
	if l.client == nil {
		return false
	}
	status, _, _ := l.client.Request("HEAD", "/_template/"+templateName, "", nil, nil)

	if status != 200 {
		return false
	}

	return true
}

type stdoutLoader struct {
	info      beat.Info
	fields    []byte
	migration bool
}

func (l *stdoutLoader) Load(config TemplateConfig) error {
	//build template from config
	tmpl, err := template(config, l.info, common.Version{}, l.migration)
	if err != nil || tmpl == nil {
		return err
	}

	//create body to print
	body, err := buildBody(l.info, tmpl, config, l.fields)
	if err != nil {
		return err
	}

	p := common.MapStr{tmpl.name: body}
	str := fmt.Sprintf("%s\n", p.StringToPrint())
	if _, err := os.Stdout.WriteString(str); err != nil {
		return fmt.Errorf("error printing template: %v", err)
	}
	return nil
}

func template(config TemplateConfig, info beat.Info, esVersion common.Version, migration bool) (*Template, error) {
	if !config.Enabled {
		logp.Info("template config not enabled")
		return nil, nil
	}
	tmpl, err := New(info.Version, info.IndexPrefix, esVersion, config, migration)
	if err != nil {
		return nil, fmt.Errorf("error creating template instance: %v", err)
	}
	return tmpl, nil
}

func buildBody(info beat.Info, tmpl *Template, config TemplateConfig, fields []byte) (common.MapStr, error) {
	if config.Overwrite {
		logp.Info("Existing template will be overwritten, as overwrite is enabled.")
	}

	var err error
	var body map[string]interface{}
	if config.JSON.Enabled {
		jsonPath := paths.Resolve(paths.Config, config.JSON.Path)
		if _, err = os.Stat(jsonPath); err != nil {
			return nil, fmt.Errorf("error checking json file %s for template: %v", jsonPath, err)
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
		body, err = tmpl.LoadBytes(fields)
		if err != nil {
			return nil, fmt.Errorf("error creating template: %v", err)
		}
	}
	return body, nil
}

func loadJSON(client Client, path string, json map[string]interface{}) ([]byte, error) {
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
