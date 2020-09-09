// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/cfgutil"
	"github.com/elastic/go-ucfg/yaml"
)

// DefaultOptions defaults options used to read the configuration
var DefaultOptions = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

// Config custom type over a ucfg.Config to add new methods on the object.
type Config ucfg.Config

// LoadYAML takes YAML configuration and return a concrete Config or any errors.
func LoadYAML(path string, opts ...ucfg.Option) (*Config, error) {
	if len(opts) == 0 {
		opts = DefaultOptions
	}
	config, err := yaml.NewConfigWithFile(path, opts...)
	if err != nil {
		return nil, err
	}
	return newConfigFrom(config), nil
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

	if in, ok := from.(io.Reader); ok {
		if closer, ok := from.(io.Closer); ok {
			defer closer.Close()
		}

		content, err := ioutil.ReadAll(in)
		if err != nil {
			return nil, err
		}
		c, err := yaml.NewConfig(content, opts...)
		return newConfigFrom(c), err
	}

	c, err := ucfg.NewFrom(from, opts...)
	return newConfigFrom(c), err
}

// MustNewConfigFrom try to create a configuration based on the type passed as arguments and panic
// on failures.
func MustNewConfigFrom(from interface{}) *Config {
	c, err := NewConfigFrom(from)
	if err != nil {
		panic(fmt.Sprintf("could not read configuration %+v", err))
	}
	return c
}

func newConfigFrom(in *ucfg.Config) *Config {
	return (*Config)(in)
}

// Unpack unpacks a struct to Config.
func (c *Config) Unpack(to interface{}) error {
	return c.access().Unpack(to, DefaultOptions...)
}

func (c *Config) access() *ucfg.Config {
	return (*ucfg.Config)(c)
}

// Merge merges two configuration together.
func (c *Config) Merge(from interface{}, opts ...ucfg.Option) error {
	if len(opts) == 0 {
		opts = DefaultOptions
	}
	return c.access().Merge(from, opts...)
}

// ToMapStr takes the config and transform it into a map[string]interface{}
func (c *Config) ToMapStr() (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := c.Unpack(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// Enabled return the configured enabled value or true by default.
func (c *Config) Enabled() bool {
	testEnabled := struct {
		Enabled bool `config:"enabled"`
	}{true}

	if c == nil {
		return false
	}
	if err := c.Unpack(&testEnabled); err != nil {
		// if unpacking fails, expect 'enabled' being set to default value
		return true
	}
	return testEnabled.Enabled
}

// LoadFile take a path and load the file and return a new configuration.
func LoadFile(path string) (*Config, error) {
	c, err := yaml.NewConfigWithFile(path, DefaultOptions...)
	if err != nil {
		return nil, err
	}

	cfg := newConfigFrom(c)
	return cfg, err
}

// LoadFiles takes multiples files, load and merge all of them in a single one.
func LoadFiles(paths ...string) (*Config, error) {
	merger := cfgutil.NewCollector(nil, DefaultOptions...)
	for _, path := range paths {
		cfg, err := LoadFile(path)
		if err := merger.Add(cfg.access(), err); err != nil {
			return nil, err
		}
	}
	return newConfigFrom(merger.Config()), nil
}
