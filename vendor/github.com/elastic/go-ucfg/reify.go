package ucfg

import (
	"reflect"
	"regexp"
	"time"
)

func (c *Config) Unpack(to interface{}, options ...Option) error {
	opts := makeOptions(options)

	if c == nil {
		return raiseNil(ErrNilConfig)
	}
	if to == nil {
		return raiseNil(ErrNilValue)
	}

	vTo := reflect.ValueOf(to)

	k := vTo.Kind()
	isValid := to != nil && (k == reflect.Ptr || k == reflect.Map)
	if !isValid {
		return raisePointerRequired(vTo)
	}

	return reifyInto(opts, vTo, c)
}

func reifyInto(opts *options, to reflect.Value, from *Config) Error {
	to = chaseValuePointers(to)

	if to, ok := tryTConfig(to); ok {
		return mergeConfig(opts, to.Addr().Interface().(*Config), from)
	}

	tTo := chaseTypePointers(to.Type())
	k := tTo.Kind()

	switch k {
	case reflect.Map:
		return reifyMap(opts, to, from)
	case reflect.Struct:
		return reifyStruct(opts, to, from)
	case reflect.Slice, reflect.Array:
		fopts := fieldOptions{opts: opts, tag: tagOptions{}, validators: nil}
		v, err := reifyMergeValue(fopts, to, cfgSub{from})
		if err != nil {
			return err
		}
		to.Set(v)
		return nil
	}

	return raiseInvalidTopLevelType(to.Interface())
}

func reifyMap(opts *options, to reflect.Value, from *Config) Error {
	if to.Type().Key().Kind() != reflect.String {
		return raiseKeyInvalidTypeUnpack(to.Type(), from)
	}

	fields := from.fields.dict()
	if len(fields) == 0 {
		return nil
	}

	if to.IsNil() {
		to.Set(reflect.MakeMap(to.Type()))
	}
	for k, value := range fields {
		key := reflect.ValueOf(k)

		old := to.MapIndex(key)
		var v reflect.Value
		var err Error

		if !old.IsValid() {
			v, err = reifyValue(fieldOptions{opts: opts}, to.Type().Elem(), value)
		} else {
			v, err = reifyMergeValue(fieldOptions{opts: opts}, old, value)
		}

		if err != nil {
			return err
		}
		to.SetMapIndex(key, v)
	}

	return nil
}

func reifyStruct(opts *options, orig reflect.Value, cfg *Config) Error {
	orig = chaseValuePointers(orig)

	to := chaseValuePointers(reflect.New(chaseTypePointers(orig.Type())))
	if orig.Kind() == reflect.Struct { // if orig is has been allocated copy into to
		to.Set(orig)
	}

	if v, ok := implementsUnpacker(to); ok {
		reified, err := cfgSub{cfg}.reify(opts)
		if err != nil {
			return raisePathErr(err, cfg.metadata, "", cfg.Path("."))
		}

		if err := unpackWith(cfg.ctx, cfg.metadata, v, reified); err != nil {
			return err
		}
	} else {
		numField := to.NumField()
		for i := 0; i < numField; i++ {
			stField := to.Type().Field(i)
			vField := to.Field(i)
			name, tagOpts := parseTags(stField.Tag.Get(opts.tag))

			validators, err := parseValidatorTags(stField.Tag.Get(opts.validatorTag))
			if err != nil {
				return raiseCritical(err, "")
			}

			if tagOpts.squash {
				vField := chaseValue(vField)
				switch vField.Kind() {
				case reflect.Struct, reflect.Map:
					if err := reifyInto(opts, vField, cfg); err != nil {
						return err
					}
				case reflect.Slice, reflect.Array:
					fopts := fieldOptions{opts: opts, tag: tagOpts, validators: validators}
					v, err := reifyMergeValue(fopts, vField, cfgSub{cfg})
					if err != nil {
						return err
					}
					vField.Set(v)

				default:
					return raiseInlineNeedsObject(cfg, stField.Name, vField.Type())
				}
			} else {
				name = fieldName(name, stField.Name)
				fopts := fieldOptions{opts: opts, tag: tagOpts, validators: validators}
				if err := reifyGetField(cfg, fopts, name, vField); err != nil {
					return err
				}
			}
		}
	}

	if err := tryValidate(to); err != nil {
		return raiseValidation(cfg.ctx, cfg.metadata, "", err)
	}

	orig.Set(pointerize(orig.Type(), to.Type(), to))
	return nil
}

