// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ucfg

import (
	"reflect"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"
)

// Unpack unpacks c into a struct, a map, or a slice allocating maps, slices,
// and pointers as necessary.
//
// Unpack supports the options: PathSep, StructTag, ValidatorTag, Env, Resolve,
// ResolveEnv, ReplaceValues, AppendValues, PrependValues.
//
// When unpacking into a value, Unpack first will try to call Unpack if the
// value implements the Unpacker interface. Otherwise, Unpack tries to convert
// the internal value into the target type:
//
//  # Primitive types
//
//  bool: requires setting of type bool or string which parses into a
//     boolean value (true, false, on, off)
//  int(8, 16, 32, 64): requires any number type convertible to int or a string
//      parsing to int. Fails if the target value would overflow.
//  uint(8, 16, 32, 64): requires any number type convertible to int or a string
//       parsing to int. Fails if the target value is negative or would overflow.
//  float(32, 64): requires any number type convertible to float or a string
//       parsing to float. Fails if the target value is negative or would overflow.
//  string: requires any primitive value which is serialized into a string.
//
//  # Special types:
//
//  time.Duration: requires a number setting converted to seconds or a string
//       parsed into time.Duration via time.ParseDuration.
//  *regexp.Regexp: requires a string being compiled into a regular expression
//       using regexp.Compile.
//  *Config: requires a Config object to be stored by pointer into the target
//       value. Can be used to capture a sub-Config without interpreting
//       the settings yet.
//
//   # Arrays/Slices:
//
//  Requires a Config object with indexed entries. Named entries will not be
//  unpacked into the Array/Slice. Primitive values will be handled like arrays
//  of length 1.
//
//  # Map
//
//  Requires a Config object with all named top-level entries being unpacked into
//  the map.
//
//  # Struct
//
//  Requires a Config object. All named values in the Config object will be unpacked
//  into the struct its fields, if the name is available in the struct.
//  A field its name is set using the `config` struct tag (configured by StructTag)
//  If tag is missing or no field name is configured in the tag, the field name
//  itself will be used.
//  If the tag sets the `,ignore` flag, the field will not be overwritten.
//  If the tag sets the `,inline` or `,squash` flag, Unpack will apply the current
//  configuration namespace to the fields.
//  If the tag option `replace` is configured, arrays and *ucfg.Config
//  convertible fields are replaced by the new values.
//  If the tag options `append` or `prepend` is used, arrays will be merged by
//  appending/prepending the new array contents.
//  The struct tag options `replace`, `append`, and `prepend` overwrites the
//  global value merging strategy (e.g. ReplaceValues, AppendValues, ...) for all sub-fields.
//
// Fields available in a struct or a map, but not in the Config object, will not
// be touched. Default values should be set in the target value before calling Unpack.
//
// Type aliases like "type myTypeAlias T" are unpacked using Unpack if the alias
// implements the Unpacker interface. Otherwise unpacking rules for type T will be used.
//
// When unpacking a value, the Validate method will be called if the value
// implements the Validator interface. Unpacking a struct field the validator
// options will be applied to the unpacked value as well.
//
// Struct field validators are set using the `validate` tag (configurable by
// ValidatorTag). Default validators options are:
//
//  required: check value is set and not empty
//  nonzero: check numeric value != 0 or string/slice not being empty
//  positive: check numeric value >= 0
//  min=<value>: check numeric value >= <value>. If target type is time.Duration,
//       <value> can be a duration.
//  max=<value>: check numeric value <= <value>. If target type is time.Duration,
//     <value> can be a duration.
//
// If a config value is not the convertible to the target type, or overflows the
// target type, Unpack will abort immediately and return the appropriate error.
//
// If validator tags or validation provided by Validate or Unmarshal fails,
// Unpack will abort immediately and return the validate error.
//
// When unpacking into an interface{} value, Unpack will store a value of one of
// these types in the value:
//
//   bool for boolean values
//   int64 for signed integer values
//   uint64 for unsigned integer values
//   float64 for floating point values
//   string for string values
//   []interface{} for list-only Config objects
//   map[string]interface{} for Config objects
//   nil for pointers if key has a nil value
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

	return raiseInvalidTopLevelType(to.Interface(), opts.meta)
}

