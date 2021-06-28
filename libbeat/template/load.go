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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
)

var (
	templateLoaderPath = map[IndexTemplateType]string{
		IndexTemplateLegacy:    "/_template/",
		IndexTemplateComponent: "/_component_template/",
		IndexTemplateIndex:     "/_index_template/",
	}
)

// Loader interface for loading templates.
type Loader interface {
	Load(config TemplateConfig, info beat.Info, fields []byte, migration bool) error
}

// ESLoader implements Loader interface for loading templates to Elasticsearch.
type ESLoader struct {
	client  ESClient
	builder *templateBuilder
	log     *logp.Logger
}

// ESClient is a subset of the Elasticsearch client API capable of
// loading the template.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

// FileLoader implements Loader interface for loading templates to a File.
type FileLoader struct {
	client  FileClient
	builder *templateBuilder
	log     *logp.Logger
}

// FileClient defines the minimal interface required for the FileLoader
type FileClient interface {
	GetVersion() common.Version
	Write(component string, name string, body string) error
}

type StatusError struct {
	status int
}

type templateBuilder struct {
	log *logp.Logger
}

// NewESLoader creates a new template loader for ES
func NewESLoader(client ESClient) *ESLoader {
	return &ESLoader{client: client, builder: newTemplateBuilder(), log: logp.NewLogger("template_loader")}
}

// NewFileLoader creates a new template loader for the given file.
func NewFileLoader(c FileClient) *FileLoader {
	return &FileLoader{client: c, builder: newTemplateBuilder(), log: logp.NewLogger("file_template_loader")}
}

func newTemplateBuilder() *templateBuilder {
	return &templateBuilder{log: logp.NewLogger("template")}
}

// Load checks if the index mapping template should be loaded.
// In case the template is not already loaded or overwriting is enabled, the
// template is built and written to index.
func (l *ESLoader) Load(config TemplateConfig, info beat.Info, fields []byte, migration bool) error {
	if l.client == nil {
		return errors.New("can not load template without active Elasticsearch client")
	}

	// build template from config
	tmpl, err := l.builder.template(config, info, l.client.GetVersion(), migration)
	if err != nil || tmpl == nil {
		return err
	}

	// Check if template already exist or should be overwritten
	templateName := tmpl.GetName()
	if config.JSON.Enabled {
		templateName = config.JSON.Name
	}

	exists, err := l.templateExists(templateName, config.Type)
	if err != nil {
		return fmt.Errorf("failure while checking if template exists: %w", err)
	}

	if exists && !config.Overwrite {
		l.log.Infof("Template %q already exists and will not be overwritten.", templateName)
		return nil
	}

	// loading template to ES
	body, err := l.builder.buildBody(tmpl, config, fields)
	if err != nil {
		return err
	}
	if err := l.loadTemplate(templateName, config.Type, body); err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}
	l.log.Infof("Template with name %q loaded.", templateName)
	return nil
}

// loadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use CheckTemplate prior to calling this method.
func (l *ESLoader) loadTemplate(templateName string, templateType IndexTemplateType, template map[string]interface{}) error {
	l.log.Infof("Try loading template %s to Elasticsearch", templateName)
	clientVersion := l.client.GetVersion()
	path := templateLoaderPath[templateType] + templateName
	params := esVersionParams(clientVersion)
	status, body, err := l.client.Request("PUT", path, "", params, template)
	if err != nil {
		return fmt.Errorf("couldn't load template: %v. Response body: %s", err, body)
	}
	if status > http.StatusMultipleChoices { //http status 300
		return fmt.Errorf("couldn't load json. Status: %v", status)
	}
	return nil
}

func (l *ESLoader) templateExists(templateName string, templateType IndexTemplateType) (bool, error) {
	if templateType == IndexTemplateComponent {
		return l.checkExistsComponentTemplate(templateName)
	}
	return l.checkExistsTemplate(templateName)
}

// existsTemplate checks if a given template already exist, using the
// `_cat/templates/<name>` API.
//
// An error is returned if the loader failed to execute the request, or a
// status code indicating some problems is encountered.
func (l *ESLoader) checkExistsTemplate(name string) (bool, error) {
	status, body, err := l.client.Request("GET", "/_cat/templates/"+name, "", nil, nil)
	if err != nil {
		return false, err
	}

	// Elasticsearch API returns 200, even if the template does not exists. We
	// need to validate the body to be sure the template is actually known. Any
	// status code other than 200 will be treated as error.
	if status != http.StatusOK {
		return false, &StatusError{status: status}
	}
	return strings.Contains(string(body), name), nil
}

