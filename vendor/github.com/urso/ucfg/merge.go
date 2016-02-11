package ucfg

import "reflect"

func (c *Config) Merge(from interface{}) error {
	return mergeInto(c.fields, reflect.ValueOf(from))
}

func mergeInto(to map[string]value, from reflect.Value) error {
	vFrom := chaseValue(from)

	switch vFrom.Type() {
	case tConfig:
		return mergeConfig(to, vFrom.Convert(tConfig).Interface().(Config).fields)
	case tConfigMap:
		return mergeMap(to, vFrom)
	default:
		switch vFrom.Kind() {
		case reflect.Struct:
			return mergeStruct(to, vFrom)
		case reflect.Map:
			return mergeMap(to, vFrom)
		}

		return ErrTypeMismatch
	}
}

func mergeConfig(to, from map[string]value) error {
	for k, v := range from {
		old, ok := to[k]
		if !ok {
			to[k] = v
			continue
		}

		subOld, ok := old.(cfgSub)
		if !ok {
			to[k] = v
			continue
		}

		subFrom, ok := v.(cfgSub)
		if !ok {
			to[k] = v
			continue
		}

		err := mergeConfig(subOld.c.fields, subFrom.c.fields)
		if err != nil {
			return err
		}
	}
	return nil
}

func mergeMap(to map[string]value, from reflect.Value) error {
	k := from.Type().Key().Kind()
	if k != reflect.String && k != reflect.Interface {
		return ErrTypeMismatch
	}

	for _, k := range from.MapKeys() {
		k = chaseValueInterfaces(k)
		if k.Kind() != reflect.String {
			return ErrTypeMismatch
		}

		key := k.String()
		v := from.MapIndex(k)

		var new value
		var err error

		if old, ok := to[key]; ok {
			new, err = mergeValue(old, v)
		} else {
			new, err = normalizeValue(v)
		}

		if err != nil {
			return err
		}

		to[key] = new
	}
	return nil
}

func mergeStruct(to map[string]value, from reflect.Value) error {
	v := chaseValue(from)
	numField := v.NumField()

	for i := 0; i < numField; i++ {
		stField := v.Type().Field(i)
		name, opts := parseTags(stField.Tag.Get("config"))

		if opts.squash {
			var err error

			vField := chaseValue(v.Field(i))
			switch vField.Kind() {
			case reflect.Struct:
				err = mergeStruct(to, vField)
			case reflect.Map:
				err = mergeMap(to, vField)
			default:
				err = ErrTypeMismatch
			}

			if err != nil {
				return err
			}
		} else {
			name = fieldName(name, stField.Name)
			var new value
			var err error

			if old, ok := to[name]; ok {
				new, err = mergeValue(old, v.Field(i))
			} else {
				new, err = normalizeValue(v.Field(i))
			}

			if err != nil {
				return err
			}

			to[name] = new
		}
	}

	return nil
}

func mergeValue(old value, new reflect.Value) (value, error) {
	if sub, ok := old.(cfgSub); ok {
		v := chaseValue(new)
		t := v.Type()
		k := t.Kind()
		isSub := t == tConfig || t == tConfigMap ||
			k == reflect.Struct || k == reflect.Map
		if isSub {
			err := mergeInto(sub.c.fields, new)
			return sub, err
		}
	}
	return normalizeValue(new)
}

func normalizeValue(v reflect.Value) (value, error) {
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
		return normalizeArray(v)
	case reflect.Map:
		return normalizeMap(v)
	case reflect.Struct:
		tmp := make(map[string]value)

		if v.Type().ConvertibleTo(tConfig) {
			var c *Config
			if !v.CanAddr() {
				vTmp := reflect.New(tConfig)
				vTmp.Elem().Set(v)
				c = vTmp.Interface().(*Config)
			} else {
				c = v.Addr().Interface().(*Config)
			}
			mergeConfig(tmp, c.fields)
		} else if err := mergeStruct(tmp, v); err != nil {
			return nil, err
		}

		return cfgSub{c: &Config{tmp}}, nil
	default:
		return nil, ErrTypeMismatch
	}
}

func normalizeArray(v reflect.Value) (value, error) {
	l := v.Len()
	out := make([]value, 0, l)
	for i := 0; i < l; i++ {
		tmp, err := normalizeValue(v.Index(i))
		if err != nil {
			return nil, err
		}
		out = append(out, tmp)
	}
	return &cfgArray{arr: out}, nil
}

func normalizeMap(m reflect.Value) (value, error) {
	to := make(map[string]value)
	if err := mergeMap(to, m); err != nil {
		return nil, err
	}
	return cfgSub{&Config{to}}, nil
}
