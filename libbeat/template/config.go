package template

type TemplateConfig struct {
	Enabled   bool             `config:"enabled"`
	Name      string           `config:"name"`
	Fields    string           `config:"fields"`
	Overwrite bool             `config:"overwrite"`
	Settings  TemplateSettings `config:"settings"`
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
