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

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"time"
)

// Config object to store hierarchical configurations into. Config can be
// both a dictionary and a list holding primitive values. Primitive values
// can be booleans, integers, float point numbers and strings.
//
// Config provides a low level interface for setting and getting settings
// via SetBool, SetInt, SetUing, SetFloat, SetString, SetChild, Bool, Int, Uint,
// Float, String, and Child.
//
// A more user-friendly high level interface is provided via Unpack and Merge.
type Config struct {
	ctx      context
	metadata *Meta
	fields   *fields
}

type fieldOptions struct {
	opts       *options
	tag        tagOptions
	validators []validatorTag
}

type fields struct {
	d map[string]value
	a []value
}

// Meta holds additional meta data per config value.
type Meta struct {
	Source string
}

var (
	tConfig         = reflect.TypeOf(Config{})
	tConfigPtr      = reflect.PtrTo(tConfig)
	tConfigMap      = reflect.TypeOf((map[string]interface{})(nil))
	tInterfaceArray = reflect.TypeOf([]interface{}(nil))

	// interface types
	tError       = reflect.TypeOf((*error)(nil)).Elem()
	iInitializer = reflect.TypeOf((*Initializer)(nil)).Elem()
	tValidator   = reflect.TypeOf((*Validator)(nil)).Elem()

	// primitives
	tBool     = reflect.TypeOf(true)
	tInt64    = reflect.TypeOf(int64(0))
	tUint64   = reflect.TypeOf(uint64(0))
	tFloat64  = reflect.TypeOf(float64(0))
	tString   = reflect.TypeOf("")
	tDuration = reflect.TypeOf(time.Duration(0))
	tRegexp   = reflect.TypeOf(regexp.Regexp{})
)

// New creates a new empty Config object.
func New() *Config {
	return &Config{
		fields: &fields{nil, nil},
	}
}

// MustNewFrom creates a new config object normalizing and copying from into the new
// Config object. MustNewFrom uses Merge to copy from.
//
// MustNewFrom supports the options: PathSep, MetaData, StructTag, VarExp
func MustNewFrom(from interface{}, opts ...Option) *Config {
	c := New()
	if err := c.Merge(from, opts...); err != nil {
		panic(err)
	}
	return c
}

// NewFrom creates a new config object normalizing and copying from into the new
// Config object. NewFrom uses Merge to copy from.
//
// NewFrom supports the options: PathSep, MetaData, StructTag, VarExp
func NewFrom(from interface{}, opts ...Option) (*Config, error) {
	c := New()
	if err := c.Merge(from, opts...); err != nil {
		return nil, err
	}
	return c, nil
}

// IsDict checks if c has named keys.
func (c *Config) IsDict() bool {
	return c.fields.dict() != nil
}

// IsArray checks if c has index only accessible settings.
func (c *Config) IsArray() bool {
	return c.fields.array() != nil
}

// GetFields returns a list of all top-level named keys in c.
func (c *Config) GetFields() []string {
	var names []string
	for k := range c.fields.dict() {
		names = append(names, k)
	}
	return names
}

// Has checks if a field by the given path+idx configuration exists.
// Has returns an error if the path can not be resolved because a primitive
// value is found in the middle of the traversal.
func (c *Config) Has(name string, idx int, options ...Option) (bool, error) {
	opts := makeOptions(options)
	p := parsePathIdx(name, opts.pathSep, idx)
	return p.Has(c, opts)
}

// HasField checks if c has a top-level named key name.
func (c *Config) HasField(name string) bool {
	_, ok := c.fields.get(name)
	return ok
}

// Remove removes a setting from the config. If the configuration references
// another configuration namespace, then the setting will be removed from the
// linked reference.
// Remove returns true if the setting was removed. If the path can't be
// resolved (e.g. due to type mismatch) Remove will return an error.
//
// Settings can be created on Unpack via Env, Resolve, and ResolveEnv. Settings
// generated dynamically on Unpack can not be removed. Remove ignores any
// configured environments and will return an error if a value can not be
// removed for this reason.
//
// The setting path is constructed from name and idx. If name is set and idx is -1,
// only the name is used to access the setting by name. If name is empty, idx
// must be >= 0, assuming the Config is a list. If both name and idx are set,
// the name must point to a list.
//
// Remove supports the options: PathSep
func (c *Config) Remove(name string, idx int, options ...Option) (bool, error) {
	opts := makeOptions(options)

	// ignore environments
	opts.env = nil
	opts.resolvers = nil
	opts.noParse = true

	p := parsePathIdx(name, opts.pathSep, idx)
	return p.Remove(c, opts)
}

