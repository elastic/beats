// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Config for central management
type Config struct {
	Enabled                   bool                    `config:"enabled" yaml:"enabled"`
	Blacklist                 ConfigBlacklistSettings `config:"blacklist" yaml:"blacklist"`
	RestartOnOutputChange     bool                    `config:"restart_on_output_change" yaml:"restart_on_output_change"`
	InsecureGRPCURLForTesting string                  `config:"insecure_grpc_url_for_testing" yaml:"insecure_grpc_url_for_testing"`
}

// ConfigBlock stores a piece of config from central management
type ConfigBlock struct {
	Raw map[string]interface{}
}

// ConfigBlocksWithType is a list of config blocks with the same type
type ConfigBlocksWithType struct {
	Type   string
	Blocks []*ConfigBlock
}

// ConfigBlocks holds a list of type + configs objects
type ConfigBlocks []ConfigBlocksWithType

// DefaultConfig returns the default config for the V2 manager
func DefaultConfig() *Config {
	return &Config{
		Blacklist: ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}
}

// Config returns a config.C object holding the config from this block
func (c *ConfigBlock) Config() (*conf.C, error) {
	return conf.NewConfigFrom(c.Raw)
}

// ConfigWithMeta returns a reload.ConfigWithMeta object holding the config from this block, meta will be nil
func (c *ConfigBlock) ConfigWithMeta() (*reload.ConfigWithMeta, error) {
	config, err := c.Config()
	if err != nil {
		return nil, err
	}
	return &reload.ConfigWithMeta{
		Config: config,
	}, nil
}

func (c Config) Validate() error {
	if !c.Enabled && c.InsecureGRPCURLForTesting != "" {
		return errors.New("'management.insecure_grpc_url_for_testing' can only be used if 'management.enabled' is set.")
	}
	return nil
}
