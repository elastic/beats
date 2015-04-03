package common

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/mapstructure"
)

type Config struct {
	Options MapStr
	Meta    toml.MetaData
}

func DecodeConfig(config Config, options interface{}) error {
	err := mapstructure.Decode(config.Options, options)
	if err != nil {
		return fmt.Errorf("Error while decoding configuration: %v", err)
	}
	return nil
}
