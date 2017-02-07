package ucfg

import (
	"fmt"
	"reflect"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"
)

// Merge a map, a slice, a struct or another Config object into c.
//
// Merge traverses the value from recursively copying all values into a hierarchy
// of Config objects plus primitives into c.
//
// Merge supports the options: PathSep, MetaData, StructTag, VarExp
//
// Merge uses the type-dependent default encodings:
//  - Boolean values are encoded as booleans.
//  - Integer are encoded as int64 values, unsigned integer values as uint64 and
//    floats as float64 values.
//  - Strings are copied into string values.
//    If the VarExp is set, string fields will be parsed into
//    variable expansion expressions. The expression can reference any
//    other setting by absolute name.
//  - Array and slices are copied into new Config objects with index accessors only.
//  - Struct values and maps with key type string are encoded as Config objects with
//    named field accessors.
//  - Config objects will be copied and added to the current hierarchy.
//
// The `config` struct tag (configurable via StructTag option) can be used to
// set the field name and enable additional merging settings per field:
//
//  // field appears in Config as key "myName"
//  Field int `config:"myName"`
//
//  // field appears in sub-Config "mySub" as key "myName" (requires PathSep("."))
//  Field int `config:"mySub.myName"`
//
//  // field is processed as if keys are part of outer struct (type can be a
//  // struct, a slice, an array, a map or of type *Config)
//  Field map[string]interface{} `config:",inline"`
//
//  // field is ignored by Merge
//  Field string `config:",ignore"`
//
//
// Returns an error if merging fails to normalize and validate the from value.
// If duplicate setting names are detected in the input, merging fails as well.
//
// Config cannot represent cyclic structures and Merge does not handle them
// well. Passing cyclic structures to Merge will result in an infinite recursive
// loop.
func (c *Config) Merge(from interface{}, options ...Option) error {
	// from is empty in case of empty config file
	if from == nil {
		return nil
	}

	opts := makeOptions(options)
	other, err := normalize(opts, from)

	if err != nil {
		return err
	}
	return mergeConfig(opts, c, other)
}

func mergeConfig(opts *options, to, from *Config) Error {
	if err := mergeConfigDict(opts, to, from); err != nil {
		return err
	}
	return mergeConfigArr(opts, to, from)
}

func mergeConfigDict(opts *options, to, from *Config) Error {
	for k, v := range from.fields.dict() {
		ctx := context{
			parent: cfgSub{to},
			field:  k,
		}

		old, _ := to.fields.get(k)
		merged, err := mergeValues(opts, old, v)
		if err != nil {
			return err
		}

		to.fields.set(k, merged.cpy(ctx))
	}
	return nil
}

func mergeConfigArr(opts *options, to, from *Config) Error {
	l := len(to.fields.array())
	if l > len(from.fields.array()) {
		l = len(from.fields.array())
	}

	// merge array indexes available in to and from
	for i := 0; i < l; i++ {
		ctx := context{
			parent: cfgSub{to},
			field:  fmt.Sprintf("%v", i),
		}

		old := to.fields.array()[i]
		v := from.fields.array()[i]
		merged, err := mergeValues(opts, old, v)
		if err != nil {
			return err
		}
		to.fields.setAt(i, cfgSub{to}, merged.cpy(ctx))
	}

	end := len(from.fields.array())
	if end <= l {
		return nil
	}

	// add additional array entries not yet in 'to'
	for ; l < end; l++ {
		ctx := context{
			parent: cfgSub{to},
			field:  fmt.Sprintf("%v", l),
		}
		v := from.fields.array()[l]
		to.fields.setAt(l, cfgSub{to}, v.cpy(ctx))
	}

	return nil
}

