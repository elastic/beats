package template

type TemplateConfig struct {
	Enabled      bool                   `config:"enabled"`
	Name         string                 `config:"name"`
	Fields       string                 `config:"fields"`
	Overwrite    bool                   `config:"overwrite"`
	OutputToFile string                 `config:"output_to_file"`
	Settings     map[string]interface{} `config:"settings"`
}

var (
	defaultConfig = TemplateConfig{
		Enabled: true,
		Fields:  "fields.yml",
	}
)
