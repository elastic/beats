package gotype

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	structform "github.com/urso/go-structform"
)

type typeFoldRegistry struct {
	// mu sync.RWMutex
	m map[reflect.Type]reFoldFn
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
		fv, err := fieldFold(c, t, i)
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

func fieldFold(c *foldContext, t reflect.Type, idx int) (reFoldFn, error) {
	st := t.Field(idx)

	name := st.Name
	rune, _ := utf8.DecodeRuneInString(name)
	if !unicode.IsUpper(rune) {
		// ignore non exported fields
		return nil, nil
	}

	tagName, tagOpts := parseTags(st.Tag.Get(c.opts.tag))
	if !tagOpts.squash {
		if tagName != "" {
			name = tagName
		} else {
			name = strings.ToLower(name)
		}

		valueVisitor, err := getReflectFold(c, st.Type)
		if err != nil {
			return nil, err
		}

		return makeFieldFold(name, idx, valueVisitor)
	}

	var (
		N, bt       = baseType(st.Type)
		baseVisitor reFoldFn
		err         error
	)

	switch bt.Kind() {
	case reflect.Struct:
		baseVisitor, err = getReflectFoldStruct(c, bt, true)
	case reflect.Map:
		baseVisitor, err = getReflectFoldMapKeys(c, bt)
	case reflect.Interface:
		baseVisitor, err = getReflectFoldInlineInterface(c, bt)
	default:
		err = errSquashNeedObject
	}
	if err != nil {
		return nil, err
	}

	valueVisitor := makePointerFold(N, baseVisitor)
	return makeFieldInlineFold(idx, valueVisitor)
}

func makeFieldFold(name string, idx int, fn reFoldFn) (reFoldFn, error) {
	return func(C *foldContext, v reflect.Value) error {
		if err := C.OnKey(name); err != nil {
			return err
		}
		return fn(C, v.Field(idx))
	}, nil
}

func makeFieldInlineFold(idx int, fn reFoldFn) (reFoldFn, error) {
	return func(C *foldContext, v reflect.Value) error {
		return fn(C, v.Field(idx))
	}, nil
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
	return &typeFoldRegistry{m: map[reflect.Type]reFoldFn{}}
}

func (r *typeFoldRegistry) find(t reflect.Type) reFoldFn {
	// r.mu.RLock()
	// defer r.mu.RUnlock()
	return r.m[t]
}

func (r *typeFoldRegistry) set(t reflect.Type, f reFoldFn) {
	// r.mu.Lock()
	// defer r.mu.Unlock()
	r.m[t] = f
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