func reifyGetField(
	cfg *Config,
	opts fieldOptions,
	name string,
	to reflect.Value,
) Error {
	p := parsePath(name, opts.opts.pathSep)
	value, err := p.GetValue(cfg, opts.opts)
	if err != nil {
		if err.Reason() != ErrMissing {
			return err
		}
		value = nil
	}

	if isNil(value) {
		if err := runValidators(nil, opts.validators); err != nil {
			return raiseValidation(cfg.ctx, cfg.metadata, name, err)
		}
		return nil
	}

	v, err := reifyMergeValue(opts, to, value)
	if err != nil {
		return err
	}

	to.Set(v)
	return nil
}

func reifyValue(
	opts fieldOptions,
	t reflect.Type,
	val value,
) (reflect.Value, Error) {
	if t.Kind() == reflect.Interface && t.NumMethod() == 0 {
		reified, err := val.reify(opts.opts)
		if err != nil {
			ctx := val.Context()
			return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
		}
		return reflect.ValueOf(reified), nil
	}

	baseType := chaseTypePointers(t)
	if tConfig.ConvertibleTo(baseType) {
		cfg, err := val.toConfig(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseExpectedObject(opts.opts, val)
		}

		v := reflect.ValueOf(cfg).Convert(reflect.PtrTo(baseType))
		if t == baseType { // copy config
			v = v.Elem()
		} else {
			v = pointerize(t, baseType, v)
		}
		return v, nil
	}

	if baseType.Kind() == reflect.Struct {
		sub, err := val.toConfig(opts.opts)
		if err != nil {
			return reifyPrimitive(opts, val, t, baseType)
		}

		newSt := reflect.New(baseType)
		if err := reifyInto(opts.opts, newSt, sub); err != nil {
			return reflect.Value{}, err
		}

		if t.Kind() != reflect.Ptr {
			return newSt.Elem(), nil
		}
		return pointerize(t, baseType, newSt), nil
	}

	switch baseType.Kind() {
	case reflect.Map:
		sub, err := val.toConfig(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseExpectedObject(opts.opts, val)
		}

		if baseType.Key().Kind() != reflect.String {
			return reflect.Value{}, raiseKeyInvalidTypeUnpack(baseType, sub)
		}

		newMap := reflect.MakeMap(baseType)
		if err := reifyInto(opts.opts, newMap, sub); err != nil {
			return reflect.Value{}, err
		}
		return newMap, nil

	case reflect.Slice:
		v, err := reifySlice(opts, baseType, val)
		if err != nil {
			return reflect.Value{}, err
		}
		return pointerize(t, baseType, v), nil
	}

	return reifyPrimitive(opts, val, t, baseType)
}

func reifyMergeValue(
	opts fieldOptions,
	oldValue reflect.Value, val value,
) (reflect.Value, Error) {
	old := chaseValueInterfaces(oldValue)
	t := old.Type()
	old = chaseValuePointers(old)
	if (old.Kind() == reflect.Ptr || old.Kind() == reflect.Interface) && old.IsNil() {
		return reifyValue(opts, t, val)
	}

	baseType := chaseTypePointers(old.Type())
	if tConfig.ConvertibleTo(baseType) {
		sub, err := val.toConfig(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseExpectedObject(opts.opts, val)
		}

		if t == baseType {
			// no pointer -> return type mismatch
			return reflect.Value{}, raisePointerRequired(oldValue)
		}

		// check if old is nil -> copy reference only
		if old.Kind() == reflect.Ptr && old.IsNil() {
			v, err := val.reflect(opts.opts)
			if err != nil {
				ctx := val.Context()
				return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
			}

			v = v.Convert(reflect.PtrTo(baseType))
			return pointerize(t, baseType, v), nil
		}

		// check if old == value
		subOld := chaseValuePointers(old).Addr().Convert(tConfigPtr).Interface().(*Config)
		if sub == subOld {
			return oldValue, nil
		}

		// old != value -> merge value into old
		return oldValue, mergeConfig(opts.opts, subOld, sub)
	}

	switch baseType.Kind() {
	case reflect.Map:
		sub, err := val.toConfig(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseExpectedObject(opts.opts, val)
		}
		return old, reifyMap(opts.opts, old, sub)

	case reflect.Struct:
		sub, err := val.toConfig(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseExpectedObject(opts.opts, val)
		}
		return oldValue, reifyStruct(opts.opts, old, sub)

	case reflect.Array:
		return reifyArray(opts, old, baseType, val)

	case reflect.Slice:
		return reifySlice(opts, baseType, val)
	}

	if v, ok := implementsUnpacker(old); ok {
		reified, err := val.reify(opts.opts)
		if err != nil {
			ctx := val.Context()
			return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
		}

		if err := unpackWith(val.Context(), val.meta(), v, reified); err != nil {
			return reflect.Value{}, err
		}
		return old, nil
	}
	return reifyPrimitive(opts, val, t, baseType)
}

