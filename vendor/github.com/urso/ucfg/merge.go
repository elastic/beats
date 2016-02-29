package ucfg

import (
	"reflect"
	"strings"
)

func (c *Config) Merge(from interface{}, options ...Option) error {
	opts := makeOptions(options)
	other, err := normalize(opts, from)
	if err != nil {
		return err
	}
	return mergeConfig(c.fields, other.fields)
}

func mergeConfig(to, from map[string]value) error {
	for k, v := range from {
		old, ok := to[k]
		if !ok {
			to[k] = v
			continue
		}

		subOld, err := old.toConfig()
		if err != nil {
			to[k] = v
			continue
		}

		subFrom, err := v.toConfig()
		if err != nil {
			to[k] = v
			continue
		}

		err = mergeConfig(subOld.fields, subFrom.fields)
		if err != nil {
			return err
		}
	}
	return nil
}

// convert from into normalized *Config checking for errors
// before merging generated(normalized) config with current config
func normalize(opts options, from interface{}) (*Config, error) {
	vFrom := chaseValue(reflect.ValueOf(from))

	switch vFrom.Type() {
	case tConfig:
		return vFrom.Addr().Interface().(*Config), nil
	case tConfigMap:
		return normalizeMap(opts, vFrom)
	default:
		switch vFrom.Kind() {
		case reflect.Struct:
			return normalizeStruct(opts, vFrom)
		case reflect.Map:
			return normalizeMap(opts, vFrom)
		}
	}

	return nil, raise(ErrTypeMismatch)
}

func normalizeCfgPath(cfg *Config, opts options, field string) (*Config, string, error) {
	if opts.pathSep == "" {
		return cfg, field, nil
	}

	path := strings.Split(field, opts.pathSep)
	for len(path) > 1 {
		field = path[0]
		path = path[1:]

		sub, exists := cfg.fields[field]
		if exists {
			vSub, ok := sub.(cfgSub)
			if !ok {
				return nil, field, raise(ErrExpectedObject)
			}

			cfg = vSub.c
			continue
		}

		next := New()
		cfg.fields[field] = cfgSub{next}
		cfg = next
	}
	field = path[0]

	return cfg, field, nil
}

func normalizeMap(opts options, from reflect.Value) (*Config, error) {
	cfg := New()
	if err := normalizeMapInto(cfg, opts, from); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeMapInto(cfg *Config, opts options, from reflect.Value) error {
	k := from.Type().Key().Kind()
	if k != reflect.String && k != reflect.Interface {
		return raise(ErrTypeMismatch)
	}

	for _, k := range from.MapKeys() {
		k = chaseValueInterfaces(k)
		if k.Kind() != reflect.String {
			return raise(ErrKeyTypeNotString)
		}

		err := normalizeSetField(cfg, opts, k.String(), from.MapIndex(k))
		if err != nil {
			return err
		}
	}
	return nil
}

func normalizeStruct(opts options, from reflect.Value) (*Config, error) {
	cfg := New()
	if err := normalizeStructInto(cfg, opts, from); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeStructInto(cfg *Config, opts options, from reflect.Value) error {
	v := chaseValue(from)
	numField := v.NumField()

	for i := 0; i < numField; i++ {
		var err error
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
				return raise(ErrTypeMismatch)
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

func normalizeSetField(cfg *Config, opts options, name string, v reflect.Value) error {
	to, name, err := normalizeCfgPath(cfg, opts, name)
	if err != nil {
		return err
	}
	if to.HasField(name) {
		return errDuplicateKey(name)
	}

	val, err := normalizeValue(opts, v)
	if err != nil {
		return err
	}

	to.fields[name] = val
	return nil
}

func normalizeStructValue(opts options, from reflect.Value) (value, error) {
	sub, err := normalizeStruct(opts, from)
	if err != nil {
		return nil, err
	}
	return cfgSub{sub}, nil
}

func normalizeMapValue(opts options, from reflect.Value) (value, error) {
	sub, err := normalizeMap(opts, from)
	if err != nil {
		return nil, err
	}
	return cfgSub{sub}, nil
}

func normalizeArray(opts options, v reflect.Value) (value, error) {
	l := v.Len()
	out := make([]value, 0, l)
	for i := 0; i < l; i++ {
		tmp, err := normalizeValue(opts, v.Index(i))
		if err != nil {
			return nil, err
		}
		out = append(out, tmp)
	}
	return &cfgArray{arr: out}, nil
}

func normalizeValue(opts options, v reflect.Value) (value, error) {
	v = chaseValue(v)

	// handle primitives
	switch v.Kind() {
	case reflect.Bool:
		return &cfgBool{b: v.Bool()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &cfgInt{i: v.Int()}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &cfgInt{i: int64(v.Uint())}, nil
	case reflect.Float32, reflect.Float64:
		return &cfgFloat{f: v.Float()}, nil
	case reflect.String:
		return &cfgString{s: v.String()}, nil
	case reflect.Array, reflect.Slice:
		return normalizeArray(opts, v)
	case reflect.Map:
		return normalizeMapValue(opts, v)
	case reflect.Struct:
		if v.Type().ConvertibleTo(tConfig) {
			var c *Config
			if !v.CanAddr() {
				vTmp := reflect.New(tConfig)
				vTmp.Elem().Set(v)
				c = vTmp.Interface().(*Config)
			} else {
				c = v.Addr().Interface().(*Config)
			}
			return cfgSub{c}, nil
		}
		return normalizeStructValue(opts, v)
	default:
		if v.IsNil() {
			return cfgNil{}, nil
		}
		return nil, raise(ErrTypeMismatch)
	}
}
