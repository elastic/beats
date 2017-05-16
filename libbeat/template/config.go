package template

type TemplateConfig struct {
	Enabled      bool             `config:"enabled"`
	Name         string           `config:"name"`
	Fields       string           `config:"fields"`
	Overwrite    bool             `config:"overwrite"`
	OutputToFile string           `config:"output_to_file"`
	Settings     templateSettings `config:"settings"`
}

type templateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

var (
	defaultConfig = TemplateConfig{
		Enabled: true,
		Fields:  "fields.yml",
	}
)
