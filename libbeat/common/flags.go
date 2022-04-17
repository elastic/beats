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
	"flag"
	"strings"

	ucfg "github.com/menderesk/go-ucfg"
	cfgflag "github.com/menderesk/go-ucfg/flag"
)

// StringsFlag collects multiple usages of the same flag into an array of strings.
// Duplicate values will be ignored.
type StringsFlag struct {
	list      *[]string
	isDefault bool
	flag      *flag.Flag
}

// SettingsFlag captures key/values pairs into an Config object.
// The flag backed by SettingsFlag can be used multiple times.
// Values are overwritten by the last usage of a key.
type SettingsFlag cfgflag.FlagValue

// flagOverwrite provides a flag value, which always overwrites the same setting
// in an Config object.
type flagOverwrite struct {
	config *ucfg.Config
	path   string
	value  string
}

// StringArrFlag creates and registers a new StringsFlag with the given FlagSet.
// If no FlagSet is passed, flag.CommandLine will be used as target FlagSet.
func StringArrFlag(fs *flag.FlagSet, name, def, usage string) *StringsFlag {
	var arr *[]string
	if def != "" {
		arr = &[]string{def}
	} else {
		arr = &[]string{}
	}

	return StringArrVarFlag(fs, arr, name, usage)
}

// StringArrVarFlag creates and registers a new StringsFlag with the given
// FlagSet.  Results of the flag usage will be appended to `arr`. If the slice
// is not initially empty, its first value will be used as default. If the flag
// is used, the slice will be emptied first.  If no FlagSet is passed,
// flag.CommandLine will be used as target FlagSet.
func StringArrVarFlag(fs *flag.FlagSet, arr *[]string, name, usage string) *StringsFlag {
	if fs == nil {
		fs = flag.CommandLine
	}
	f := NewStringsFlag(arr)
	f.Register(fs, name, usage)
	return f
}

// NewStringsFlag creates a new, but unregistered StringsFlag instance.
// Results of the flag usage will be appended to `arr`. If the slice is not
// initially empty, its first value will be used as default. If the flag is
// used, the slice will be emptied first.
func NewStringsFlag(arr *[]string) *StringsFlag {
	if arr == nil {
		panic("No target array")
	}
	return &StringsFlag{list: arr, isDefault: true}
}

// Register registers the StringsFlag instance with a FlagSet.
// A valid FlagSet must be used.
// Register panics if the flag is already registered.
func (f *StringsFlag) Register(fs *flag.FlagSet, name, usage string) {
	if f.flag != nil {
		panic("StringsFlag is already registered")
	}

	fs.Var(f, name, usage)
	f.flag = fs.Lookup(name)
	if f.flag == nil {
		panic("Failed to lookup registered flag")
	}

	if len(*f.list) > 0 {
		f.flag.DefValue = (*f.list)[0]
	}
}

// String joins all it's values set into a comma-separated string.
func (f *StringsFlag) String() string {
	if f == nil || f.list == nil {
		return ""
	}

	l := *f.list
	return strings.Join(l, ", ")
}

// SetDefault sets the flags new default value.
// This overwrites the contents in the backing array.
func (f *StringsFlag) SetDefault(v string) {
	if f.flag != nil {
		f.flag.DefValue = v
	}

	*f.list = []string{v}
	f.isDefault = true
}

// Set is used to pass usage of the flag to StringsFlag. Set adds the new value
// to the backing array. The array will be emptied on Set, if the backing array
// still contains the default value.
func (f *StringsFlag) Set(v string) error {
	// Ignore duplicates, can be caused by multiple flag parses
	if f.isDefault {
		*f.list = []string{v}
	} else {
		for _, old := range *f.list {
			if old == v {
				return nil
			}
		}
		*f.list = append(*f.list, v)
	}
	f.isDefault = false
	return nil
}

// Get returns the backing slice its contents as interface{}. The type used is
// `[]string`.
func (f *StringsFlag) Get() interface{} {
	return f.List()
}

// List returns the current set values.
func (f *StringsFlag) List() []string {
	return *f.list
}

// Type reports the type of contents (string) expected to be parsed by Set.
// It is used to build the CLI usage string.
func (f *StringsFlag) Type() string {
	return "string"
}

// SettingFlag defines a setting flag, name and it's usage. The return value is
// the Config object settings are applied to.
func SettingFlag(fs *flag.FlagSet, name, usage string) *Config {
	cfg := NewConfig()
	SettingVarFlag(fs, cfg, name, usage)
	return cfg
}

// SettingVarFlag defines a setting flag, name and it's usage.
// Settings are applied to the Config object passed.
func SettingVarFlag(fs *flag.FlagSet, def *Config, name, usage string) {
	if fs == nil {
		fs = flag.CommandLine
	}

	f := NewSettingsFlag(def)
	fs.Var(f, name, usage)
}

// NewSettingsFlag creates a new SettingsFlag instance, not registered with any
// FlagSet.
func NewSettingsFlag(def *Config) *SettingsFlag {
	opts := append(
		[]ucfg.Option{
			ucfg.MetaData(ucfg.Meta{Source: "command line flag"}),
		},
		configOpts...,
	)

	tmp := cfgflag.NewFlagKeyValue(def.access(), true, opts...)
	return (*SettingsFlag)(tmp)
}

func (f *SettingsFlag) access() *cfgflag.FlagValue {
	return (*cfgflag.FlagValue)(f)
}

// Config returns the config object the SettingsFlag stores applied settings to.
func (f *SettingsFlag) Config() *Config {
	return fromConfig(f.access().Config())
}

// Set sets a settings value in the Config object.  The input string must be a
// key-value pair like `key=value`. If the value is missing, the value is set
// to the boolean value `true`.
func (f *SettingsFlag) Set(s string) error {
	return f.access().Set(s)
}

// Get returns the Config object used to store values.
func (f *SettingsFlag) Get() interface{} {
	return f.Config()
}

// String always returns an empty string. It is required to fulfil
// the flag.Value interface.
func (f *SettingsFlag) String() string {
	return ""
}

// Type reports the type of contents (setting=value) expected to be parsed by Set.
// It is used to build the CLI usage string.
func (f *SettingsFlag) Type() string {
	return "setting=value"
}

// ConfigOverwriteFlag defines a new flag updating a setting in an Config
// object.  The name is used as the flag its name the path parameter is the
// full setting name to be used when the flag is set.
func ConfigOverwriteFlag(
	fs *flag.FlagSet,
	config *Config,
	name, path, def, usage string,
) *string {
	if config == nil {
		panic("Missing configuration")
	}
	if path == "" {
		panic("empty path")
	}

	if fs == nil {
		fs = flag.CommandLine
	}

	if def != "" {
		err := config.SetString(path, -1, def)
		if err != nil {
			panic(err)
		}
	}

	f := newOverwriteFlag(config, path, def)
	fs.Var(f, name, usage)
	return &f.value
}

func newOverwriteFlag(config *Config, path, def string) *flagOverwrite {
	return &flagOverwrite{config: config.access(), path: path, value: def}
}

func (f *flagOverwrite) String() string {
	return f.value
}

func (f *flagOverwrite) Set(v string) error {
	opts := append(
		[]ucfg.Option{
			ucfg.MetaData(ucfg.Meta{Source: "command line flag"}),
		},
		configOpts...,
	)

	err := f.config.SetString(f.path, -1, v, opts...)
	if err != nil {
		return err
	}
	f.value = v
	return nil
}

func (f *flagOverwrite) Get() interface{} {
	return f.value
}

func (f *flagOverwrite) Type() string {
	return "string"
}
