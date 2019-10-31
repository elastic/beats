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

package gotype

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	structform "github.com/elastic/go-structform"
)

type typeFoldRegistry struct {
	// mu sync.RWMutex
	m map[typeFoldKey]reFoldFn
}

type typeFoldKey struct {
	ty     reflect.Type
	inline bool
}

var _foldRegistry = newTypeFoldRegistry()

func getReflectFold(c *foldContext, t reflect.Type) (reFoldFn, error) {
	var err error

	f := c.reg.find(t)
	if f != nil {
		return f, nil
	}

	f = getReflectFoldPrimitive(t)
	if f != nil {
		c.reg.set(t, f)
		return f, nil
	}

	if t.Implements(tFolder) {
		f := reFoldFolderIfc
		c.reg.set(t, f)
		return f, nil
	}

	switch t.Kind() {
	case reflect.Ptr:
		f, err = getFoldPointer(c, t)
	case reflect.Struct:
		f, err = getReflectFoldStruct(c, t, false)
	case reflect.Map:
		f, err = getReflectFoldMap(c, t)
	case reflect.Slice, reflect.Array:
		f, err = getReflectFoldSlice(c, t)
	case reflect.Interface:
		f, err = getReflectFoldElem(c, t)
	default:
		f, err = getReflectFoldPrimitiveKind(t)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}
	c.reg.set(t, f)
	return f, nil
}

func getReflectFoldMap(c *foldContext, t reflect.Type) (reFoldFn, error) {
	iterVisitor, err := getReflectFoldMapKeys(c, t)
	if err != nil {
		return nil, err
	}

	return func(C *foldContext, rv reflect.Value) error {
		if err := C.OnObjectStart(rv.Len(), structform.AnyType); err != nil {
			return err
		}
		if err := iterVisitor(C, rv); err != nil {
			return err
		}
		return C.OnObjectFinished()
	}, nil
}

func getFoldPointer(c *foldContext, t reflect.Type) (reFoldFn, error) {
	N, bt := baseType(t)
	elemVisitor, err := getReflectFold(c, bt)
	if err != nil {
		return nil, err
	}
	return makePointerFold(N, elemVisitor), nil
}

func makePointerFold(N int, elemVisitor reFoldFn) reFoldFn {
	if N == 0 {
		return elemVisitor
	}

	return func(C *foldContext, v reflect.Value) error {
		for i := 0; i < N; i++ {
			if v.IsNil() {
				return C.OnNil()
			}
			v = v.Elem()
		}
		return elemVisitor(C, v)
	}
}

func getReflectFoldElem(c *foldContext, t reflect.Type) (reFoldFn, error) {
	return foldInterfaceElem, nil
}

func foldInterfaceElem(C *foldContext, v reflect.Value) error {
	if v.IsNil() {
		return C.visitor.OnNil()
	}
	return foldAnyReflect(C, v.Elem())
}

func getReflectFoldStruct(c *foldContext, t reflect.Type, inline bool) (reFoldFn, error) {
	fields, err := getStructFieldsFolds(c, t)
	if err != nil {
		return nil, err
	}

	if inline {
		return makeFieldsFold(fields), nil
	}
	return makeStructFold(fields), nil
}

// TODO: benchmark field accessors based on pointer offsets
func getStructFieldsFolds(c *foldContext, t reflect.Type) ([]reFoldFn, error) {
	count := t.NumField()
	fields := make([]reFoldFn, 0, count)

	for i := 0; i < count; i++ {
		fv, err := buildFieldFold(c, t, i)
		if err != nil {
			return nil, err
		}

		if fv == nil {
			continue
		}

		fields = append(fields, fv)
	}

	if len(fields) < cap(fields) {
		tmp := make([]reFoldFn, len(fields))
		copy(tmp, fields)
		fields = tmp
	}

	return fields, nil
}

func makeStructFold(fields []reFoldFn) reFoldFn {
	fieldsVisitor := makeFieldsFold(fields)
	return func(C *foldContext, v reflect.Value) error {
		if err := C.OnObjectStart(len(fields), structform.AnyType); err != nil {
			return err
		}
		if err := fieldsVisitor(C, v); err != nil {
			return err
		}
		return C.OnObjectFinished()
	}
}

func makeFieldsFold(fields []reFoldFn) reFoldFn {
	return func(C *foldContext, v reflect.Value) error {
		for _, fv := range fields {
			if err := fv(C, v); err != nil {
				return err
			}
		}
		return nil
	}
}