func reifyArray(
	opts fieldOptions,
	to reflect.Value, tTo reflect.Type,
	val value,
) (reflect.Value, Error) {
	arr, err := castArr(opts.opts, val)
	if err != nil {
		return reflect.Value{}, err
	}

	if len(arr) != tTo.Len() {
		ctx := val.Context()
		return reflect.Value{}, raiseArraySize(ctx, val.meta(), len(arr), tTo.Len())
	}
	return reifyDoArray(opts, to, tTo.Elem(), val, arr)
}

func reifySlice(
	opts fieldOptions,
	tTo reflect.Type,
	val value,
) (reflect.Value, Error) {
	arr, err := castArr(opts.opts, val)
	if err != nil {
		return reflect.Value{}, err
	}

	to := reflect.MakeSlice(tTo, len(arr), len(arr))
	return reifyDoArray(opts, to, tTo.Elem(), val, arr)
}

func reifyDoArray(
	opts fieldOptions,
	to reflect.Value, elemT reflect.Type,
	val value,
	arr []value,
) (reflect.Value, Error) {
	for i, from := range arr {
		v, err := reifyValue(opts, elemT, from)
		if err != nil {
			return reflect.Value{}, err
		}
		to.Index(i).Set(v)
	}

	if err := runValidators(to.Interface(), opts.validators); err != nil {
		ctx := val.Context()
		return reflect.Value{}, raiseValidation(ctx, val.meta(), "", err)
	}

	return to, nil
}

func castArr(opts *options, v value) ([]value, Error) {
	if sub, ok := v.(cfgSub); ok {
		return sub.c.fields.array(), nil
	}
	if ref, ok := v.(*cfgRef); ok {
		unrefed, err := ref.resolve(opts)
		if err != nil {
			return nil, raiseMissing(ref.ctx.getParent(), ref.ref.String())
		}

		if sub, ok := unrefed.(cfgSub); ok {
			return sub.c.fields.array(), nil
		}
	}

	l, err := v.Len(opts)
	if err != nil {
		ctx := v.Context()
		return nil, raisePathErr(err, v.meta(), "", ctx.path("."))
	}

	if l == 0 {
		return nil, nil
	}

	return []value{v}, nil
}

func reifyPrimitive(
	opts fieldOptions,
	val value,
	t, baseType reflect.Type,
) (reflect.Value, Error) {
	// zero initialize value if val==nil
	if isNil(val) {
		return pointerize(t, baseType, reflect.Zero(baseType)), nil
	}

	var v reflect.Value
	var err Error
	var ok bool

	if v, ok = typeIsUnpacker(baseType); ok {
		reified, err := val.reify(opts.opts)
		if err != nil {
			ctx := val.Context()
			return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
		}

		if err := unpackWith(val.Context(), val.meta(), v, reified); err != nil {
			return reflect.Value{}, err
		}
	} else {
		v, err = doReifyPrimitive(opts, val, baseType)
		if err != nil {
			return v, err
		}
	}

	if err := runValidators(v.Interface(), opts.validators); err != nil {
		return reflect.Value{}, raiseValidation(val.Context(), val.meta(), "", err)
	}

	if err := tryValidate(v); err != nil {
		return reflect.Value{}, raiseValidation(val.Context(), val.meta(), "", err)
	}

	return pointerize(t, baseType, chaseValuePointers(v)), nil
}

