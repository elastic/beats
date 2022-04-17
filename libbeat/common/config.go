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

package common

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/menderesk/beats/v7/libbeat/common/file"
	"github.com/menderesk/beats/v7/libbeat/logp"
	ucfg "github.com/menderesk/go-ucfg"
	"github.com/menderesk/go-ucfg/cfgutil"
	"github.com/menderesk/go-ucfg/yaml"
)

var flagStrictPerms = flag.Bool("strict.perms", true, "Strict permission checking on config files")

// IsStrictPerms returns true if strict permission checking on config files is
// enabled.
func IsStrictPerms() bool {
	if !*flagStrictPerms || os.Getenv("BEAT_STRICT_PERMS") == "false" {
		return false
	}
	return true
}

// Config object to store hierarchical configurations into.
// See https://godoc.org/github.com/menderesk/go-ucfg#Config
type Config ucfg.Config

// ConfigNamespace storing at most one configuration section by name and sub-section.
type ConfigNamespace struct {
	name   string
	config *Config
}

var configOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

const (
	selectorConfig             = "config"
	selectorConfigWithPassword = "config-with-passwords"
)

// make hasSelector and configDebugf available for unit testing
var hasSelector = logp.HasSelector
var configDebugf = logp.Debug

func NewConfig() *Config {
	return fromConfig(ucfg.New())
}

// NewConfigFrom creates a new Config object from the given input.
// From can be any kind of structured data (struct, map, array, slice).
//
// If from is a string, the contents is treated like raw YAML input. The string
// will be parsed and a structure config object is build from the parsed
// result.
func NewConfigFrom(from interface{}) (*Config, error) {
	if str, ok := from.(string); ok {
		c, err := yaml.NewConfig([]byte(str), configOpts...)
		return fromConfig(c), err
	}

	c, err := ucfg.NewFrom(from, configOpts...)
	return fromConfig(c), err
}

// MustNewConfigFrom creates a new Config object from the given input.
// From can be any kind of structured data (struct, map, array, slice).
//
// If from is a string, the contents is treated like raw YAML input. The string
// will be parsed and a structure config object is build from the parsed
// result.
//
// MustNewConfigFrom panics if an error occurs.
func MustNewConfigFrom(from interface{}) *Config {
	cfg, err := NewConfigFrom(from)
	if err != nil {
		panic(err)
	}
	return cfg
}

func MergeConfigs(cfgs ...*Config) (*Config, error) {
	config := NewConfig()
	for _, c := range cfgs {
		if err := config.Merge(c); err != nil {
			return nil, err
		}
	}
	return config, nil
}

func MergeConfigsWithOptions(cfgs []*Config, options ...ucfg.Option) (*Config, error) {
	config := NewConfig()
	for _, c := range cfgs {
		if err := config.MergeWithOpts(c, options...); err != nil {
			return nil, err
		}
	}
	return config, nil
}

