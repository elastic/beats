package ucfg

import "reflect"

func (c *Config) Unpack(to interface{}) error {
	vTo := reflect.ValueOf(to)
	if to == nil || (vTo.Kind() != reflect.Ptr && vTo.Kind() != reflect.Map) {
		return ErrPointerRequired
	}
	return reifyInto(vTo, c.fields)
}

func reifyInto(to reflect.Value, from map[string]value) error {
	to = chaseValuePointers(to)

	if to.Type() == tConfig {
		return mergeConfig(to.Addr().Interface().(*Config).fields, from)
	}

	switch to.Kind() {
	case reflect.Map:
		return reifyMap(to, from)
	case reflect.Struct:
		return reifyStruct(to, from)
	}

	return ErrTypeMismatch
}

func reifyMap(to reflect.Value, from map[string]value) error {
	if to.Type().Key().Kind() != reflect.String {
		return ErrTypeMismatch
	}

	if to.IsNil() {
		to.Set(reflect.MakeMap(to.Type()))
	}

	for k, value := range from {
		key := reflect.ValueOf(k)

		old := to.MapIndex(key)
		var v reflect.Value
		var err error

		if !old.IsValid() {
			v, err = reifyValue(to.Type().Elem(), value)
		} else {
			v, err = reifyMergeValue(old, value)
		}

		if err != nil {
			return err
		}
		to.SetMapIndex(key, v)
	}

	return nil
}

func reifyStruct(to reflect.Value, from map[string]value) error {
	to = chaseValuePointers(to)
	numField := to.NumField()

	for i := 0; i < numField; i++ {
		stField := to.Type().Field(i)
		name, _ := parseTags(stField.Tag.Get("config"))
		name = fieldName(name, stField.Name)

		value, ok := from[name]
		if !ok {
			// TODO: handle missing config
			continue
		}

		vField := to.Field(i)
		v, err := reifyMergeValue(vField, value)
		if err != nil {
			return err
		}
		vField.Set(v)
	}

	return nil
}

func reifyValue(t reflect.Type, val value) (reflect.Value, error) {
	if t.Kind() == reflect.Interface && t.NumMethod() == 0 {
		return reflect.ValueOf(val.reify()), nil
	}

	baseType := chaseTypePointers(t)
	if baseType == tConfig {
		if _, ok := val.(cfgSub); !ok {
			return reflect.Value{}, ErrTypeMismatch
		}

		v := val.reflect()
		if t == baseType { // copy config
			v = v.Elem()
		} else {
			v = pointerize(t, baseType, v)
		}
		return v, nil
	}

	if baseType.Kind() == reflect.Struct {
		if _, ok := val.(cfgSub); !ok {
			return reflect.Value{}, ErrTypeMismatch
		}

		newSt := reflect.New(baseType)
		if err := reifyInto(newSt, val.(cfgSub).c.fields); err != nil {
			return reflect.Value{}, err
		}

		if t.Kind() != reflect.Ptr {
			return newSt.Elem(), nil
		}
		return pointerize(t, baseType, newSt), nil
	}

	if baseType.Kind() == reflect.Map {
		if _, ok := val.(cfgSub); !ok {
			return reflect.Value{}, ErrTypeMismatch
		}

		if baseType.Key().Kind() != reflect.String {
			return reflect.Value{}, ErrTypeMismatch
		}

		newMap := reflect.MakeMap(baseType)
		if err := reifyInto(newMap, val.(cfgSub).c.fields); err != nil {
			return reflect.Value{}, err
		}
		return newMap, nil
	}

	if baseType.Kind() == reflect.Slice {
		arr, ok := val.(*cfgArray)
		if !ok {
			arr = &cfgArray{arr: []value{val}}
		}

		v, err := reifySlice(baseType, arr)
		if err != nil {
			return reflect.Value{}, err
		}
		return pointerize(t, baseType, v), nil
	}

	v := val.reflect()
	if v.Type().ConvertibleTo(baseType) {
		v = pointerize(t, baseType, v.Convert(baseType))
		return v, nil
	}

	return reflect.Value{}, ErrTypeMismatch
}

func reifyMergeValue(oldValue reflect.Value, val value) (reflect.Value, error) {
	old := chaseValueInterfaces(oldValue)
	t := old.Type()
	old = chaseValuePointers(old)
	if (old.Kind() == reflect.Ptr || old.Kind() == reflect.Interface) && old.IsNil() {
		return reifyValue(t, val)
	}

	baseType := chaseTypePointers(old.Type())
	if baseType == tConfig {
		sub, ok := val.(cfgSub)
		if !ok {
			return reflect.Value{}, ErrTypeMismatch
		}

		if t == baseType {
			// no pointer -> return type mismatch
			return reflect.Value{}, ErrTypeMismatch
		}

		// check if old is nil -> copy reference only
		if old.Kind() == reflect.Ptr && old.IsNil() {
			return pointerize(t, baseType, val.reflect()), nil
		}

		// check if old == value
		subOld := chaseValuePointers(old).Addr().Interface().(*Config)
		if sub.c == subOld {
			return oldValue, nil
		}

		// old != value -> merge value into old
		err := mergeConfig(subOld.fields, sub.c.fields)
		return oldValue, err
	}

	switch baseType.Kind() {
	case reflect.Map:
		sub, ok := val.(cfgSub)
		if !ok {
			return reflect.Value{}, ErrTypeMismatch
		}
		err := reifyMap(old, sub.c.fields)
		return old, err
	case reflect.Struct:
		sub, ok := val.(cfgSub)
		if !ok {
			return reflect.Value{}, ErrTypeMismatch
		}
		err := reifyStruct(old, sub.c.fields)
		return oldValue, err
	case reflect.Array:
		arr, ok := val.(*cfgArray)
		if !ok {
			// convert single value to array for merging
			arr = &cfgArray{
				arr: []value{val},
			}
		}
		return reifyArray(old, baseType, arr)
	case reflect.Slice:
		arr, ok := val.(*cfgArray)
		if !ok {
			// convert single value to array for merging
			arr = &cfgArray{
				arr: []value{val},
			}
		}
		return reifySlice(baseType, arr)
	}

	// try primitive conversion
	v := val.reflect()
	if v.Type().ConvertibleTo(baseType) {
		return pointerize(t, baseType, v.Convert(baseType)), nil
	}

	return reflect.Value{}, ErrTODO
}

func reifyArray(to reflect.Value, tTo reflect.Type, arr *cfgArray) (reflect.Value, error) {
	if arr.Len() != tTo.Len() {
		return reflect.Value{}, ErrArraySizeMistach
	}
	return reifyDoArray(to, tTo.Elem(), arr)
}

func reifySlice(tTo reflect.Type, arr *cfgArray) (reflect.Value, error) {
	to := reflect.MakeSlice(tTo, arr.Len(), arr.Len())
	return reifyDoArray(to, tTo.Elem(), arr)
}

func reifyDoArray(to reflect.Value, elemT reflect.Type, arr *cfgArray) (reflect.Value, error) {
	for i, from := range arr.arr {
		v, err := reifyValue(elemT, from)
		if err != nil {
			return reflect.Value{}, ErrTODO
		}
		to.Index(i).Set(v)
	}
	return to, nil
}

func pointerize(t, base reflect.Type, v reflect.Value) reflect.Value {
	if t == base {
		return v
	}

	if t.Kind() == reflect.Interface {
		return v
	}

	for t != v.Type() {
		if !v.CanAddr() {
			tmp := reflect.New(v.Type())
			tmp.Elem().Set(v)
			v = tmp
		} else {
			v = v.Addr()
		}
	}
	return v
}
