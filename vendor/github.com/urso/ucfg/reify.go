package ucfg

import (
	"reflect"
	"strings"
	"time"
)

func (c *Config) Unpack(to interface{}, options ...Option) error {
	if c == nil {
		return raise(ErrNilConfig)
	}
	if to == nil {
		return raise(ErrNilValue)
	}

	opts := makeOptions(options)
	vTo := reflect.ValueOf(to)
	if to == nil || (vTo.Kind() != reflect.Ptr && vTo.Kind() != reflect.Map) {
		return raise(ErrPointerRequired)
	}
	return reifyInto(opts, vTo, c)
}

func reifyInto(opts options, to reflect.Value, from *Config) error {
	to = chaseValuePointers(to)

	if to.Type() == tConfig {
		return mergeConfig(to.Addr().Interface().(*Config).fields, from.fields)
	}

	tTo := chaseTypePointers(to.Type())
	switch tTo.Kind() {
	case reflect.Map:
		return reifyMap(opts, to, from)
	case reflect.Struct:
		return reifyStruct(opts, to, from)
	}

	return raise(ErrTypeMismatch)
}

func reifyMap(opts options, to reflect.Value, from *Config) error {
	if to.Type().Key().Kind() != reflect.String {
		return raise(ErrTypeMismatch)
	}

	if len(from.fields) == 0 {
		return nil
	}

	if to.IsNil() {
		to.Set(reflect.MakeMap(to.Type()))
	}
	for k, value := range from.fields {
		key := reflect.ValueOf(k)

		old := to.MapIndex(key)
		var v reflect.Value
		var err error

		if !old.IsValid() {
			v, err = reifyValue(opts, to.Type().Elem(), value)
		} else {
			v, err = reifyMergeValue(opts, old, value)
		}

		if err != nil {
			return err
		}
		to.SetMapIndex(key, v)
	}

	return nil
}

func reifyStruct(opts options, orig reflect.Value, cfg *Config) error {
	orig = chaseValuePointers(orig)

	to := chaseValuePointers(reflect.New(chaseTypePointers(orig.Type())))
	if orig.Kind() == reflect.Struct { // if orig is has been allocated copy into to
		to.Set(orig)
	}

	numField := to.NumField()
	for i := 0; i < numField; i++ {
		var err error
		stField := to.Type().Field(i)
		vField := to.Field(i)
		name, tagOpts := parseTags(stField.Tag.Get(opts.tag))

		if tagOpts.squash {
			vField := chaseValue(vField)
			switch vField.Kind() {
			case reflect.Struct, reflect.Map:
				err = reifyInto(opts, vField, cfg)
			default:
				return raise(ErrTypeMismatch)
			}
		} else {
			name = fieldName(name, stField.Name)
			err = reifyGetField(cfg, opts, name, vField)
		}

		if err != nil {
			return err
		}
	}

	orig.Set(pointerize(orig.Type(), to.Type(), to))
	return nil
}

func reifyGetField(cfg *Config, opts options, name string, to reflect.Value) error {
	from, field, err := reifyCfgPath(cfg, opts, name)
	if err != nil {
		return err
	}

	value, ok := from.fields[field]
	if !ok {
		// TODO: handle missing config
		return nil
	}

	v, err := reifyMergeValue(opts, to, value)
	if err != nil {
		return err
	}

	to.Set(v)
	return nil
}

func reifyCfgPath(cfg *Config, opts options, field string) (*Config, string, error) {
	if opts.pathSep == "" {
		return cfg, field, nil
	}

	path := strings.Split(field, opts.pathSep)
	for len(path) > 1 {
		field = path[0]
		path = path[1:]

		sub, exists := cfg.fields[field]
		if !exists {
			return nil, field, raise(ErrMissing)
		}

		cSub, err := sub.toConfig()
		if err != nil {
			return nil, field, raise(ErrExpectedObject)
		}
		cfg = cSub
	}
	field = path[0]

	return cfg, field, nil
}

