package ucfg

import (
	"errors"
	"reflect"
)

type Config struct {
	fields map[string]value
}

var (
	ErrMissing = errors.New("field name missing")

	ErrTypeNoArray = errors.New("field is no array")

	ErrTypeMismatch = errors.New("type mismatch")

	ErrIndexOutOfRange = errors.New("index out of range")

	ErrPointerRequired = errors.New("requires pointer for unpacking")

	ErrArraySizeMistach = errors.New("Array size mismatch")

	ErrTODO = errors.New("TODO - implement me")
)

var (
	tConfig         = reflect.TypeOf(Config{})
	tConfigMap      = reflect.TypeOf((map[string]interface{})(nil))
	tInterfaceArray = reflect.TypeOf([]interface{}(nil))
)

func New() *Config {
	return &Config{
		fields: make(map[string]value),
	}
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
