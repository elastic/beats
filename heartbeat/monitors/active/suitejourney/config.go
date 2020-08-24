package suitejourney

import "github.com/elastic/beats/v7/libbeat/common"

type Config struct {
	Path string `config:"path"`
	ScriptParams common.MapStr `config:"script_params"`
	JourneyName string `config:"journey_name"`
}

func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{
}
