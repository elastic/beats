// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	xmanagement "github.com/elastic/beats/x-pack/libbeat/management"
)

// Config for central management
type Config struct {
	Enabled   bool                                `config:"enabled" yaml:"enabled"`
	Mode      string                              `config:"mode" yaml:"mode"`
	Blacklist xmanagement.ConfigBlacklistSettings `config:"blacklist" yaml:"blacklist"`
}

func defaultConfig() *Config {
	return &Config{
		Mode: xmanagement.ModeCentralManagement,
		Blacklist: xmanagement.ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}
}
