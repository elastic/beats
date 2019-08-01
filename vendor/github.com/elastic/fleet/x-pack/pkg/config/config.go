// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
)

// DefaultOptions defaults options used to read the configuration
var DefaultOptions = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

// RawConfig is the raw ucfg configuration.
// NOTES: This type alias is used to make use the ucfg's unpack is executed.
type RawConfig ucfg.Config

// Config is a wrapper on top of ucfg.
type Config struct {
	*ucfg.Config
	opts []ucfg.Option
}

// ReadFile reads a configuration from disk.
func ReadFile(file string) (*Config, error) {
	return nil, nil
}

// LoadYAML takes YAML configuration and return a concrete Config or any errors.
func LoadYAML(path string, opts ...ucfg.Option) (*Config, error) {
	if len(opts) == 0 {
		opts = DefaultOptions
	}
	config, err := yaml.NewConfigWithFile(path, opts...)
	if err != nil {
		return nil, err
	}
	return newConfigFrom(config, opts...), nil
}

// NewConfigFrom takes a interface and read the configuration like it was YAML.
func NewConfigFrom(from interface{}, opts ...ucfg.Option) (*Config, error) {
	if len(opts) == 0 {
		opts = DefaultOptions
	}

	if str, ok := from.(string); ok {
		c, err := yaml.NewConfig([]byte(str), opts...)
		return newConfigFrom(c), err
	}

	c, err := ucfg.NewFrom(from, opts...)
	return newConfigFrom(c), err
}

// MustNewConfigFrom try to create a configuration based on the type passed as arguments and panic
// on failures.
func MustNewConfigFrom(from interface{}, opts ...ucfg.Option) *Config {
	c, err := NewConfigFrom(from, opts...)
	if err != nil {
		panic(fmt.Sprintf("could not read configuration %+v", err))
	}
	return c
}

// New return a new config with configured options.
func New(opts ...ucfg.Option) *Config {
	return &Config{opts: opts}
}

// NewWithDefaults returns a new config with a predefined set of options.
func NewWithDefaults() *Config {
	return New(DefaultOptions...)
}

func newConfigFrom(in *ucfg.Config, opts ...ucfg.Option) *Config {
	return &Config{Config: in, opts: opts}
}

// Wrap wraps the current config with a set of predefined options.
func (c *Config) Wrap(opts ...ucfg.Option) *Config {
	return &Config{Config: c.Config, opts: opts}
}

// Unpack unpacks a struct to Config.
func (c *Config) Unpack(to interface{}) error {
	return c.access().Unpack(to, c.opts...)
}

func (c *Config) access() *ucfg.Config {
	return (*ucfg.Config)(c.Config)
}
