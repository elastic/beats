package ucfg

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

type value interface {
	typ(opts *options) (typeInfo, error)

	cpy(c context) value

	Context() context
	SetContext(c context)

	meta() *Meta
	setMeta(m *Meta)

	Len(opts *options) (int, error)

	reflect(opts *options) (reflect.Value, error)
	reify(opts *options) (interface{}, error)

	toBool(opts *options) (bool, error)
	toString(opts *options) (string, error)
	toInt(opts *options) (int64, error)
	toUint(opts *options) (uint64, error)
	toFloat(opts *options) (float64, error)
	toConfig(opts *options) (*Config, error)
}

type typeInfo struct {
	name   string
	gotype reflect.Type
}

type context struct {
	parent value
	field  string
}

type cfgBool struct {
	cfgPrimitive
	b bool
}

type cfgInt struct {
	cfgPrimitive
	i int64
}

type cfgUint struct {
	cfgPrimitive
	u uint64
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

type cfgRef struct {
	cfgPrimitive
	ref *reference
}

type cfgSplice struct {
	cfgPrimitive
	splice varEvaler
}

type cfgNil struct{ cfgPrimitive }

type cfgPrimitive struct {
	ctx      context
	metadata *Meta
}

func (c *context) empty() bool {
	return c.parent == nil
}

func (c *context) getParent() *Config {
	if c.parent == nil {
		return nil
	}

	if cfg, ok := c.parent.(cfgSub); ok {
		return cfg.c
	}
	return nil
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

func newUint(ctx context, m *Meta, u uint64) *cfgUint {
	return &cfgUint{cfgPrimitive{ctx, m}, u}
}

func newFloat(ctx context, m *Meta, f float64) *cfgFloat {
	return &cfgFloat{cfgPrimitive{ctx, m}, f}
}

func newString(ctx context, m *Meta, s string) *cfgString {
	return &cfgString{cfgPrimitive{ctx, m}, s}
}

func newRef(ctx context, m *Meta, ref *reference) *cfgRef {
	return &cfgRef{cfgPrimitive{ctx, m}, ref}
}

func newSplice(ctx context, m *Meta, s varEvaler) *cfgSplice {
	return &cfgSplice{cfgPrimitive{ctx, m}, s}
}

func (p *cfgPrimitive) Context() context                { return p.ctx }
func (p *cfgPrimitive) SetContext(c context)            { p.ctx = c }
func (p *cfgPrimitive) meta() *Meta                     { return p.metadata }
func (p *cfgPrimitive) setMeta(m *Meta)                 { p.metadata = m }
func (cfgPrimitive) Len(*options) (int, error)          { return 1, nil }
func (cfgPrimitive) toBool(*options) (bool, error)      { return false, ErrTypeMismatch }
func (cfgPrimitive) toString(*options) (string, error)  { return "", ErrTypeMismatch }
func (cfgPrimitive) toInt(*options) (int64, error)      { return 0, ErrTypeMismatch }
func (cfgPrimitive) toUint(*options) (uint64, error)    { return 0, ErrTypeMismatch }
func (cfgPrimitive) toFloat(*options) (float64, error)  { return 0, ErrTypeMismatch }
func (cfgPrimitive) toConfig(*options) (*Config, error) { return nil, ErrTypeMismatch }

func (c *cfgNil) cpy(ctx context) value             { return &cfgNil{cfgPrimitive{ctx, c.metadata}} }
func (*cfgNil) Len(*options) (int, error)           { return 0, nil }
func (*cfgNil) toString(*options) (string, error)   { return "null", nil }
func (*cfgNil) toInt(*options) (int64, error)       { return 0, ErrTypeMismatch }
func (*cfgNil) toUint(*options) (uint64, error)     { return 0, ErrTypeMismatch }
func (*cfgNil) toFloat(*options) (float64, error)   { return 0, ErrTypeMismatch }
func (*cfgNil) reify(*options) (interface{}, error) { return nil, nil }
func (*cfgNil) typ(*options) (typeInfo, error)      { return typeInfo{"any", reflect.PtrTo(tConfig)}, nil }
func (c *cfgNil) meta() *Meta                       { return c.metadata }
func (c *cfgNil) setMeta(m *Meta)                   { c.metadata = m }

func (c *cfgNil) reflect(opts *options) (reflect.Value, error) {
	cfg, _ := c.toConfig(opts)
	return reflect.ValueOf(cfg), nil
}

func (c *cfgNil) toConfig(*options) (*Config, error) {
	n := New()
	n.ctx = c.ctx
	return n, nil
}

func (c *cfgBool) cpy(ctx context) value                   { return newBool(ctx, c.meta(), c.b) }
func (c *cfgBool) toBool(*options) (bool, error)           { return c.b, nil }
func (c *cfgBool) reflect(*options) (reflect.Value, error) { return reflect.ValueOf(c.b), nil }
func (c *cfgBool) reify(*options) (interface{}, error)     { return c.b, nil }
func (c *cfgBool) toString(*options) (string, error)       { return fmt.Sprintf("%t", c.b), nil }
func (c *cfgBool) typ(*options) (typeInfo, error)          { return typeInfo{"bool", tBool}, nil }

func (c *cfgInt) cpy(ctx context) value                   { return newInt(ctx, c.meta(), c.i) }
func (c *cfgInt) toInt(*options) (int64, error)           { return c.i, nil }
func (c *cfgInt) reflect(*options) (reflect.Value, error) { return reflect.ValueOf(c.i), nil }
func (c *cfgInt) reify(*options) (interface{}, error)     { return c.i, nil }
func (c *cfgInt) toString(*options) (string, error)       { return fmt.Sprintf("%d", c.i), nil }
func (c *cfgInt) typ(*options) (typeInfo, error)          { return typeInfo{"int", tInt64}, nil }
func (c *cfgInt) toUint(*options) (uint64, error) {
	if c.i < 0 {
		return 0, ErrNegative
	}
	return uint64(c.i), nil
}

func (c *cfgUint) cpy(ctx context) value                   { return newUint(ctx, c.meta(), c.u) }
func (c *cfgUint) reflect(*options) (reflect.Value, error) { return reflect.ValueOf(c.u), nil }
func (c *cfgUint) reify(*options) (interface{}, error)     { return c.u, nil }
func (c *cfgUint) toString(*options) (string, error)       { return fmt.Sprintf("%d", c.u), nil }
func (c *cfgUint) typ(*options) (typeInfo, error)          { return typeInfo{"uint", tUint64}, nil }
func (c *cfgUint) toUint(*options) (uint64, error)         { return c.u, nil }
func (c *cfgUint) toInt(*options) (int64, error) {
	if c.u > math.MaxInt64 {
		return 0, ErrOverflow
	}
	return int64(c.u), nil
}

func (c *cfgFloat) cpy(ctx context) value                   { return newFloat(ctx, c.meta(), c.f) }
func (c *cfgFloat) toFloat(*options) (float64, error)       { return c.f, nil }
func (c *cfgFloat) reflect(*options) (reflect.Value, error) { return reflect.ValueOf(c.f), nil }
func (c *cfgFloat) reify(*options) (interface{}, error)     { return c.f, nil }
func (c *cfgFloat) toString(*options) (string, error)       { return fmt.Sprintf("%v", c.f), nil }
func (c *cfgFloat) typ(*options) (typeInfo, error)          { return typeInfo{"float", tFloat64}, nil }

func (c *cfgFloat) toUint(*options) (uint64, error) {
	if c.f < 0 {
		return 0, ErrNegative
	}
	if c.f > math.MaxUint64 {
		return 0, ErrOverflow
	}
	return uint64(c.f), nil
}

func (c *cfgFloat) toInt(*options) (int64, error) {
	if c.f < math.MinInt64 || math.MaxInt64 < c.f {
		return 0, ErrOverflow
	}
	return int64(c.f), nil
}

func (c *cfgString) cpy(ctx context) value             { return newString(ctx, c.meta(), c.s) }
func (c *cfgString) toString(*options) (string, error) { return c.s, nil }
func (c *cfgString) reflect(*options) (reflect.Value, error) {
	return reflect.ValueOf(c.s), nil
}
func (c *cfgString) reify(*options) (interface{}, error) { return c.s, nil }
func (c *cfgString) typ(*options) (typeInfo, error)      { return typeInfo{"string", tString}, nil }

func (c cfgSub) Context() context                   { return c.c.ctx }
func (cfgSub) toBool(*options) (bool, error)        { return false, ErrTypeMismatch }
func (cfgSub) toString(*options) (string, error)    { return "", ErrTypeMismatch }
func (cfgSub) toInt(*options) (int64, error)        { return 0, ErrTypeMismatch }
func (cfgSub) toUint(*options) (uint64, error)      { return 0, ErrTypeMismatch }
func (cfgSub) toFloat(*options) (float64, error)    { return 0, ErrTypeMismatch }
func (c cfgSub) toConfig(*options) (*Config, error) { return c.c, nil }

func (c cfgSub) Len(*options) (int, error) {
	arr := c.c.fields.array()
	if arr != nil {
		return len(arr), nil
	}

	return 1, nil
}

func (c cfgSub) typ(*options) (typeInfo, error) {
	return typeInfo{"object", reflect.PtrTo(tConfig)}, nil
}

// func (cfgSub) typ() (typeInfo, error)            { return typeInfo{"object", reflect.PtrTo(tConfig)}, nil }
func (c cfgSub) reflect(*options) (reflect.Value, error) { return reflect.ValueOf(c.c), nil }
func (c cfgSub) meta() *Meta                             { return c.c.metadata }
func (c cfgSub) setMeta(m *Meta)                         { c.c.metadata = m }

func (c cfgSub) cpy(ctx context) value {
	newC := cfgSub{
		c: &Config{ctx: ctx, metadata: c.c.metadata},
	}

	dict := c.c.fields.dict()
	arr := c.c.fields.array()
	fields := &fields{}

	for name, f := range dict {
		ctx := f.Context()
		v := f.cpy(context{field: ctx.field, parent: newC})
		fields.set(name, v)
	}

	if arr != nil {
		fields.a = make([]value, len(arr))
		for i, f := range arr {
			ctx := f.Context()
			v := f.cpy(context{field: ctx.field, parent: newC})
			fields.setAt(i, newC, v)
		}
	}

	newC.c.fields = fields
	return newC
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

func (c cfgSub) reify(opts *options) (interface{}, error) {
	fields := c.c.fields.dict()
	arr := c.c.fields.array()

	switch {
	case len(fields) == 0 && len(arr) == 0:
		return nil, nil
	case len(fields) > 0 && len(arr) == 0:
		m := make(map[string]interface{})
		for k, v := range fields {
			var err error
			if m[k], err = v.reify(opts); err != nil {
				return nil, err
			}
		}
		return m, nil
	case len(fields) == 0 && len(arr) > 0:
		m := make([]interface{}, len(arr))
		for i, v := range arr {
			var err error
			if m[i], err = v.reify(opts); err != nil {
				return nil, err
			}
		}
		return m, nil
	default:
		m := make(map[string]interface{})
		for k, v := range fields {
			var err error
			if m[k], err = v.reify(opts); err != nil {
				return nil, err
			}
		}
		for i, v := range arr {
			var err error
			m[fmt.Sprintf("%d", i)], err = v.reify(opts)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	}
}

func (r *cfgRef) typ(opt *options) (typeInfo, error) {
	v, err := r.resolve(opt)
	if err != nil {
		return typeInfo{}, err
	}
	return v.typ(opt)
}

func (r *cfgRef) cpy(ctx context) value {
	return newRef(ctx, r.meta(), r.ref)
}

func (r *cfgRef) Len(opt *options) (int, error) {
	v, err := r.resolve(opt)
	if err != nil {
		return 0, err
	}
	return v.Len(opt)
}

func (r *cfgRef) reflect(opts *options) (reflect.Value, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return reflect.Value{}, err
	}
	return v.reflect(opts)
}

func (r *cfgRef) reify(opts *options) (interface{}, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return reflect.Value{}, err
	}
	return v.reify(opts)
}

func (r *cfgRef) toBool(opts *options) (bool, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return false, err
	}
	return v.toBool(opts)
}

func (r *cfgRef) toString(opts *options) (string, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return "", err
	}
	return v.toString(opts)
}