func doReifyPrimitive(
	opts fieldOptions,
	val value,
	baseType reflect.Type,
) (reflect.Value, Error) {
	extras := map[reflect.Type]func(fieldOptions, value, reflect.Type) (reflect.Value, Error){
		tDuration: reifyDuration,
		tRegexp:   reifyRegexp,
	}

	valT, err := val.typ(opts.opts)
	if err != nil {
		ctx := val.Context()
		return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
	}

	// try primitive conversion
	kind := baseType.Kind()
	switch {
	case valT.gotype == baseType:
		v, err := val.reflect(opts.opts)
		if err != nil {
			ctx := val.Context()
			return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
		}
		return v, nil

	case kind == reflect.String:
		s, err := val.toString(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseConversion(opts.opts, val, err, "string")
		}
		return reflect.ValueOf(s), nil

	case extras[baseType] != nil:
		v, err := extras[baseType](opts, val, baseType)
		if err != nil {
			return v, err
		}
		return v, nil

	case isInt(kind):
		v, err := reifyInt(opts, val, baseType)
		if err != nil {
			return v, err
		}
		return v, nil

	case isUint(kind):
		v, err := reifyUint(opts, val, baseType)
		if err != nil {
			return v, err
		}
		return v, nil

	case isFloat(kind):
		v, err := reifyFloat(opts, val, baseType)
		if err != nil {
			return v, err
		}
		return v, nil

	case kind == reflect.Bool:
		v, err := reifyBool(opts, val, baseType)
		if err != nil {
			return v, err
		}
		return v, nil

	case valT.gotype.ConvertibleTo(baseType):
		v, err := val.reflect(opts.opts)
		if err != nil {
			ctx := val.Context()
			return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
		}
		return v.Convert(baseType), nil
	}

	return reflect.Value{}, raiseToTypeNotSupported(opts.opts, val, baseType)
}

func reifyDuration(
	opts fieldOptions,
	val value,
	_ reflect.Type,
) (reflect.Value, Error) {
	var d time.Duration
	var err error

	switch v := val.(type) {
	case *cfgInt:
		d = time.Duration(v.i) * time.Second
	case *cfgUint:
		d = time.Duration(v.u) * time.Second
	case *cfgFloat:
		d = time.Duration(v.f * float64(time.Second))
	case *cfgString:
		d, err = time.ParseDuration(v.s)
	default:
		var s string
		s, err = val.toString(opts.opts)
		if err != nil {
			return reflect.Value{}, raiseInvalidDuration(val, err)
		}

		d, err = time.ParseDuration(s)
	}

	if err != nil {
		return reflect.Value{}, raiseInvalidDuration(val, err)
	}
	return reflect.ValueOf(d), nil
}

func reifyRegexp(
	opts fieldOptions,
	val value,
	_ reflect.Type,
) (reflect.Value, Error) {
	s, err := val.toString(opts.opts)
	if err != nil {
		return reflect.Value{}, raiseConversion(opts.opts, val, err, "regex")
	}

	r, err := regexp.Compile(s)
	if err != nil {
		return reflect.Value{}, raiseInvalidRegexp(val, err)
	}
	return reflect.ValueOf(r).Elem(), nil
}

func reifyInt(
	opts fieldOptions,
	val value,
	t reflect.Type,
) (reflect.Value, Error) {
	i, err := val.toInt(opts.opts)
	if err != nil {
		return reflect.Value{}, raiseConversion(opts.opts, val, err, "int")
	}

	tmp := reflect.Zero(t)
	if tmp.OverflowInt(i) {
		return reflect.Value{}, raiseConversion(opts.opts, val, ErrOverflow, "int")
	}
	return reflect.ValueOf(i).Convert(t), nil
}

func reifyUint(
	opts fieldOptions,
	val value,
	t reflect.Type,
) (reflect.Value, Error) {
	u, err := val.toUint(opts.opts)
	if err != nil {
		return reflect.Value{}, raiseConversion(opts.opts, val, err, "uint")
	}

	tmp := reflect.Zero(t)
	if tmp.OverflowUint(u) {
		return reflect.Value{}, raiseConversion(opts.opts, val, ErrOverflow, "uint")
	}
	return reflect.ValueOf(u).Convert(t), nil
}

func reifyFloat(
	opts fieldOptions,
	val value,
	t reflect.Type,
) (reflect.Value, Error) {
	f, err := val.toFloat(opts.opts)
	if err != nil {
		return reflect.Value{}, raiseConversion(opts.opts, val, err, "float")
	}

	tmp := reflect.Zero(t)
	if tmp.OverflowFloat(f) {
		return reflect.Value{}, raiseConversion(opts.opts, val, ErrOverflow, "float")
	}
	return reflect.ValueOf(f).Convert(t), nil
}

func reifyBool(
	opts fieldOptions,
	val value,
	t reflect.Type,
) (reflect.Value, Error) {
	b, err := val.toBool(opts.opts)
	if err != nil {
		return reflect.Value{}, raiseConversion(opts.opts, val, err, "bool")
	}
	return reflect.ValueOf(b).Convert(t), nil
}
