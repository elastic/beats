package script

import "github.com/elastic/beats/v7/libbeat/common"

type Config struct {
	Script string `config:"script"`
	ScriptParams common.MapStr `config:"script_params"`
}

func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{
}
