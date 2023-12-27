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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/lifecycle"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/elastic-agent-libs/version"
)

// Loader interface for loading templates.
type Loader interface {
	Load(config TemplateConfig, info beat.Info, fields []byte, migration bool) error
}

// ESLoader implements Loader interface for loading templates to Elasticsearch.
type ESLoader struct {
	client          ESClient
	lifecycleClient lifecycle.ClientHandler
	builder         *templateBuilder
	log             *logp.Logger
}

// ESClient is a subset of the Elasticsearch client API capable of
// loading the template.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() version.V
	IsServerless() bool
}

// FileLoader implements Loader interface for loading templates to a File.
type FileLoader struct {
	client  FileClient
	builder *templateBuilder
	log     *logp.Logger
}

// FileClient defines the minimal interface required for the FileLoader
type FileClient interface {
	GetVersion() version.V
	Write(component string, name string, body string) error
}

type StatusError struct {
	status int
}

type templateBuilder struct {
	log          *logp.Logger
	isServerless bool
}

// NewESLoader creates a new template loader for ES
func NewESLoader(client ESClient, lifecycleClient lifecycle.ClientHandler) (*ESLoader, error) {
	if client == nil {
		return nil, errors.New("can not load template without active Elasticsearch client")
	}
	return &ESLoader{client: client, lifecycleClient: lifecycleClient,
		builder: newTemplateBuilder(client.IsServerless()), log: logp.NewLogger("template_loader")}, nil
}

// NewFileLoader creates a new template loader for the given file.
func NewFileLoader(c FileClient, isServerless bool) *FileLoader {
	// other components of the file loader will fail if both ILM and DSL are set,
	// so at this point it's fairly safe to just pass cfg.DSL.Enabled
	return &FileLoader{client: c, builder: newTemplateBuilder(isServerless), log: logp.NewLogger("file_template_loader")}
}

func newTemplateBuilder(serverlessMode bool) *templateBuilder {
	return &templateBuilder{log: logp.NewLogger("template"), isServerless: serverlessMode}
}

// Load checks if the index mapping template should be loaded.
// In case the template is not already loaded or overwriting is enabled, the
// template is built and written to index.
func (l *ESLoader) Load(config TemplateConfig, info beat.Info, fields []byte, migration bool) error {

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

	exists, err := l.checkExistsTemplate(templateName)
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
	if err := l.loadTemplate(templateName, body); err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}
	l.log.Infof("Template with name %q loaded.", templateName)

	// if JSON template is loaded and it is not a data stream
	// we are done with loading.
	if config.JSON.Enabled && !config.JSON.IsDataStream {
		return nil
	}

	// If a data stream already exists, we do not attempt to delete or overwrite
	// it because it would delete all backing indices, and the user would lose all
	// their documents.
	dataStreamExist, err := l.checkExistsDatastream(templateName)
	if err != nil {
		return fmt.Errorf("failed to check data stream: %w", err)
	}
	if dataStreamExist {
		l.log.Infof("Data stream with name %q already exists.", templateName)
		// for serverless, we can update the lifecycle policy safely
		// Note that updating the lifecycle will delete older documents
		// if the policy requires it; i.e, changing the data_retention from 10d to 7d
		// will delete the documents older than 7 days.
		if l.client.IsServerless() {
			l.log.Infof("overwriting lifecycle policy")
			err = l.lifecycleClient.CreatePolicyFromConfig()
			if err != nil {
				return fmt.Errorf("error updating lifecycle policy: %w", err)
			}
		}
		return nil
	}

	if err := l.putDataStream(templateName); err != nil {
		return fmt.Errorf("failed to put data stream: %w", err)
	}
	l.log.Infof("Data stream with name %q loaded.", templateName)

	return nil
}

// loadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use CheckTemplate prior to calling this method.
func (l *ESLoader) loadTemplate(templateName string, template map[string]interface{}) error {
	l.log.Infof("Try loading template %s to Elasticsearch", templateName)
	path := "/_index_template/" + templateName
	status, body, err := l.client.Request("PUT", path, "", nil, template)
	if err != nil {
		return fmt.Errorf("couldn't load template: %w. Response body: %s", err, body)
	}
	if status > http.StatusMultipleChoices { //http status 300
		return fmt.Errorf("couldn't load json. Status: %v", status)
	}
	return nil
}

func (l *ESLoader) checkExistsDatastream(name string) (bool, error) {
	status, _, err := l.client.Request("GET", "/_data_stream/"+name, "", nil, nil)
	if status == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (l *ESLoader) putDataStream(name string) error {
	l.log.Infof("Try loading data stream %s to Elasticsearch", name)
	path := "/_data_stream/" + name
	_, body, err := l.client.Request("PUT", path, "", nil, nil)
	if err != nil {
		return fmt.Errorf("could not put data stream: %w. Response body: %s", err, body)
	}
	return nil
}

// existsTemplate checks if a given template already exist, using the
// `/_index_template/<name>` API.
//
// An error is returned if the loader failed to execute the request, or a
// status code indicating some problems is encountered.
func (l *ESLoader) checkExistsTemplate(name string) (bool, error) {
	status, _, err := l.client.Request("HEAD", "/_index_template/"+name, "", nil, nil)
	if status == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
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
		return fmt.Errorf("error printing template: %w", err)
	}
	return nil
}

func (b *templateBuilder) template(config TemplateConfig, info beat.Info, esVersion version.V, migration bool) (*Template, error) {
	if !config.Enabled {
		b.log.Info("template config not enabled")
		return nil, nil
	}
	tmpl, err := New(b.isServerless, info.Version, info.IndexPrefix, info.ElasticLicensed, esVersion, config, migration)
	if err != nil {
		return nil, fmt.Errorf("error creating template instance: %w", err)
	}
	return tmpl, nil
}

func (b *templateBuilder) buildBody(tmpl *Template, config TemplateConfig, fields []byte) (mapstr.M, error) {
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
		b.log.Debug("Load minimal template")
		return tmpl.LoadMinimal(), nil
	}
	return b.buildBodyFromFields(tmpl, fields)
}

func (b *templateBuilder) buildBodyFromJSON(config TemplateConfig) (mapstr.M, error) {
	jsonPath := paths.Resolve(paths.Config, config.JSON.Path)
	if _, err := os.Stat(jsonPath); err != nil {
		return nil, fmt.Errorf("error checking json file %s for template: %w", jsonPath, err)
	}
	b.log.Debugf("Loading json template from file %s", jsonPath)
	content, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s for template: %w", jsonPath, err)

	}
	var body map[string]interface{}
	err = json.Unmarshal(content, &body)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json template: %w", err)
	}
	return body, nil
}

func (b *templateBuilder) buildBodyFromFile(tmpl *Template, config TemplateConfig) (mapstr.M, error) {
	b.log.Debugf("Load fields.yml from file: %s", config.Fields)
	fieldsPath := paths.Resolve(paths.Config, config.Fields)
	body, err := tmpl.LoadFile(fieldsPath)
	if err != nil {
		return nil, fmt.Errorf("error creating template from file %s: %w", fieldsPath, err)
	}
	return body, nil
}

func (b *templateBuilder) buildBodyFromFields(tmpl *Template, fields []byte) (mapstr.M, error) {
	b.log.Debug("Load default fields")
	body, err := tmpl.LoadBytes(fields)
	if err != nil {
		return nil, fmt.Errorf("error creating template: %w", err)
	}
	return body, nil
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("request failed with http status code %v", e.status)
}
