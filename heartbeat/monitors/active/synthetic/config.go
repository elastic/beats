package synthetic

type Config struct {
	Script string `config:"script"`
	Browsers []string `config:"browsers"`
	RunnerURL string `config:"runner_url"`
	ApiKey string `config:"api_key"`
}

func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{
	RunnerURL: "http://localhost:5678",
	Browsers: []string{"chromium", "webkit", "firefox"},
}
