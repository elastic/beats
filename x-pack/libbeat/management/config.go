// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
)

// ModeFleet is a management mode where fleet is used to retrieve configurations.
const ModeFleet = "x-pack-fleet"

// Config for central management
type Config struct {
	Enabled   bool                    `config:"enabled" yaml:"enabled"`
	Mode      string                  `config:"mode" yaml:"mode"`
	Blacklist ConfigBlacklistSettings `config:"blacklist" yaml:"blacklist"`
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

func defaultConfig() *Config {
	return &Config{
		Mode: ModeFleet,
		Blacklist: ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}
}

// Config returns a common.Config object holding the config from this block
func (c *ConfigBlock) Config() (*common.Config, error) {
	return common.NewConfigFrom(c.Raw)
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

// ConfigBlocksEqual returns true if the given config blocks are equal, false if not
func ConfigBlocksEqual(a, b ConfigBlocks) (bool, error) {
	// If there is an errors when hashing the config blocks its because the format changed.
	aHash, err := hashstructure.Hash(a, nil)
	if err != nil {
		return false, errors.Wrap(err, "could not hash config blocks")
	}

	bHash, err := hashstructure.Hash(b, nil)
	if err != nil {
		return false, errors.Wrap(err, "could not hash config blocks")
	}

	return aHash == bHash, nil
}