func buildFieldFold(C *foldContext, t reflect.Type, idx int) (reFoldFn, error) {
	st := t.Field(idx)

	name := st.Name
	rune, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsUpper(rune) {
		// ignore non exported fields
		return nil, nil
	}

	tagName, tagOpts := parseTags(st.Tag.Get(C.opts.tag))
	if tagOpts.squash && tagOpts.omitEmpty {
		return nil, errInlineAndOmitEmpty
	}

	if tagOpts.omit {
		// ignore omitted fields
		return nil, nil
	}

	if tagOpts.squash {
		return buildFieldFoldInline(C, t, idx, tagOpts.omitEmpty)
	}

	foldT := st.Type
	if tagOpts.omitEmpty {
		_, foldT = baseType(st.Type)
	}
	valueVisitor, err := getReflectFold(C, foldT)
	if err != nil {
		return nil, err
	}

	if tagName != "" {
		name = tagName
	} else {
		name = strings.ToLower(name)
	}

	if tagOpts.omitEmpty {
		return makeNonEmptyFieldFold(name, idx, st.Type, valueVisitor)
	}
	return makeFieldFold(name, idx, valueVisitor)
}

func buildFieldFoldInline(
	C *foldContext,
	t reflect.Type,
	idx int,
	omitEmpty bool,
) (reFoldFn, error) {
	var (
		st          = t.Field(idx)
		N, bt       = baseType(st.Type)
		baseVisitor reFoldFn
		err         error
	)

	f := C.reg.findInline(st.Type)
	if f != nil {
		return makeFieldInlineFold(idx, f), nil
	}

	baseVisitor = C.reg.findInline(bt)
	if baseVisitor == nil {
		baseVisitor, err = fieldFoldGenInline(C, bt)
		if err != nil {
			return nil, err
		}
		C.reg.setInline(bt, baseVisitor)
	}

	f = makePointerFold(N, baseVisitor)
	C.reg.setInline(st.Type, f)

	return makeFieldInlineFold(idx, f), nil
}

func fieldFoldGenInline(C *foldContext, t reflect.Type) (reFoldFn, error) {
	if C.userReg != nil {
		if f := C.userReg[t]; f != nil {
			f = embeddObjReFold(C, f)
		}
	}

	if t.Implements(tFolder) {
		return embeddObjReFold(C, reFoldFolderIfc), nil
	}

	switch t.Kind() {
	case reflect.Struct:
		return getReflectFoldStruct(C, t, true)
	case reflect.Map:
		return getReflectFoldMapKeys(C, t)
	case reflect.Interface:
		return getReflectFoldInlineInterface(C, t)
	}

	return nil, errSquashNeedObject
}

func makeFieldFold(name string, idx int, fn reFoldFn) (reFoldFn, error) {
	return func(C *foldContext, v reflect.Value) error {
		if err := C.OnKey(name); err != nil {
			return err
		}
		return fn(C, v.Field(idx))
	}, nil
}

func makeFieldInlineFold(idx int, fn reFoldFn) reFoldFn {
	return func(C *foldContext, v reflect.Value) error {
		return fn(C, v.Field(idx))
	}
}

func makeNonEmptyFieldFold(name string, idx int, t reflect.Type, fn reFoldFn) (reFoldFn, error) {
	resolver := makeResolveValue(t)
	if resolver == nil {
		return makeFieldFold(name, idx, fn)
	}

	return func(C *foldContext, v reflect.Value) (err error) {
		field, ok := resolver(v.Field(idx))
		if ok {
			if err = C.OnKey(name); err != nil {
				return
			}
			err = fn(C, field)
		}
		return
	}, nil
}

func makeResolveValue(st reflect.Type) func(reflect.Value) (reflect.Value, bool) {
	type resolver func(reflect.Value) (reflect.Value, bool)

	resolveBySize := func(v reflect.Value) (reflect.Value, bool) {
		return v, v.Len() > 0
	}

	resolveNonNil := func(v reflect.Value) (reflect.Value, bool) {
		return v, !v.IsNil()
	}

	var resolvers []resolver
	for {
		switch st.Kind() {
		case reflect.Ptr:
			var r resolver
			st, r = makeResolvePointers(st)
			resolvers = append(resolvers, r)
			continue
		case reflect.Interface:
			resolvers = append(resolvers, resolveNonNil)
		case reflect.Map, reflect.String, reflect.Slice, reflect.Array:
			resolvers = append(resolvers, resolveBySize)
		default:
		}
		break
	}

	if len(resolvers) == 0 {
		return nil
	}
	if len(resolvers) == 1 {
		return resolvers[0]
	}

	return func(v reflect.Value) (reflect.Value, bool) {
		for _, r := range resolvers {
			var ok bool
			if v, ok = r(v); !ok {
				return v, ok
			}
		}
		return v, true
	}
}

func makeResolvePointers(st reflect.Type) (reflect.Type, func(reflect.Value) (reflect.Value, bool)) {
	N, bt := baseType(st)
	return bt, func(v reflect.Value) (reflect.Value, bool) {
		for i := 0; i < N; i++ {
			if v.IsNil() {
				return v, false
			}
			v = v.Elem()
		}
		return v, true
	}
}