func mergeValues(opts *options, old, v value) (value, Error) {
	if old == nil {
		return v, nil
	}

	// check if new and old value evaluate to sub-configurations. If one is no
	// sub-configuration, use new value only.
	subOld, err := old.toConfig(opts)
	if err != nil {
		return v, nil
	}
	subV, err := v.toConfig(opts)
	if err != nil {
		return v, nil
	}

	// merge new and old evaluated sub-configurations and return subOld for
	// reassigning to old key in case of subOld being generated dynamically
	if err := mergeConfig(opts, subOld, subV); err != nil {
		return nil, err
	}
	return cfgSub{subOld}, nil
}

// convert from into normalized *Config checking for errors
// before merging generated(normalized) config with current config
func normalize(opts *options, from interface{}) (*Config, Error) {
	vFrom := chaseValue(reflect.ValueOf(from))

	switch vFrom.Type() {
	case tConfig:
		return vFrom.Addr().Interface().(*Config), nil
	case tConfigMap:
		return normalizeMap(opts, vFrom)
	default:
		// try to convert vFrom into Config (rebranding)
		if v, ok := tryTConfig(vFrom); ok {
			return v.Addr().Interface().(*Config), nil
		}

		// normalize given map/struct value
		switch vFrom.Kind() {
		case reflect.Struct:
			return normalizeStruct(opts, vFrom)
		case reflect.Map:
			return normalizeMap(opts, vFrom)
		case reflect.Array, reflect.Slice:
			tmp, err := normalizeArray(opts, tagOptions{}, context{}, vFrom)
			if err != nil {
				return nil, err
			}
			c, _ := tmp.toConfig(opts)
			return c, nil
		}

	}

	return nil, raiseInvalidTopLevelType(from, opts.meta)
}

