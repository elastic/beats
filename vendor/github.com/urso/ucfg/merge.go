package ucfg

import (
	"fmt"
	"reflect"
	"strings"
)

func (c *Config) Merge(from interface{}, options ...Option) error {
	opts := makeOptions(options)
	other, err := normalize(opts, from)
	if err != nil {
		return err
	}
	return mergeConfig(c, other)
}

func mergeConfig(to, from *Config) Error {
	for k, v := range from.fields.fields {
		ctx := context{
			parent: cfgSub{to},
			field:  k,
		}

		old, ok := to.fields.fields[k]
		if !ok {
			to.fields.fields[k] = v.cpy(ctx)
			continue
		}

		subOld, err := old.toConfig()
		if err != nil {
			to.fields.fields[k] = v.cpy(ctx)
			continue
		}

		subFrom, err := v.toConfig()
		if err != nil {
			to.fields.fields[k] = v.cpy(ctx)
			continue
		}

		if err := mergeConfig(subOld, subFrom); err != nil {
			return err
		}
	}
	return nil
}

// convert from into normalized *Config checking for errors
// before merging generated(normalized) config with current config
func normalize(opts options, from interface{}) (*Config, Error) {
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
		}

	}

	return nil, raiseInvalidTopLevelType(from)
}

func normalizeCfgPath(cfg *Config, opts options, field string) (*Config, string, Error) {
	if opts.pathSep == "" {
		return cfg, field, nil
	}

	path := strings.Split(field, opts.pathSep)
	for len(path) > 1 {
		field = path[0]
		path = path[1:]

		sub, exists := cfg.fields.fields[field]
		if exists {
			vSub, ok := sub.(cfgSub)
			if !ok {
				return nil, field, raiseExpectedObject(sub)
			}

			cfg = vSub.c
			continue
		}

		next := New()
		next.metadata = opts.meta
		v := cfgSub{next}
		v.SetContext(context{
			parent: cfgSub{cfg},
			field:  field,
		})
		cfg.fields.fields[field] = v
		cfg = next
	}
	field = path[0]

	return cfg, field, nil
}

func normalizeMap(opts options, from reflect.Value) (*Config, Error) {
	cfg := New()
	cfg.metadata = opts.meta
	if err := normalizeMapInto(cfg, opts, from); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeMapInto(cfg *Config, opts options, from reflect.Value) Error {
	k := from.Type().Key().Kind()
	if k != reflect.String && k != reflect.Interface {
		return raiseKeyInvalidTypeMerge(cfg, from.Type())
	}

	for _, k := range from.MapKeys() {
		k = chaseValueInterfaces(k)
		if k.Kind() != reflect.String {
			return raiseKeyInvalidTypeMerge(cfg, from.Type())
		}

		err := normalizeSetField(cfg, opts, k.String(), from.MapIndex(k))
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeStruct(opts options, from reflect.Value) (*Config, Error) {
	cfg := New()
	cfg.metadata = opts.meta
	if err := normalizeStructInto(cfg, opts, from); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeStructInto(cfg *Config, opts options, from reflect.Value) Error {
	v := chaseValue(from)
	numField := v.NumField()

	for i := 0; i < numField; i++ {
		var err Error
		stField := v.Type().Field(i)
		name, tagOpts := parseTags(stField.Tag.Get(opts.tag))

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
			err = normalizeSetField(cfg, opts, name, v.Field(i))
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeSetField(cfg *Config, opts options, name string, v reflect.Value) Error {
	to, name, err := normalizeCfgPath(cfg, opts, name)
	if err != nil {
		return err
	}
	if to.HasField(name) {
		return raiseDuplicateKey(to, name)
	}

	ctx := context{
		parent: cfgSub{cfg},
		field:  name,
	}
	val, err := normalizeValue(opts, ctx, v)
	if err != nil {
		return err
	}

	to.fields.fields[name] = val
	return nil
}

func normalizeStructValue(opts options, ctx context, from reflect.Value) (value, Error) {
	sub, err := normalizeStruct(opts, from)
	if err != nil {
		return nil, err
	}
	v := cfgSub{sub}
	v.SetContext(ctx)
	return v, nil
}

func normalizeMapValue(opts options, ctx context, from reflect.Value) (value, Error) {
	sub, err := normalizeMap(opts, from)
	if err != nil {
		return nil, err
	}
	v := cfgSub{sub}
	v.SetContext(ctx)
	return v, nil
}

func normalizeArray(opts options, ctx context, v reflect.Value) (value, Error) {
	l := v.Len()
	out := make([]value, 0, l)

	arr := &cfgArray{cfgPrimitive{ctx, opts.meta}, nil}

	for i := 0; i < l; i++ {
		ctx := context{
			parent: arr,
			field:  fmt.Sprintf("%v", i),
		}
		tmp, err := normalizeValue(opts, ctx, v.Index(i))
		if err != nil {
			return nil, err
		}
		out = append(out, tmp)
	}

	arr.arr = out
	return arr, nil
}

func normalizeValue(opts options, ctx context, v reflect.Value) (value, Error) {
	v = chaseValue(v)

	// handle primitives
	switch v.Kind() {
	case reflect.Bool:
		return newBool(ctx, opts.meta, v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return newInt(ctx, opts.meta, v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return newInt(ctx, opts.meta, int64(v.Uint())), nil
	case reflect.Float32, reflect.Float64:
		return newFloat(ctx, opts.meta, v.Float()), nil
	case reflect.String:
		return newString(ctx, opts.meta, v.String()), nil
	case reflect.Array, reflect.Slice:
		return normalizeArray(opts, ctx, v)
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
		return nil, raiseUnsupportedInputType(ctx, opts, v)
	}
}
