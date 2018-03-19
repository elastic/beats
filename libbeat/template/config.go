package template

import "github.com/elastic/beats/libbeat/common"

type TemplateConfig struct {
	Enabled      bool             `config:"enabled"`
	Name         string           `config:"name"`
	Pattern      string           `config:"pattern"`
	Fields       string           `config:"fields"`
	AppendFields common.Fields    `config:"append_fields"`
	Overwrite    bool             `config:"overwrite"`
	Settings     TemplateSettings `config:"settings"`
}

type TemplateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

var (
	// DefaultConfig for index template
	DefaultConfig = TemplateConfig{
		Enabled: true,
		Fields:  "fields.yml",
	}
)
