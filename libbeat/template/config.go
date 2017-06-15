package template

type TemplateConfig struct {
	Enabled      bool             `config:"enabled"`
	Name         string           `config:"name"`
	Fields       string           `config:"fields"`
	Overwrite    bool             `config:"overwrite"`
	Settings     TemplateSettings `config:"settings"`
	OutputToFile OutputToFile     `config:"output_to_file"`
}

// OutputToFile contains the configuration options for generating
// and writing the template into a file.
type OutputToFile struct {
	Path    string `config:"path"`
	Version string `config:"version"`
}

type TemplateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

var (
	defaultConfig = TemplateConfig{
		Enabled: true,
		Fields:  "fields.yml",
	}
)
