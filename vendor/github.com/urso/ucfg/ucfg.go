package ucfg

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

type Config struct {
	fields map[string]value
}

type Option func(*options)

type options struct {
	tag     string
	pathSep string
}

var (
	ErrMissing = errors.New("field name missing")

	ErrTypeNoArray = errors.New("field is no array")

	ErrTypeMismatch = errors.New("type mismatch")

	ErrKeyTypeNotString = errors.New("key must be a string")

	ErrIndexOutOfRange = errors.New("index out of range")

	ErrPointerRequired = errors.New("requires pointer for unpacking")

	ErrArraySizeMistach = errors.New("Array size mismatch")

	ErrExpectedObject = errors.New("expected object")

	ErrNilConfig = errors.New("config is nil")

	ErrNilValue = errors.New("unexpected nil value")

	ErrTODO = errors.New("TODO - implement me")
)

var (
	tConfig         = reflect.TypeOf(Config{})
	tConfigMap      = reflect.TypeOf((map[string]interface{})(nil))
	tInterfaceArray = reflect.TypeOf([]interface{}(nil))
	tDuration       = reflect.TypeOf(time.Duration(0))

	tBool    = reflect.TypeOf(true)
	tInt64   = reflect.TypeOf(int64(0))
	tFloat64 = reflect.TypeOf(float64(0))
	tString  = reflect.TypeOf("")
)

func New() *Config {
	return &Config{
		fields: make(map[string]value),
	}
}

func NewFrom(from interface{}, opts ...Option) (*Config, error) {
	c := New()
	if err := c.Merge(from, opts...); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) GetFields() []string {
	var names []string
	for k := range c.fields {
		names = append(names, k)
	}
	return names
}

func (c *Config) HasField(name string) bool {
	_, ok := c.fields[name]
	return ok
}

func StructTag(tag string) Option {
	return func(o *options) {
		o.tag = tag
	}
}

func PathSep(sep string) Option {
	return func(o *options) {
		o.pathSep = sep
	}
}

func makeOptions(opts []Option) options {
	o := options{
		tag:     "config",
		pathSep: "", // no separator by default
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func errDuplicateKey(name string) error {
	return fmt.Errorf("duplicate field key '%v'", name)
}

func raise(err error) error {
	// fmt.Println(string(debug.Stack()))
	return err
}
