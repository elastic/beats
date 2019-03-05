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

package ucfg

// ******************************************************************************
// Low level getters and setters
// ******************************************************************************

func convertErr(opts *options, v value, err error, to string) Error {
	if err == nil {
		return nil
	}
	return raiseConversion(opts, v, err, to)
}

// CountField returns number of entries in a table or 1 if entry is a primitive value.
// Primitives settings can be handled like a list with 1 entry.
//
// If name is empty, the total number of top-level settings is returned.
//
// CountField supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) CountField(name string, opts ...Option) (int, error) {
	if name == "" {
		return len(c.fields.array()) + len(c.fields.dict()), nil
	}

	if v, ok := c.fields.get(name); ok {
		return v.Len(makeOptions(opts))
	}
	return -1, raiseMissing(c, name)
}

// Bool reads a boolean setting returning an error if the setting has no
// boolean value.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// Bool supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) Bool(name string, idx int, opts ...Option) (bool, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return false, err
	}
	b, fail := v.toBool(O)
	return b, convertErr(O, v, fail, "bool")
}

// Strings reads a string setting returning an error if the setting has
// no string or primitive value convertible to string.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// String supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) String(name string, idx int, opts ...Option) (string, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return "", err
	}
	s, fail := v.toString(O)
	return s, convertErr(O, v, fail, "string")
}

// Int reads an int64 value returning an error if the setting is
// not integer value, the primitive value is not convertible to int or a conversion
// would create an integer overflow.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// Int supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) Int(name string, idx int, opts ...Option) (int64, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return 0, err
	}

	i, fail := v.toInt(O)
	return i, convertErr(O, v, fail, "int")
}

// Uint reads an uint64 value returning an error if the setting is
// not unsigned value, the primitive value is not convertible to uint64 or a conversion
// would create an integer overflow.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// Uint supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) Uint(name string, idx int, opts ...Option) (uint64, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return 0, err
	}
	u, fail := v.toUint(O)
	return u, convertErr(O, v, fail, "uint")
}

// Float reads a float64 value returning an error if the setting is
// not a float value or the primitive value is not convertible to float.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// Float supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) Float(name string, idx int, opts ...Option) (float64, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return 0, err
	}
	f, fail := v.toFloat(O)
	return f, convertErr(O, v, fail, "float")
}

// Child returns a child configuration or an error if the setting requested is a
// primitive value only.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// Child supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) Child(name string, idx int, opts ...Option) (*Config, error) {
	O := makeOptions(opts)
	v, err := c.getField(name, idx, O)
	if err != nil {
		return nil, err
	}
	c, fail := v.toConfig(O)
	return c, convertErr(O, v, fail, "object")
}

// SetBool sets a boolean primitive value. An error is returned if the new name
// is invalid.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// SetBool supports the options: PathSep, MetaData
func (c *Config) SetBool(name string, idx int, value bool, opts ...Option) error {
	return c.setField(name, idx, &cfgBool{b: value}, opts)
}

// SetInt sets an integer primitive value. An error is returned if the new
// name is invalid.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// SetInt supports the options: PathSep, MetaData
func (c *Config) SetInt(name string, idx int, value int64, opts ...Option) error {
	return c.setField(name, idx, &cfgInt{i: value}, opts)
}

// SetUint sets an unsigned integer primitive value. An error is returned if
// the name is invalid.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// SetUint supports the options: PathSep, MetaData
func (c *Config) SetUint(name string, idx int, value uint64, opts ...Option) error {
	return c.setField(name, idx, &cfgUint{u: value}, opts)
}

// SetFloat sets an floating point primitive value. An error is returned if
// the name is invalid.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// SetFloat supports the options: PathSep, MetaData
func (c *Config) SetFloat(name string, idx int, value float64, opts ...Option) error {
	return c.setField(name, idx, &cfgFloat{f: value}, opts)
}

// SetString sets string value. An error is returned if the name is invalid.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// SetString supports the options: PathSep, MetaData
func (c *Config) SetString(name string, idx int, value string, opts ...Option) error {
	return c.setField(name, idx, &cfgString{s: value}, opts)
}

// SetChild adds a sub-configuration. An error is returned if the name is invalid.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list. The number of entries in a named list can be read
// using CountField.
//
// SetChild supports the options: PathSep, MetaData
func (c *Config) SetChild(name string, idx int, value *Config, opts ...Option) error {
	return c.setField(name, idx, cfgSub{c: value}, opts)
}

// getField supports the options: PathSep, Env, Resolve, ResolveEnv
func (c *Config) getField(name string, idx int, opts *options) (value, Error) {
	p := parsePathIdx(name, opts.pathSep, idx)
	v, err := p.GetValue(c, opts)
	if err != nil {
		return v, err
	}

	if v == nil {
		return nil, raiseMissing(c, p.String())
	}
	return v, nil
}

// setField supports the options: PathSep, MetaData
func (c *Config) setField(name string, idx int, v value, options []Option) Error {
	opts := makeOptions(options)
	p := parsePathIdx(name, opts.pathSep, idx)

	err := p.SetValue(c, opts, v)
	if err != nil {
		return err
	}

	if opts.meta != nil {
		v.setMeta(opts.meta)
	}
	return nil
}
