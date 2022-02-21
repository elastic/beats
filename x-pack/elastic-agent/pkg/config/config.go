// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/cfgutil"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// options hold the specified options
type options struct {
	skipKeys []string
}

// Option is an option type that modifies how loading configs work
type Option func(*options)

// VarSkipKeys prevents variable expansion for these keys.
//
// The provided keys only skip if the keys are top-level keys.
func VarSkipKeys(keys ...string) Option {
	return func(opts *options) {
		opts.skipKeys = keys
	}
}

// DefaultOptions defaults options used to read the configuration
var DefaultOptions = []interface{}{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
	VarSkipKeys("inputs"),
}

// Config custom type over a ucfg.Config to add new methods on the object.
type Config ucfg.Config

// New creates a new empty config.
func New() *Config {
	return newConfigFrom(ucfg.New())
}

// NewConfigFrom takes a interface and read the configuration like it was YAML.
func NewConfigFrom(from interface{}, opts ...interface{}) (*Config, error) {
	if len(opts) == 0 {
		opts = DefaultOptions
	}
	var ucfgOpts []ucfg.Option
	var localOpts []Option
	for _, o := range opts {
		switch ot := o.(type) {
		case ucfg.Option:
			ucfgOpts = append(ucfgOpts, ot)
		case Option:
			localOpts = append(localOpts, ot)
		default:
			return nil, fmt.Errorf("unknown option type %T", o)
		}
	}
	local := &options{}
	for _, o := range localOpts {
		o(local)
	}

	var data map[string]interface{}
	var err error
	if bytes, ok := from.([]byte); ok {
		err = yaml.Unmarshal(bytes, &data)
		if err != nil {
			return nil, err
		}
	} else if str, ok := from.(string); ok {
		err = yaml.Unmarshal([]byte(str), &data)
		if err != nil {
			return nil, err
		}
	} else if in, ok := from.(io.Reader); ok {
		if closer, ok := from.(io.Closer); ok {
			defer closer.Close()
		}
		fData, err := ioutil.ReadAll(in)
		if err != nil {
			return nil, err
		}
		logp.Info("NewConfigFrom io.Reader: %s", fData)
		err = yaml.Unmarshal(fData, &data)
		if err != nil {
			return nil, err
		}
	} else if contents, ok := from.(map[string]interface{}); ok {
		data = contents
	} else {
		c, err := ucfg.NewFrom(from, ucfgOpts...)
		return newConfigFrom(c), err
	}

	skippedKeys := map[string]interface{}{}
	for _, skip := range local.skipKeys {
		val, ok := data[skip]
		if ok {
			skippedKeys[skip] = val
			delete(data, skip)
		}
	}
	cfg, err := ucfg.NewFrom(data, ucfgOpts...)
	if err != nil {
		return nil, err
	}
	if len(skippedKeys) > 0 {
		err = cfg.Merge(skippedKeys, ucfg.ResolveNOOP)

		// we modified incoming object
		// cleanup so skipped keys are not missing
		for k, v := range skippedKeys {
			data[k] = v
		}
	}
	return newConfigFrom(cfg), err
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
func (c *Config) Unpack(to interface{}, opts ...interface{}) error {
	ucfgOpts, err := getUcfgOptions(opts...)
	if err != nil {
		return err
	}
	return c.access().Unpack(to, ucfgOpts...)
}

func (c *Config) access() *ucfg.Config {
	return (*ucfg.Config)(c)
}

// Merge merges two configuration together.
func (c *Config) Merge(from interface{}, opts ...interface{}) error {
	ucfgOpts, err := getUcfgOptions(opts...)
	if err != nil {
		return err
	}
	return c.access().Merge(from, ucfgOpts...)
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
	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return NewConfigFrom(fp)
}

// LoadFiles takes multiples files, load and merge all of them in a single one.
func LoadFiles(paths ...string) (*Config, error) {
	merger := cfgutil.NewCollector(nil)
	for _, path := range paths {
		cfg, err := LoadFile(path)
		if err := merger.Add(cfg.access(), err); err != nil {
			return nil, err
		}
	}
	return newConfigFrom(merger.Config()), nil
}

func getUcfgOptions(opts ...interface{}) ([]ucfg.Option, error) {
	if len(opts) == 0 {
		opts = DefaultOptions
	}
	var ucfgOpts []ucfg.Option
	for _, o := range opts {
		switch ot := o.(type) {
		case ucfg.Option:
			ucfgOpts = append(ucfgOpts, ot)
		case Option:
			// ignored during unpack
			continue
		default:
			return nil, fmt.Errorf("unknown option type %T", o)
		}
	}
	return ucfgOpts, nil
}
