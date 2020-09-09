// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package script

import "github.com/elastic/beats/v7/libbeat/common"

type Config struct {
	Script       string        `config:"script"`
	ScriptParams common.MapStr `config:"script_params"`
}

func (c *Config) Validate() error {
	return nil
}

var defaultConfig = Config{}
