package beat

type TemplateConfig struct {
	Enabled      bool              `config:"enabled"`
	Name         string            `config:"name"`
	Fields       string            `config:"fields"`
	Overwrite    bool              `config:"overwrite"`
	OutputToFile string            `config:"output_to_file"`
	Settings     map[string]string `config:"settings"`
}