func getReflectFoldSlice(c *foldContext, t reflect.Type) (reFoldFn, error) {
	elemVisitor, err := getReflectFold(c, t.Elem())
	if err != nil {
		return nil, err
	}

	return func(C *foldContext, rv reflect.Value) error {
		count := rv.Len()

		if err := C.OnArrayStart(count, structform.AnyType); err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			if err := elemVisitor(C, rv.Index(i)); err != nil {
				return err
			}
		}

		return C.OnArrayFinished()
	}, nil
}

/*
// TODO: create visitors casting the actual values via reflection instead of
//       golang type conversion:
func getReflectFoldPrimitive(t reflect.Type) reFoldFn {
	switch t.Kind() {
	case reflect.Bool:
		return reFoldBool
	case reflect.Int:
		return reFoldInt
	case reflect.Int8:
		return reFoldInt8
	case reflect.Int16:
		return reFoldInt16
	case reflect.Int32:
		return reFoldInt32
	case reflect.Int64:
		return reFoldInt64
	case reflect.Uint:
		return reFoldUint
	case reflect.Uint8:
		return reFoldUint8
	case reflect.Uint16:
		return reFoldUint16
	case reflect.Uint32:
		return reFoldUint32
	case reflect.Uint64:
		return reFoldUint64
	case reflect.Float32:
		return reFoldFloat32
	case reflect.Float64:
		return reFoldFloat64
	case reflect.String:
		return reFoldString

	case reflect.Slice:
		switch t.Elem().Kind() {
		case reflect.Interface:
			return reFoldArrAny
		case reflect.Bool:
			return reFoldArrBool
		case reflect.Int:
			return reFoldArrInt
		case reflect.Int8:
			return reFoldArrInt8
		case reflect.Int16:
			return reFoldArrInt16
		case reflect.Int32:
			return reFoldArrInt32
		case reflect.Int64:
			return reFoldArrInt64
		case reflect.Uint:
			return reFoldArrUint
		case reflect.Uint8:
			return reFoldArrUint8
		case reflect.Uint16:
			return reFoldArrUint16
		case reflect.Uint32:
			return reFoldArrUint32
		case reflect.Uint64:
			return reFoldArrUint64
		case reflect.Float32:
			return reFoldArrFloat32
		case reflect.Float64:
			return reFoldArrFloat64
		case reflect.String:
			return reFoldArrString
		}

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil
		}

		switch t.Elem().Kind() {
		case reflect.Interface:
			return reflectMapAny
		case reflect.Bool:
			return reFoldMapBool
		case reflect.Int:
			return reFoldMapInt
		case reflect.Int8:
			return reFoldMapInt8
		case reflect.Int16:
			return reFoldMapInt16
		case reflect.Int32:
			return reFoldMapInt32
		case reflect.Int64:
			return reFoldMapInt64
		case reflect.Uint:
			return reFoldMapUint
		case reflect.Uint8:
			return reFoldMapUint8
		case reflect.Uint16:
			return reFoldMapUint16
		case reflect.Uint32:
			return reFoldMapUint32
		case reflect.Uint64:
			return reFoldMapUint64
		case reflect.Float32:
			return reFoldMapFloat32
		case reflect.Float64:
			return reFoldMapFloat64
		case reflect.String:
			return reFoldMapString
		}
	}

	return nil
}
*/

func foldAnyReflect(C *foldContext, v reflect.Value) error {
	f, err := getReflectFold(C, v.Type())
	if err != nil {
		return err
	}
	return f(C, v)
}

func newTypeFoldRegistry() *typeFoldRegistry {
	return &typeFoldRegistry{m: map[typeFoldKey]reFoldFn{}}
}

func (r *typeFoldRegistry) find(t reflect.Type) reFoldFn {
	// r.mu.RLock()
	// defer r.mu.RUnlock()
	return r.m[typeFoldKey{ty: t, inline: false}]
}

func (r *typeFoldRegistry) findInline(t reflect.Type) reFoldFn {
	// r.mu.RLock()
	// defer r.mu.RUnlock()
	return r.m[typeFoldKey{ty: t, inline: true}]
}

func (r *typeFoldRegistry) set(t reflect.Type, f reFoldFn) {
	// r.mu.Lock()
	// defer r.mu.Unlock()
	r.m[typeFoldKey{ty: t, inline: false}] = f
}

func (r *typeFoldRegistry) setInline(t reflect.Type, f reFoldFn) {
	// r.mu.Lock()
	// defer r.mu.Unlock()
	r.m[typeFoldKey{ty: t, inline: true}] = f
}

func liftFold(sample interface{}, fn foldFn) reFoldFn {
	t := reflect.TypeOf(sample)
	return func(C *foldContext, v reflect.Value) error {
		if v.Type().Name() != "" {
			v = v.Convert(t)
		}
		return fn(C, v.Interface())
	}
}

func baseType(t reflect.Type) (int, reflect.Type) {
	i := 0
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		i++
	}
	return i, t
}