// existsComponentTemplate checks if a component template exists by querying
// the `_component_template/<name>` API.
//
// The resource is assumed as present if a 200 OK status is returned and missing if a 404 is returned.
// Other status codes or IO errors during the request are reported as error.
func (l *ESLoader) checkExistsComponentTemplate(name string) (bool, error) {
	status, _, err := l.client.Request("GET", "/_component_template/"+name, "", nil, nil)

	switch status {
	case http.StatusNotFound:
		return false, nil
	case http.StatusOK:
		return true, nil
	default:
		if err == nil {
			err = &StatusError{status: status}
		}
		return false, err
	}
}

// Load reads the template from the config, creates the template body and prints it to the configured file.
func (l *FileLoader) Load(config TemplateConfig, info beat.Info, fields []byte, migration bool) error {
	//build template from config
	tmpl, err := l.builder.template(config, info, l.client.GetVersion(), migration)
	if err != nil || tmpl == nil {
		return err
	}

	//create body to print
	body, err := l.builder.buildBody(tmpl, config, fields)
	if err != nil {
		return err
	}

	str := fmt.Sprintf("%s\n", body.StringToPrint())
	if err := l.client.Write("template", tmpl.name, str); err != nil {
		return fmt.Errorf("error printing template: %v", err)
	}
	return nil
}

func (b *templateBuilder) template(config TemplateConfig, info beat.Info, esVersion common.Version, migration bool) (*Template, error) {
	if !config.Enabled {
		b.log.Info("template config not enabled")
		return nil, nil
	}
	tmpl, err := New(info.Version, info.IndexPrefix, info.ElasticLicensed, esVersion, config, migration)
	if err != nil {
		return nil, fmt.Errorf("error creating template instance: %v", err)
	}
	return tmpl, nil
}

func (b *templateBuilder) buildBody(tmpl *Template, config TemplateConfig, fields []byte) (common.MapStr, error) {
	if config.Overwrite {
		b.log.Info("Existing template will be overwritten, as overwrite is enabled.")
	}

	if config.JSON.Enabled {
		return b.buildBodyFromJSON(config)
	}
	if config.Fields != "" {
		return b.buildBodyFromFile(tmpl, config)
	}
	if fields == nil {
		return b.buildMinimalTemplate(tmpl)
	}
	return b.buildBodyFromFields(tmpl, fields)
}

func (b *templateBuilder) buildBodyFromJSON(config TemplateConfig) (common.MapStr, error) {
	jsonPath := paths.Resolve(paths.Config, config.JSON.Path)
	if _, err := os.Stat(jsonPath); err != nil {
		return nil, fmt.Errorf("error checking json file %s for template: %v", jsonPath, err)
	}
	b.log.Debugf("Loading json template from file %s", jsonPath)
	content, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s for template: %v", jsonPath, err)

	}
	var body map[string]interface{}
	err = json.Unmarshal(content, &body)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json template: %s", err)
	}
	return body, nil
}

func (b *templateBuilder) buildBodyFromFile(tmpl *Template, config TemplateConfig) (common.MapStr, error) {
	b.log.Debugf("Load fields.yml from file: %s", config.Fields)
	fieldsPath := paths.Resolve(paths.Config, config.Fields)
	body, err := tmpl.LoadFile(fieldsPath)
	if err != nil {
		return nil, fmt.Errorf("error creating template from file %s: %v", fieldsPath, err)
	}
	return body, nil
}

func (b *templateBuilder) buildBodyFromFields(tmpl *Template, fields []byte) (common.MapStr, error) {
	b.log.Debug("Load default fields")
	body, err := tmpl.LoadBytes(fields)
	if err != nil {
		return nil, fmt.Errorf("error creating template: %v", err)
	}
	return body, nil
}

func (b *templateBuilder) buildMinimalTemplate(tmpl *Template) (common.MapStr, error) {
	b.log.Debug("Load minimal template")
	body, err := tmpl.LoadMinimal()
	if err != nil {
		return nil, fmt.Errorf("error creating mimimal template: %v", err)
	}
	return body, nil
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("request failed with http status code %v", e.status)
}

func esVersionParams(ver common.Version) map[string]string {
	if ver.Major == 6 && ver.Minor == 7 {
		return map[string]string{
			"include_type_name": "true",
		}
	}

	return nil
}