func reifyValue(opts options, t reflect.Type, val value) (reflect.Value, error) {
	if t.Kind() == reflect.Interface && t.NumMethod() == 0 {
		return reflect.ValueOf(val.reify()), nil
	}

	baseType := chaseTypePointers(t)
	if baseType == tConfig {
		if _, err := val.toConfig(); err != nil {
			return reflect.Value{}, raise(ErrTypeMismatch)
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
		sub, err := val.toConfig()
		if err != nil {
			return reflect.Value{}, raise(ErrTypeMismatch)
		}

		newSt := reflect.New(baseType)
		if err := reifyInto(opts, newSt, sub); err != nil {
			return reflect.Value{}, err
		}

		if t.Kind() != reflect.Ptr {
			return newSt.Elem(), nil
		}
		return pointerize(t, baseType, newSt), nil
	}

	switch baseType.Kind() {
	case reflect.Map:
		sub, err := val.toConfig()
		if err != nil {
			return reflect.Value{}, raise(ErrTypeMismatch)
		}

		if baseType.Key().Kind() != reflect.String {
			return reflect.Value{}, raise(ErrTypeMismatch)
		}

		newMap := reflect.MakeMap(baseType)
		if err := reifyInto(opts, newMap, sub); err != nil {
			return reflect.Value{}, err
		}
		return newMap, nil

	case reflect.Slice:
		v, err := reifySlice(opts, baseType, castArr(val))
		if err != nil {
			return reflect.Value{}, err
		}
		return pointerize(t, baseType, v), nil
	}

	return reifyPrimitive(opts, val, t, baseType)
}

func reifyMergeValue(
	opts options,
	oldValue reflect.Value, val value,
) (reflect.Value, error) {
	old := chaseValueInterfaces(oldValue)
	t := old.Type()
	old = chaseValuePointers(old)
	if (old.Kind() == reflect.Ptr || old.Kind() == reflect.Interface) && old.IsNil() {
		return reifyValue(opts, t, val)
	}

	baseType := chaseTypePointers(old.Type())
	if baseType == tConfig {
		sub, err := val.toConfig()
		if err != nil {
			return reflect.Value{}, raise(ErrTypeMismatch)
		}

		if t == baseType {
			// no pointer -> return type mismatch
			return reflect.Value{}, raise(ErrTypeMismatch)
		}

		// check if old is nil -> copy reference only
		if old.Kind() == reflect.Ptr && old.IsNil() {
			return pointerize(t, baseType, val.reflect()), nil
		}

		// check if old == value
		subOld := chaseValuePointers(old).Addr().Interface().(*Config)
		if sub == subOld {
			return oldValue, nil
		}

		// old != value -> merge value into old
		err = mergeConfig(subOld.fields, sub.fields)
		return oldValue, err
	}

	switch baseType.Kind() {
	case reflect.Map:
		sub, err := val.toConfig()
		if err != nil {
			return reflect.Value{}, raise(ErrTypeMismatch)
		}
		err = reifyMap(opts, old, sub)
		return old, err

	case reflect.Struct:
		sub, err := val.toConfig()
		if err != nil {
			return reflect.Value{}, raise(ErrTypeMismatch)
		}
		err = reifyStruct(opts, old, sub)
		return oldValue, err

	case reflect.Array:
		return reifyArray(opts, old, baseType, castArr(val))

	case reflect.Slice:
		return reifySlice(opts, baseType, castArr(val))
	}

	return reifyPrimitive(opts, val, t, baseType)
}

func reifyPrimitive(
	opts options,
	val value,
	t, baseType reflect.Type,
) (reflect.Value, error) {
	// zero initialize value if val==nil
	if _, ok := val.(cfgNil); ok {
		return pointerize(t, baseType, reflect.Zero(baseType)), nil
	}

	// try primitive conversion
	switch {
	case val.typ() == baseType:
		return pointerize(t, baseType, val.reflect()), nil

	case baseType.Kind() == reflect.String:
		s, err := val.toString()
		if err != nil {
			return reflect.Value{}, err
		}
		return pointerize(t, baseType, reflect.ValueOf(s)), nil

	case baseType == tDuration:
		s, err := val.toString()
		if err != nil {
			return reflect.Value{}, err
		}

		d, err := time.ParseDuration(s)
		if err != nil {
			return reflect.Value{}, err
		}

		return pointerize(t, baseType, reflect.ValueOf(d)), nil

	case val.typ().ConvertibleTo(baseType):
		return pointerize(t, baseType, val.reflect().Convert(baseType)), nil

	}

	return reflect.Value{}, raise(ErrTODO)
}

func reifyArray(opts options, to reflect.Value, tTo reflect.Type, arr *cfgArray) (reflect.Value, error) {
	if arr.Len() != tTo.Len() {
		return reflect.Value{}, raise(ErrArraySizeMistach)
	}
	return reifyDoArray(opts, to, tTo.Elem(), arr)
}

func reifySlice(opts options, tTo reflect.Type, arr *cfgArray) (reflect.Value, error) {
	to := reflect.MakeSlice(tTo, arr.Len(), arr.Len())
	return reifyDoArray(opts, to, tTo.Elem(), arr)
}

func reifyDoArray(
	opts options,
	to reflect.Value, elemT reflect.Type, arr *cfgArray,
) (reflect.Value, error) {
	for i, from := range arr.arr {
		v, err := reifyValue(opts, elemT, from)
		if err != nil {
			return reflect.Value{}, raise(ErrTODO)
		}
		to.Index(i).Set(v)
	}
	return to, nil
}

func castArr(v value) *cfgArray {
	if arr, ok := v.(*cfgArray); ok {
		return arr
	}

	if v.Len() == 0 {
		return &cfgArray{}
	}

	return &cfgArray{
		arr: []value{v},
	}
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