func normalizeMap(opts *options, from reflect.Value) (*Config, Error) {
	cfg := New()
	cfg.metadata = opts.meta
	if err := normalizeMapInto(cfg, opts, from); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeMapInto(cfg *Config, opts *options, from reflect.Value) Error {
	k := from.Type().Key().Kind()
	if k != reflect.String && k != reflect.Interface {
		return raiseKeyInvalidTypeMerge(cfg, from.Type())
	}

	for _, k := range from.MapKeys() {
		k = chaseValueInterfaces(k)
		if k.Kind() != reflect.String {
			return raiseKeyInvalidTypeMerge(cfg, from.Type())
		}

		err := normalizeSetField(cfg, opts, noTagOpts, k.String(), from.MapIndex(k))
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeStruct(opts *options, from reflect.Value) (*Config, Error) {
	cfg := New()
	cfg.metadata = opts.meta
	if err := normalizeStructInto(cfg, opts, from); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeStructInto(cfg *Config, opts *options, from reflect.Value) Error {
	v := chaseValue(from)
	numField := v.NumField()

	for i := 0; i < numField; i++ {
		var err Error
		stField := v.Type().Field(i)

		// ignore non exported fields
		if rune, _ := utf8.DecodeRuneInString(stField.Name); !unicode.IsUpper(rune) {
			continue
		}

		name, tagOpts := parseTags(stField.Tag.Get(opts.tag))
		if tagOpts.ignore {
			continue
		}

		if tagOpts.squash {
			vField := chaseValue(v.Field(i))
			switch vField.Kind() {
			case reflect.Struct:
				err = normalizeStructInto(cfg, opts, vField)
			case reflect.Map:
				err = normalizeMapInto(cfg, opts, vField)
			default:
				return raiseSquashNeedsObject(cfg, opts, stField.Name, vField.Type())
			}
		} else {
			name = fieldName(name, stField.Name)
			err = normalizeSetField(cfg, opts, tagOpts, name, v.Field(i))
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeSetField(
	cfg *Config,
	opts *options,
	tagOpts tagOptions,
	name string,
	v reflect.Value,
) Error {
	val, err := normalizeValue(opts, tagOpts, context{}, v)
	if err != nil {
		return err
	}

	p := parsePath(name, opts.pathSep)
	old, err := p.GetValue(cfg, opts)
	if err != nil {
		if err.Reason() != ErrMissing {
			return err
		}
		old = nil
	}

	switch {
	case !isNil(old) && isNil(val):
		return nil
	case isNil(old):
		return p.SetValue(cfg, opts, val)
	case isSub(old) && isSub(val):
		cfgOld, _ := old.toConfig(opts)
		cfgVal, _ := val.toConfig(opts)
		return mergeConfig(opts, cfgOld, cfgVal)
	default:
		return raiseDuplicateKey(cfg, name)
	}
}

func normalizeStructValue(opts *options, ctx context, from reflect.Value) (value, Error) {
	sub, err := normalizeStruct(opts, from)
	if err != nil {
		return nil, err
	}
	v := cfgSub{sub}
	v.SetContext(ctx)
	return v, nil
}

func normalizeMapValue(opts *options, ctx context, from reflect.Value) (value, Error) {
	sub, err := normalizeMap(opts, from)
	if err != nil {
		return nil, err
	}
	v := cfgSub{sub}
	v.SetContext(ctx)
	return v, nil
}

func normalizeArray(
	opts *options,
	tagOpts tagOptions,
	ctx context,
	v reflect.Value,
) (value, Error) {
	l := v.Len()
	out := make([]value, 0, l)

	cfg := New()
	cfg.metadata = opts.meta
	cfg.ctx = ctx
	val := cfgSub{cfg}

	for i := 0; i < l; i++ {
		idx := fmt.Sprintf("%v", i)
		ctx := context{
			parent: val,
			field:  idx,
		}
		tmp, err := normalizeValue(opts, tagOpts, ctx, v.Index(i))
		if err != nil {
			return nil, err
		}
		out = append(out, tmp)
	}

	cfg.fields.a = out
	return val, nil
}

func normalizeValue(
	opts *options,
	tagOpts tagOptions,
	ctx context,
	v reflect.Value,
) (value, Error) {
	v = chaseValue(v)

	switch v.Type() {
	case tDuration:
		d := v.Interface().(time.Duration)
		return newString(ctx, opts.meta, d.String()), nil
	case tRegexp:
		r := v.Addr().Interface().(*regexp.Regexp)
		return newString(ctx, opts.meta, r.String()), nil
	}

	// handle primitives
	switch v.Kind() {
	case reflect.Bool:
		return newBool(ctx, opts.meta, v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Int()
		if i > 0 {
			return newUint(ctx, opts.meta, uint64(i)), nil
		}
		return newInt(ctx, opts.meta, i), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return newUint(ctx, opts.meta, v.Uint()), nil
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		return newFloat(ctx, opts.meta, f), nil
	case reflect.String:
		return normalizeString(ctx, opts, v.String())
	case reflect.Array, reflect.Slice:
		return normalizeArray(opts, tagOpts, ctx, v)
	case reflect.Map:
		return normalizeMapValue(opts, ctx, v)
	case reflect.Struct:
		if v, ok := tryTConfig(v); ok {
			c := v.Addr().Interface().(*Config)
			ret := cfgSub{c}
			if ret.Context().parent != ctx.parent {
				ret.SetContext(ctx)
			}
			return ret, nil
		}

		return normalizeStructValue(opts, ctx, v)
	default:
		if v.IsNil() {
			return &cfgNil{cfgPrimitive{ctx, opts.meta}}, nil
		}
		return nil, raiseUnsupportedInputType(ctx, opts.meta, v)
	}
}

func normalizeString(ctx context, opts *options, str string) (value, Error) {
	if !opts.varexp {
		return newString(ctx, opts.meta, str), nil
	}

	varexp, err := parseSplice(str, opts.pathSep)
	if err != nil {
		return nil, raiseParseSplice(ctx, opts.meta, err)
	}

	switch p := varexp.(type) {
	case constExp:
		return newString(ctx, opts.meta, str), nil
	case *reference:
		return newRef(ctx, opts.meta, p), nil
	}

	return newSplice(ctx, opts.meta, varexp), nil
}