func NewConfigWithYAML(in []byte, source string) (*Config, error) {
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

func LoadFile(path string) (*Config, error) {
	if IsStrictPerms() {
		if err := OwnerHasExclusiveWritePerms(path); err != nil {
			return nil, err
		}
	}

	c, err := yaml.NewConfigWithFile(path, configOpts...)
	if err != nil {
		return nil, err
	}

	cfg := fromConfig(c)
	cfg.PrintDebugf("load config file '%v' =>", path)
	return cfg, err
}

func LoadFiles(paths ...string) (*Config, error) {
	merger := cfgutil.NewCollector(nil, configOpts...)
	for _, path := range paths {
		cfg, err := LoadFile(path)
		if err := merger.Add(cfg.access(), err); err != nil {
			return nil, err
		}
	}
	return fromConfig(merger.Config()), nil
}

func (c *Config) Merge(from interface{}) error {
	return c.access().Merge(from, configOpts...)
}

func (c *Config) MergeWithOpts(from interface{}, opts ...ucfg.Option) error {
	o := configOpts
	if opts != nil {
		o = append(o, opts...)
	}
	return c.access().Merge(from, o...)
}

func (c *Config) Unpack(to interface{}) error {
	return c.access().Unpack(to, configOpts...)
}

func (c *Config) Path() string {
	return c.access().Path(".")
}

func (c *Config) PathOf(field string) string {
	return c.access().PathOf(field, ".")
}

func (c *Config) Remove(name string, idx int) (bool, error) {
	return c.access().Remove(name, idx, configOpts...)
}

func (c *Config) Has(name string, idx int) (bool, error) {
	return c.access().Has(name, idx, configOpts...)
}

func (c *Config) HasField(name string) bool {
	return c.access().HasField(name)
}

func (c *Config) CountField(name string) (int, error) {
	return c.access().CountField(name)
}

func (c *Config) Bool(name string, idx int) (bool, error) {
	return c.access().Bool(name, idx, configOpts...)
}

func (c *Config) String(name string, idx int) (string, error) {
	return c.access().String(name, idx, configOpts...)
}

func (c *Config) Int(name string, idx int) (int64, error) {
	return c.access().Int(name, idx, configOpts...)
}

func (c *Config) Float(name string, idx int) (float64, error) {
	return c.access().Float(name, idx, configOpts...)
}

func (c *Config) Child(name string, idx int) (*Config, error) {
	sub, err := c.access().Child(name, idx, configOpts...)
	return fromConfig(sub), err
}

func (c *Config) SetBool(name string, idx int, value bool) error {
	return c.access().SetBool(name, idx, value, configOpts...)
}

func (c *Config) SetInt(name string, idx int, value int64) error {
	return c.access().SetInt(name, idx, value, configOpts...)
}

func (c *Config) SetFloat(name string, idx int, value float64) error {
	return c.access().SetFloat(name, idx, value, configOpts...)
}

func (c *Config) SetString(name string, idx int, value string) error {
	return c.access().SetString(name, idx, value, configOpts...)
}

func (c *Config) SetChild(name string, idx int, value *Config) error {
	return c.access().SetChild(name, idx, value.access(), configOpts...)
}

func (c *Config) IsDict() bool {
	return c.access().IsDict()
}

func (c *Config) IsArray() bool {
	return c.access().IsArray()
}

// FlattenedKeys return a sorted flattened views of the set keys in the configuration.
func (c *Config) FlattenedKeys() []string {
	return c.access().FlattenedKeys(configOpts...)
}

func (c *Config) PrintDebugf(msg string, params ...interface{}) {
	selector := selectorConfigWithPassword
	filtered := false
	if !hasSelector(selector) {
		selector = selectorConfig
		filtered = true

		if !hasSelector(selector) {
			return
		}
	}

	debugStr := DebugString(c, filtered)
	if debugStr != "" {
		configDebugf(selector, "%s\n%s", fmt.Sprintf(msg, params...), debugStr)
	}
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

func fromConfig(in *ucfg.Config) *Config {
	return (*Config)(in)
}

func (c *Config) access() *ucfg.Config {
	return (*ucfg.Config)(c)
}

func (c *Config) GetFields() []string {
	return c.access().GetFields()
}

// Unpack unpacks a configuration with at most one sub object. An sub object is
// ignored if it is disabled by setting `enabled: false`. If the configuration
// passed contains multiple active sub objects, Unpack will return an error.
func (ns *ConfigNamespace) Unpack(cfg *Config) error {
	fields := cfg.GetFields()
	if len(fields) == 0 {
		return nil
	}

	var (
		err   error
		found bool
	)

	for _, name := range fields {
		var sub *Config

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
func (ns *ConfigNamespace) Name() string {
	return ns.name
}

// Config return the sub-configuration section if a section has been set.
func (ns *ConfigNamespace) Config() *Config {
	return ns.config
}

// IsSet returns true if a sub-configuration section has been set.
func (ns *ConfigNamespace) IsSet() bool {
	return ns.config != nil
}

// DebugString prints a human readable representation of the underlying config using
// JSON formatting.
func DebugString(c *Config, filterPrivate bool) string {
	var bufs []string

	if c.IsDict() {
		var content map[string]interface{}
		if err := c.Unpack(&content); err != nil {
			return fmt.Sprintf("<config error> %v", err)
		}
		if filterPrivate {
			applyLoggingMask(content)
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
			applyLoggingMask(content)
		}
		j, _ := json.MarshalIndent(content, "", "  ")
		bufs = append(bufs, string(j))
	}

	if len(bufs) == 0 {
		return ""
	}
	return strings.Join(bufs, "\n")
}

// OwnerHasExclusiveWritePerms asserts that the current user or root is the
// owner of the config file and that the config file is (at most) writable by
// the owner or root (e.g. group and other cannot have write access).
func OwnerHasExclusiveWritePerms(name string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := file.Stat(name)
	if err != nil {
		return err
	}

	euid := os.Geteuid()
	fileUID, _ := info.UID()
	perm := info.Mode().Perm()

	if fileUID != 0 && euid != fileUID {
		return fmt.Errorf(`config file ("%v") must be owned by the user identifier `+
			`(uid=%v) or root`, name, euid)
	}

	// Test if group or other have write permissions.
	if perm&0022 > 0 {
		nameAbs, err := filepath.Abs(name)
		if err != nil {
			nameAbs = name
		}
		return fmt.Errorf(`config file ("%v") can only be writable by the `+
			`owner but the permissions are "%v" (to fix the permissions use: `+
			`'chmod go-w %v')`,
			name, perm, nameAbs)
	}

	return nil
}
