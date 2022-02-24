// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/str"
	ucfg "github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
)

// C object to store hierarchical configurations into.
// See https://godoc.org/github.com/elastic/go-ucfg#Config
type C ucfg.Config

// Namespace stores at most one configuration section by name and sub-section.
type Namespace struct {
	name   string
	config *C
}

const mask = "xxxxx"

var (
	configOpts = []ucfg.Option{
		ucfg.PathSep("."),
		ucfg.ResolveEnv,
		ucfg.VarExp,
	}

	maskList = str.MakeSet(
		"password",
		"passphrase",
		"key_passphrase",
		"pass",
		"proxy_url",
		"url",
		"urls",
		"host",
		"hosts",
		"authorization",
		"proxy-authorization",
	)
)

func NewConfig() *C {
	return fromConfig(ucfg.New())
}

// NewConfigFrom creates a new C object from the given input.
// From can be any kind of structured data (struct, map, array, slice).
//
// If from is a string, the contents is treated like raw YAML input. The string
// will be parsed and a structure config object is build from the parsed
// result.
func NewConfigFrom(from interface{}) (*C, error) {
	if str, ok := from.(string); ok {
		c, err := yaml.NewConfig([]byte(str), configOpts...)
		return fromConfig(c), err
	}

	c, err := ucfg.NewFrom(from, configOpts...)
	return fromConfig(c), err
}

// MustNewConfigFrom creates a new C object from the given input.
// From can be any kind of structured data (struct, map, array, slice).
//
// If from is a string, the contents is treated like raw YAML input. The string
// will be parsed and a structure config object is build from the parsed
// result.
//
// MustNewConfigFrom panics if an error occurs.
func MustNewConfigFrom(from interface{}) *C {
	cfg, err := NewConfigFrom(from)
	if err != nil {
		panic(err)
	}
	return cfg
}

// MergeConfigs merges the configs together. If there are
// different values for the same key, the last one always overwrites
// the previous values.
func MergeConfigs(cfgs ...*C) (*C, error) {
	config := NewConfig()
	for _, c := range cfgs {
		if err := config.Merge(c); err != nil {
			return nil, err
		}
	}
	return config, nil
}

// MergeConfigs merges the configs together based on the provided opts.
// If there are different values for the same key, the last one always overwrites
// the previous values.
func MergeConfigsWithOptions(cfgs []*C, options ...ucfg.Option) (*C, error) {
	config := NewConfig()
	for _, c := range cfgs {
		if err := config.MergeWithOpts(c, options...); err != nil {
			return nil, err
		}
	}
	return config, nil
}

// NewConfigWithYAML reads a YAML configuration.
func NewConfigWithYAML(in []byte, source string) (*C, error) {
	opts := append(
		[]ucfg.Option{
			ucfg.MetaData(ucfg.Meta{Source: source}),
		},
		configOpts...,
	)
	c, err := yaml.NewConfig(in, opts...)
	return fromConfig(c), err
}

// OverwriteConfigOpts allow to change the globally set config option
func OverwriteConfigOpts(options []ucfg.Option) {
	configOpts = options
}

// Merge merges the parameter into the C object.
func (c *C) Merge(from interface{}) error {
	return c.access().Merge(from, configOpts...)
}

// Merge merges the parameter into the C object based on the provided options.
func (c *C) MergeWithOpts(from interface{}, opts ...ucfg.Option) error {
	o := configOpts
	if opts != nil {
		o = append(o, opts...)
	}
	return c.access().Merge(from, o...)
}

func (c *C) Unpack(to interface{}) error {
	return c.access().Unpack(to, configOpts...)
}

func (c *C) Path() string {
	return c.access().Path(".")
}

func (c *C) PathOf(field string) string {
	return c.access().PathOf(field, ".")
}

func (c *C) Remove(name string, idx int) (bool, error) {
	return c.access().Remove(name, idx, configOpts...)
}

func (c *C) Has(name string, idx int) (bool, error) {
	return c.access().Has(name, idx, configOpts...)
}

func (c *C) HasField(name string) bool {
	return c.access().HasField(name)
}

func (c *C) CountField(name string) (int, error) {
	return c.access().CountField(name)
}

func (c *C) Bool(name string, idx int) (bool, error) {
	return c.access().Bool(name, idx, configOpts...)
}

func (c *C) String(name string, idx int) (string, error) {
	return c.access().String(name, idx, configOpts...)
}

func (c *C) Int(name string, idx int) (int64, error) {
	return c.access().Int(name, idx, configOpts...)
}