func reifyMap(opts *options, to reflect.Value, from *Config) Error {
	parentFields := opts.activeFields
	defer func() { opts.activeFields = parentFields }()

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
		opts.activeFields = newFieldSet(parentFields)
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
	parentFields := opts.activeFields
	defer func() { opts.activeFields = parentFields }()

	orig = chaseValuePointers(orig)

	to := chaseValuePointers(reflect.New(chaseTypePointers(orig.Type())))
	if orig.Kind() == reflect.Struct { // if orig is has been allocated copy into to
		to.Set(orig)
	}

	if v, ok := valueIsUnpacker(to); ok {
		err := unpackWith(opts, v, cfgSub{cfg})
		if err != nil {
			return err
		}
	} else {
		numField := to.NumField()
		for i := 0; i < numField; i++ {
			stField := to.Type().Field(i)

			// ignore non exported fields
			if rune, _ := utf8.DecodeRuneInString(stField.Name); !unicode.IsUpper(rune) {
				continue
			}
			name, tagOpts := parseTags(stField.Tag.Get(opts.tag))
			if tagOpts.ignore {
				continue
			}

			// create new context, overwriting configValueHandling for all sub-operations
			if tagOpts.cfgHandling != opts.configValueHandling {
				tmp := &options{}
				*tmp = *opts
				tmp.configValueHandling = tagOpts.cfgHandling
				opts = tmp
			}

			opts.activeFields = newFieldSet(parentFields)

			vField := to.Field(i)
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
		return oldValue, mergeFieldConfig(opts, subOld, sub)
	}

	if v, ok := valueIsUnpacker(old); ok {
		err := unpackWith(opts.opts, v, val)
		if err != nil {
			return reflect.Value{}, err
		}
		return old, nil
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
		return reifySliceMerge(opts, old, baseType, val)
	}

	return reifyPrimitive(opts, val, t, baseType)
}

func mergeFieldConfig(opts fieldOptions, to, from *Config) Error {
	return mergeConfig(opts.opts, to, from)
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
	return reifyDoArray(opts, to, tTo.Elem(), 0, val, arr)
}

func reifySlice(
	opts fieldOptions,
	tTo reflect.Type,
	val value,
) (reflect.Value, Error) {
	return reifySliceMerge(opts, reflect.Value{}, tTo, val)
}

func reifySliceMerge(
	opts fieldOptions,
	old reflect.Value,
	tTo reflect.Type,
	val value,
) (reflect.Value, Error) {
	arr, err := castArr(opts.opts, val)
	if err != nil {
		return reflect.Value{}, err
	}

	arrMergeCfg := opts.configHandling()

	l := len(arr)
	start := 0
	cpyStart := 0

	withOld := old.IsValid() && !old.IsNil()
	if withOld {
		ol := old.Len()

		switch arrMergeCfg {
		case cfgReplaceValue:
			// do nothing

		case cfgArrAppend:
			l += ol
			start = ol

		case cfgArrPrepend:
			cpyStart = l
			l += ol

		default:
			if l < ol {
				l = ol
			}
		}
	}
	tmp := reflect.MakeSlice(tTo, l, l)

	if withOld {
		reflect.Copy(tmp.Slice(cpyStart, tmp.Len()), old)
	}
	return reifyDoArray(opts, tmp, tTo.Elem(), start, val, arr)
}

func reifyDoArray(
	opts fieldOptions,
	to reflect.Value, elemT reflect.Type,
	start int,
	val value,
	arr []value,
) (reflect.Value, Error) {
	idx := start
	for _, from := range arr {
		to.Index(idx)
		v, err := reifyMergeValue(opts, to.Index(idx), from)
		if err != nil {
			return reflect.Value{}, err
		}
		to.Index(idx).Set(v)
		idx++
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
	if ref, ok := v.(*cfgDynamic); ok {
		unrefed, err := ref.getValue(opts)
		if err != nil {
			return nil, raiseMissingMsg(ref.ctx.getParent(), ref.ctx.field, err.Error())
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
		err := unpackWith(opts.opts, v, val)
		if err != nil {
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

	previous := opts.opts.activeFields
	opts.opts.activeFields = newFieldSet(previous)
	valT, err := val.typ(opts.opts)
	if err != nil {
		ctx := val.Context()
		return reflect.Value{}, raisePathErr(err, val.meta(), "", ctx.path("."))
	}
	opts.opts.activeFields = previous

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