func (r *cfgRef) toInt(opts *options) (int64, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return 0, err
	}
	return v.toInt(opts)
}

func (r *cfgRef) toUint(opts *options) (uint64, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return 0, err
	}
	return v.toUint(opts)
}

func (r *cfgRef) toFloat(opts *options) (float64, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return 0, err
	}
	return v.toFloat(opts)
}

func (r *cfgRef) toConfig(opts *options) (*Config, error) {
	v, err := r.resolve(opts)
	if err != nil {
		return nil, err
	}
	return v.toConfig(opts)
}

func (r *cfgRef) resolve(opts *options) (value, error) {
	return r.ref.resolve(r.ctx.getParent(), opts)
}

func (*cfgSplice) typ(*options) (typeInfo, error) {
	return typeInfo{"string", tString}, nil
}

func (s *cfgSplice) cpy(ctx context) value {
	return newSplice(ctx, s.meta(), s.splice)
}

func (s *cfgSplice) reflect(opt *options) (reflect.Value, error) {
	str, err := s.toString(opt)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(str), err
}

func (s *cfgSplice) reify(opt *options) (interface{}, error) {
	return s.toString(opt)
}

func (s *cfgSplice) toBool(opt *options) (bool, error) {
	str, err := s.toString(opt)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(str)
}

func (s *cfgSplice) toString(opts *options) (string, error) {
	return s.splice.eval(s.ctx.getParent(), opts)
}

func (s *cfgSplice) toInt(opt *options) (int64, error) {
	str, err := s.toString(opt)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(str, 0, 64)
}

func (s *cfgSplice) toUint(opt *options) (uint64, error) {
	str, err := s.toString(opt)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(str, 0, 64)
}

func (s *cfgSplice) toFloat(opt *options) (float64, error) {
	str, err := s.toString(opt)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(str, 64)
}

func isNil(v value) bool {
	if v == nil {
		return true
	}
	_, tst := v.(*cfgNil)
	return tst
}

func isSub(v value) bool {
	if v == nil {
		return false
	}
	_, tst := v.(cfgSub)
	return tst
}
