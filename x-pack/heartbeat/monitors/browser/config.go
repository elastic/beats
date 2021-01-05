// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

type Config struct {
	Path        string        `config:"path"`
	Script      string        `config:"script"`
	Params      common.MapStr `config:"script_params"`
	JourneyName string        `config:"journey_name"`
}

func (c *Config) Validate() error {
	if c.Script != "" && c.Path != "" {
		return fmt.Errorf("both path and script specified! Only one of these options may be present!")
	}
	return nil
}

var defaultConfig = Config{}
