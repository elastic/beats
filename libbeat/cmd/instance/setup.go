package instance

type TemplateConfig struct {
	Enabled   bool              `config:"enabled"`
	Name      string            `config:"name"`
	Fields    string            `config:"fields"`
	Overwrite bool              `config:"overwrite"`
	Settings  map[string]string `config:"settings"`
}
