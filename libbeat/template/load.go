package template

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
)

// TemplateLoader is a subset of the Elasticsearch client API capable of
// loading the template.
type ESClient interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() string
}

type Loader struct {
	config   TemplateConfig
	client   ESClient
	beatInfo common.BeatInfo
}

func NewLoader(cfg *common.Config, client ESClient, beatInfo common.BeatInfo) (*Loader, error) {

	config := defaultConfig

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	return &Loader{
		config:   config,
		client:   client,
		beatInfo: beatInfo,
	}, nil
}

// Load checks if the index mapping template should be loaded
// In case the template is not already loaded or overwriting is enabled, the
// template is written to index
func (l *Loader) Load() error {

	if l.config.Name == "" {
		l.config.Name = l.beatInfo.Beat
	}

	tmpl, err := New(l.beatInfo.Version, l.client.GetVersion(), l.config.Name, l.config.Settings)
	if err != nil {
		return fmt.Errorf("error creating template instance: %v", err)
	}

	// Check if template already exist or should be overwritten
	exists := l.CheckTemplate(tmpl.GetName())
	if !exists || l.config.Overwrite {

		logp.Info("Loading template for Elasticsearch version: %s", l.client.GetVersion())

		if l.config.Overwrite {
			logp.Info("Existing template will be overwritten, as overwrite is enabled.")
		}

		fieldsPath := paths.Resolve(paths.Config, l.config.Fields)

		output, err := tmpl.Load(fieldsPath)
		if err != nil {
			return fmt.Errorf("error creating template from file %s: %v", fieldsPath, err)
		}

		err = l.LoadTemplate(tmpl.GetName(), output)
		if err != nil {
			return fmt.Errorf("could not load template: %v", err)
		}
	} else {
		logp.Info("Template already exists and will not be overwritten.")
	}

	return nil
}

// Generate generates the template and writes it to a file based on the configuration
// from `output_to_file`.
func (l *Loader) Generate() error {
	if l.config.OutputToFile.Version == "" {
		l.config.OutputToFile.Version = l.beatInfo.Version
	}

	if l.config.Name == "" {
		l.config.Name = l.beatInfo.Beat
	}

	tmpl, err := New(l.beatInfo.Version, l.config.OutputToFile.Version, l.config.Name, l.config.Settings)
	if err != nil {
		return fmt.Errorf("error creating template instance: %v", err)
	}

	fieldsPath := paths.Resolve(paths.Config, l.config.Fields)

	output, err := tmpl.Load(fieldsPath)
	if err != nil {
		return fmt.Errorf("error creating template from file %s: %v", fieldsPath, err)
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling template: %v", err)
	}

	err = ioutil.WriteFile(l.config.OutputToFile.Path, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %v", l.config.OutputToFile.Path, err)
	}

	logp.Info("Template for Elasticsearch %s written to: %s", l.config.OutputToFile.Version, l.config.OutputToFile.Path)
	return nil
}

// LoadTemplate loads a template into Elasticsearch overwriting the existing
// template if it exists. If you wish to not overwrite an existing template
// then use CheckTemplate prior to calling this method.
func (l *Loader) LoadTemplate(templateName string, template map[string]interface{}) error {

	logp.Info("load template: %s", templateName)
	path := "/_template/" + templateName
	body, err := l.client.LoadJSON(path, template)
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

	if status != 200 {
		return false
	}

	return true
}
