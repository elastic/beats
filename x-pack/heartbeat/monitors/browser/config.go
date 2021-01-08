// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

type Config struct {
	Script      string        `config:"script"`
	Params      common.MapStr `config:"script_params"`
}

func (c *Config) Validate() error {
	if c.Script != "" {
		return fmt.Errorf("no script specified for journey!")
	}
	return nil
}

var defaultConfig = Config{}