// Path gets the absolute path of c separated by sep. If c is a root-Config an
// empty string will be returned.
func (c *Config) Path(sep string) string {
	return c.ctx.path(sep)
}

// PathOf gets the absolute path of a potential setting field in c with name
// separated by sep.
func (c *Config) PathOf(field, sep string) string {
	return c.ctx.pathOf(field, sep)
}

// Parent returns the parent configuration or nil if c is already a root
// Configuration.
func (c *Config) Parent() *Config {
	ctx := c.ctx
	for {
		if ctx.parent == nil {
			return nil
		}

		switch p := ctx.parent.(type) {
		case cfgSub:
			return p.c
		default:
			return nil
		}
	}
}

// FlattenedKeys return a sorted flattened views of the set keys in the configuration
func (c *Config) FlattenedKeys(opts ...Option) []string {
	var keys []string
	normalizedOptions := makeOptions(opts)

	if normalizedOptions.pathSep == "" {
		normalizedOptions.pathSep = "."
	}

	if c.IsDict() {
		for _, v := range c.fields.dict() {

			subcfg, err := v.toConfig(normalizedOptions)
			if err != nil {
				ctx := v.Context()
				p := ctx.path(normalizedOptions.pathSep)
				keys = append(keys, p)
			} else {
				newKeys := subcfg.FlattenedKeys(opts...)
				keys = append(keys, newKeys...)
			}
		}
	} else if c.IsArray() {
		for _, a := range c.fields.array() {
			scfg, err := a.toConfig(normalizedOptions)

			if err != nil {
				ctx := a.Context()
				p := ctx.path(normalizedOptions.pathSep)
				keys = append(keys, p)
			} else {
				newKeys := scfg.FlattenedKeys(opts...)
				keys = append(keys, newKeys...)
			}
		}
	}

	sort.Strings(keys)
	return keys
}

func (f *fields) get(name string) (value, bool) {
	if f.d == nil {
		return nil, false
	}
	v, found := f.d[name]
	return v, found
}

func (f *fields) dict() map[string]value {
	return f.d
}

func (f *fields) array() []value {
	return f.a
}

func (f *fields) del(name string) bool {
	_, exists := f.d[name]
	if exists {
		delete(f.d, name)
	}
	return exists
}

func (f *fields) delAt(i int) bool {
	a := f.a
	if i < 0 || len(a) <= i {
		return false
	}

	copy(a[i:], a[i+1:])
	a[len(a)-1] = nil
	f.a = a[:len(a)-1]
	return true
}

func (f *fields) set(name string, v value) {
	if f.d == nil {
		f.d = map[string]value{}
	}
	f.d[name] = v
}

func (f *fields) add(v value) {
	f.a = append(f.a, v)
}

func (f *fields) setAt(idx int, parent, v value) {
	l := len(f.a)
	if idx >= l {
		tmp := make([]value, idx+1)
		copy(tmp, f.a)

		for i := l; i < idx; i++ {
			ctx := context{parent: parent, field: fmt.Sprintf("%d", i)}
			tmp[i] = &cfgNil{cfgPrimitive{ctx, nil}}
		}

		f.a = tmp
	}

	f.a[idx] = v
}

func (f *fields) append(parent value, a []value) {
	l := len(f.a)
	count := len(a)
	if count == 0 {
		return
	}

	for i := 0; i < count; i, l = i+1, l+1 {
		ctx := context{
			parent: parent,
			field:  fmt.Sprintf("%v", l),
		}
		f.setAt(l, parent, a[i].cpy(ctx))
	}
}

func (o *fieldOptions) configHandling() configHandling {
	h := o.tag.cfgHandling
	if h == cfgDefaultHandling {
		h = o.opts.configValueHandling
	}
	return h
}
