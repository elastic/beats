package ucfg

import "reflect"

type value interface {
	Len() int

	reflect() reflect.Value
	reify() interface{}

	toBool() (bool, error)
	toString() (string, error)
	toInt() (int64, error)
	toFloat() (float64, error)
	toConfig() (*Config, error)
}

type cfgArray struct {
	cfgPrimitive
	arr []value
}

type cfgBool struct {
	cfgPrimitive
	b bool
}

type cfgInt struct {
	cfgPrimitive
	i int64
}

type cfgFloat struct {
	cfgPrimitive
	f float64
}

type cfgString struct {
	cfgPrimitive
	s string
}

type cfgSub struct {
	c *Config
}

type cfgPrimitive struct{}

func (cfgPrimitive) Len() int                   { return 1 }
func (cfgPrimitive) toBool() (bool, error)      { return false, ErrTypeMismatch }
func (cfgPrimitive) toString() (string, error)  { return "", ErrTypeMismatch }
func (cfgPrimitive) toInt() (int64, error)      { return 0, ErrTypeMismatch }
func (cfgPrimitive) toFloat() (float64, error)  { return 0, ErrTypeMismatch }
func (cfgPrimitive) toConfig() (*Config, error) { return nil, ErrTypeMismatch }

func (c *cfgArray) Len() int               { return len(c.arr) }
func (c *cfgArray) reflect() reflect.Value { return reflect.ValueOf(c.arr) }
func (c *cfgArray) reify() interface{} {
	r := make([]interface{}, len(c.arr))
	for i, v := range c.arr {
		r[i] = v.reify()
	}
	return r
}

func (c *cfgBool) toBool() (bool, error)  { return c.b, nil }
func (c *cfgBool) reflect() reflect.Value { return reflect.ValueOf(c.b) }
func (c *cfgBool) reify() interface{}     { return c.b }

func (c *cfgInt) toInt() (int64, error)  { return c.i, nil }
func (c *cfgInt) reflect() reflect.Value { return reflect.ValueOf(c.i) }
func (c *cfgInt) reify() interface{}     { return c.i }

func (c *cfgFloat) toFloat() (float64, error) { return c.f, nil }
func (c *cfgFloat) reflect() reflect.Value    { return reflect.ValueOf(c.f) }
func (c *cfgFloat) reify() interface{}        { return c.f }

func (c *cfgString) toString() (string, error) { return c.s, nil }
func (c *cfgString) reflect() reflect.Value    { return reflect.ValueOf(c.s) }
func (c *cfgString) reify() interface{}        { return c.s }

func (cfgSub) Len() int                     { return 1 }
func (cfgSub) toBool() (bool, error)        { return false, ErrTypeMismatch }
func (cfgSub) toString() (string, error)    { return "", ErrTypeMismatch }
func (cfgSub) toInt() (int64, error)        { return 0, ErrTypeMismatch }
func (cfgSub) toFloat() (float64, error)    { return 0, ErrTypeMismatch }
func (c cfgSub) toConfig() (*Config, error) { return c.c, nil }
func (c cfgSub) reflect() reflect.Value     { return reflect.ValueOf(c.c) }
func (c cfgSub) reify() interface{} {
	m := make(map[string]interface{})
	for k, v := range c.c.fields {
		m[k] = v.reify()
	}
	return m
}

type cfgKind uint8

const (
	kindBool cfgKind = iota
	kindString
	kindInt
	kindFloat
	kindConfig
	kindArray
)
