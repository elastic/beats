package ucfg

import (
	"reflect"
	"regexp"
	"time"
)

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

// Meta holds additional meta data per config value
type Meta struct {
	Source string
}

type Unpacker interface {
	Unpack(interface{}) error
}

var (
	tConfig         = reflect.TypeOf(Config{})
	tConfigPtr      = reflect.PtrTo(tConfig)
	tConfigMap      = reflect.TypeOf((map[string]interface{})(nil))
	tInterfaceArray = reflect.TypeOf([]interface{}(nil))

	// interface types
	tUnpacker  = reflect.TypeOf((*Unpacker)(nil)).Elem()
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

func New() *Config {
	return &Config{
		fields: &fields{nil, nil},
	}
}

func NewFrom(from interface{}, opts ...Option) (*Config, error) {
	c := New()
	if err := c.Merge(from, opts...); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) IsDict() bool {
	return c.fields.dict() != nil
}

func (c *Config) IsArray() bool {
	return c.fields.array() != nil
}

func (c *Config) GetFields() []string {
	var names []string
	for k := range c.fields.dict() {
		names = append(names, k)
	}
	return names
}

func (c *Config) HasField(name string) bool {
	_, ok := c.fields.get(name)
	return ok
}

func (c *Config) Path(sep string) string {
	return c.ctx.path(sep)
}

func (c *Config) PathOf(field, sep string) string {
	return c.ctx.pathOf(field, sep)
}

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

func (f *fields) setAt(idx int, v value) {
	if idx >= len(f.a) {
		tmp := make([]value, idx+1)
		copy(tmp, f.a)
		f.a = tmp
	}
	f.a[idx] = v
}