func (c *C) Float(name string, idx int) (float64, error) {
	return c.access().Float(name, idx, configOpts...)
}

func (c *C) Child(name string, idx int) (*C, error) {
	sub, err := c.access().Child(name, idx, configOpts...)
	return fromConfig(sub), err
}

func (c *C) SetBool(name string, idx int, value bool) error {
	return c.access().SetBool(name, idx, value, configOpts...)
}

func (c *C) SetInt(name string, idx int, value int64) error {
	return c.access().SetInt(name, idx, value, configOpts...)
}

func (c *C) SetFloat(name string, idx int, value float64) error {
	return c.access().SetFloat(name, idx, value, configOpts...)
}

func (c *C) SetString(name string, idx int, value string) error {
	return c.access().SetString(name, idx, value, configOpts...)
}

func (c *C) SetChild(name string, idx int, value *C) error {
	return c.access().SetChild(name, idx, value.access(), configOpts...)
}

func (c *C) IsDict() bool {
	return c.access().IsDict()
}

func (c *C) IsArray() bool {
	return c.access().IsArray()
}

// FlattenedKeys return a sorted flattened views of the set keys in the configuration.
func (c *C) FlattenedKeys() []string {
	return c.access().FlattenedKeys(configOpts...)
}

// Enabled return the configured enabled value or true by default.
func (c *C) Enabled() bool {
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

func fromConfig(in *ucfg.Config) *C {
	return (*C)(in)
}

func (c *C) access() *ucfg.Config {
	return (*ucfg.Config)(c)
}

// GetFields returns the list of fields in the configuration.
func (c *C) GetFields() []string {
	return c.access().GetFields()
}

// Unpack unpacks a configuration with at most one sub object. An sub object is
// ignored if it is disabled by setting `enabled: false`. If the configuration
// passed contains multiple active sub objects, Unpack will return an error.
func (ns *Namespace) Unpack(cfg *C) error {
	fields := cfg.GetFields()
	if len(fields) == 0 {
		return nil
	}

	var (
		err   error
		found bool
	)

	for _, name := range fields {
		var sub *C

		sub, err = cfg.Child(name, -1)
		if err != nil {
			// element is no configuration object -> continue so a namespace
			// Config unpacked as a namespace can have other configuration
			// values as well
			continue
		}

		if !sub.Enabled() {
			continue
		}

		if ns.name != "" {
			return errors.New("more than one namespace configured")
		}

		ns.name = name
		ns.config = sub
		found = true
	}

	if !found {
		return err
	}
	return nil
}

// Name returns the configuration sections it's name if a section has been set.
func (ns *Namespace) Name() string {
	return ns.name
}

// Config return the sub-configuration section if a section has been set.
func (ns *Namespace) Config() *C {
	return ns.config
}

// IsSet returns true if a sub-configuration section has been set.
func (ns *Namespace) IsSet() bool {
	return ns.config != nil
}

// DebugString prints a human readable representation of the underlying config using
// JSON formatting.
func DebugString(c *C, filterPrivate bool) string {
	var bufs []string

	if c.IsDict() {
		var content map[string]interface{}
		if err := c.Unpack(&content); err != nil {
			return fmt.Sprintf("<config error> %v", err)
		}
		if filterPrivate {
			ApplyLoggingMask(content)
		}
		j, _ := json.MarshalIndent(content, "", "  ")
		bufs = append(bufs, string(j))
	}
	if c.IsArray() {
		var content []interface{}
		if err := c.Unpack(&content); err != nil {
			return fmt.Sprintf("<config error> %v", err)
		}
		if filterPrivate {
			ApplyLoggingMask(content)
		}
		j, _ := json.MarshalIndent(content, "", "  ")
		bufs = append(bufs, string(j))
	}

	if len(bufs) == 0 {
		return ""
	}
	return strings.Join(bufs, "\n")
}

// ApplyLoggingMask redacts the values of keys that might
// contain sensitive data (password, passphrase, etc.).
func ApplyLoggingMask(c interface{}) {
	switch cfg := c.(type) {
	case map[string]interface{}:
		for k, v := range cfg {
			if maskList.Has(strings.ToLower(k)) {
				if arr, ok := v.([]interface{}); ok {
					for i := range arr {
						arr[i] = mask
					}
				} else {
					cfg[k] = mask
				}
			} else {
				ApplyLoggingMask(v)
			}
		}

	case []interface{}:
		for _, elem := range cfg {
			ApplyLoggingMask(elem)
		}
	}
}
