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
	tError     = reflect.TypeOf((*error)(nil)).Elem()
	tValidator = reflect.TypeOf((*Validator)(nil)).Elem()

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

// HasField checks if c has a top-level named key name.
func (c *Config) HasField(name string) bool {
	_, ok := c.fields.get(name)
	return ok
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
