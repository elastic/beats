package ucfg

import (
	"fmt"
	"reflect"
)

type value interface {
	cpy(c context) value

	Context() context
	SetContext(c context)

	meta() *Meta
	setMeta(m *Meta)

	Len() int

	reflect() reflect.Value
	typ() reflect.Type
	reify() interface{}

	typeName() string

	toBool() (bool, error)
	toString() (string, error)
	toInt() (int64, error)
	toFloat() (float64, error)
	toConfig() (*Config, error)
}

type context struct {
	parent value
	field  string
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

type cfgNil struct{ cfgPrimitive }

type cfgPrimitive struct {
	ctx      context
	metadata *Meta
}

func (c *context) empty() bool {
	return c.parent == nil
}

func (c *context) path(sep string) string {
	if c.field == "" {
		return ""
	}

	if c.parent != nil {
		p := c.parent.Context()
		if parent := p.path(sep); parent != "" {
			return fmt.Sprintf("%v%v%v", parent, sep, c.field)
		}
	}

	return c.field
}

func (c *context) pathOf(field, sep string) string {
	if p := c.path(sep); p != "" {
		return fmt.Sprintf("%v%v%v", p, sep, field)
	}
	return field
}

func newBool(ctx context, m *Meta, b bool) *cfgBool {
	return &cfgBool{cfgPrimitive{ctx, m}, b}
}

func newInt(ctx context, m *Meta, i int64) *cfgInt {
	return &cfgInt{cfgPrimitive{ctx, m}, i}
}

func newFloat(ctx context, m *Meta, f float64) *cfgFloat {
	return &cfgFloat{cfgPrimitive{ctx, m}, f}
}

func newString(ctx context, m *Meta, s string) *cfgString {
	return &cfgString{cfgPrimitive{ctx, m}, s}
}

func (p *cfgPrimitive) Context() context        { return p.ctx }
func (p *cfgPrimitive) SetContext(c context)    { p.ctx = c }
func (p *cfgPrimitive) meta() *Meta             { return p.metadata }
func (p *cfgPrimitive) setMeta(m *Meta)         { p.metadata = m }
func (cfgPrimitive) Len() int                   { return 1 }
func (cfgPrimitive) toBool() (bool, error)      { return false, ErrTypeMismatch }
func (cfgPrimitive) toString() (string, error)  { return "", ErrTypeMismatch }
func (cfgPrimitive) toInt() (int64, error)      { return 0, ErrTypeMismatch }
func (cfgPrimitive) toFloat() (float64, error)  { return 0, ErrTypeMismatch }
func (cfgPrimitive) toConfig() (*Config, error) { return nil, ErrTypeMismatch }

func (cfgArray) typeName() string          { return "array" }
func (c *cfgArray) Len() int               { return len(c.arr) }
func (c *cfgArray) reflect() reflect.Value { return reflect.ValueOf(c.arr) }
func (cfgArray) typ() reflect.Type         { return tInterfaceArray }

func (c *cfgArray) cpy(ctx context) value {
	return &cfgArray{cfgPrimitive{ctx, c.meta()}, c.arr}
}

func (c *cfgArray) reify() interface{} {
	r := make([]interface{}, len(c.arr))
	for i, v := range c.arr {
		r[i] = v.reify()
	}
	return r
}

func (c *cfgNil) cpy(ctx context) value   { return &cfgNil{cfgPrimitive{ctx, c.metadata}} }
func (*cfgNil) Len() int                  { return 0 }
func (*cfgNil) typeName() string          { return "any" }
func (*cfgNil) toString() (string, error) { return "null", nil }
func (*cfgNil) toInt() (int64, error)     { return 0, ErrTypeMismatch }
func (*cfgNil) toFloat() (float64, error) { return 0, ErrTypeMismatch }
func (*cfgNil) reify() interface{}        { return nil }
func (*cfgNil) typ() reflect.Type         { return reflect.PtrTo(tConfig) }
func (c *cfgNil) meta() *Meta             { return c.metadata }
func (c *cfgNil) setMeta(m *Meta)         { c.metadata = m }

func (c *cfgNil) reflect() reflect.Value {
	cfg, _ := c.toConfig()
	return reflect.ValueOf(cfg)
}

func (c *cfgNil) toConfig() (*Config, error) {
	n := New()
	n.ctx = c.ctx
	return n, nil
}

func (c *cfgBool) cpy(ctx context) value     { return newBool(ctx, c.meta(), c.b) }
func (*cfgBool) typeName() string            { return "bool" }
func (c *cfgBool) toBool() (bool, error)     { return c.b, nil }
func (c *cfgBool) reflect() reflect.Value    { return reflect.ValueOf(c.b) }
func (c *cfgBool) reify() interface{}        { return c.b }
func (c *cfgBool) toString() (string, error) { return fmt.Sprintf("%t", c.b), nil }
func (c *cfgBool) typ() reflect.Type         { return tBool }

func (c *cfgInt) cpy(ctx context) value     { return newInt(ctx, c.meta(), c.i) }
func (*cfgInt) typeName() string            { return "int" }
func (c *cfgInt) toInt() (int64, error)     { return c.i, nil }
func (c *cfgInt) reflect() reflect.Value    { return reflect.ValueOf(c.i) }
func (c *cfgInt) reify() interface{}        { return c.i }
func (c *cfgInt) toString() (string, error) { return fmt.Sprintf("%d", c.i), nil }
func (c *cfgInt) typ() reflect.Type         { return tInt64 }

func (c *cfgFloat) cpy(ctx context) value     { return newFloat(ctx, c.meta(), c.f) }
func (*cfgFloat) typeName() string            { return "float" }
func (c *cfgFloat) toFloat() (float64, error) { return c.f, nil }
func (c *cfgFloat) reflect() reflect.Value    { return reflect.ValueOf(c.f) }
func (c *cfgFloat) reify() interface{}        { return c.f }
func (c *cfgFloat) toString() (string, error) { return fmt.Sprintf("%v", c.f), nil }
func (c *cfgFloat) typ() reflect.Type         { return tFloat64 }

func (c *cfgString) cpy(ctx context) value     { return newString(ctx, c.meta(), c.s) }
func (*cfgString) typeName() string            { return "string" }
func (c *cfgString) toString() (string, error) { return c.s, nil }
func (c *cfgString) reflect() reflect.Value    { return reflect.ValueOf(c.s) }
func (c *cfgString) reify() interface{}        { return c.s }
func (c *cfgString) typ() reflect.Type         { return tString }

func (cfgSub) Len() int                     { return 1 }
func (cfgSub) typeName() string             { return "object" }
func (c cfgSub) Context() context           { return c.c.ctx }
func (cfgSub) toBool() (bool, error)        { return false, ErrTypeMismatch }
func (cfgSub) toString() (string, error)    { return "", ErrTypeMismatch }
func (cfgSub) toInt() (int64, error)        { return 0, ErrTypeMismatch }
func (cfgSub) toFloat() (float64, error)    { return 0, ErrTypeMismatch }
func (c cfgSub) toConfig() (*Config, error) { return c.c, nil }
func (cfgSub) typ() reflect.Type            { return reflect.PtrTo(tConfig) }
func (c cfgSub) reflect() reflect.Value     { return reflect.ValueOf(c.c) }
func (c cfgSub) meta() *Meta                { return c.c.metadata }
func (c cfgSub) setMeta(m *Meta)            { c.c.metadata = m }

func (c cfgSub) cpy(ctx context) value {
	return cfgSub{
		c: &Config{ctx: ctx, fields: c.c.fields, metadata: c.c.metadata},
	}
}

func (c cfgSub) SetContext(ctx context) {
	if c.c.ctx.empty() {
		c.c.ctx = ctx
	} else {
		c.c = &Config{
			ctx:    ctx,
			fields: c.c.fields,
		}
	}
}

func (c cfgSub) reify() interface{} {
	m := make(map[string]interface{})
	for k, v := range c.c.fields.fields {
		m[k] = v.reify()
	}
	return m
}
