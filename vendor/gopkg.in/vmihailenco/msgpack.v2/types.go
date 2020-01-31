package msgpack

import (
	"reflect"
	"sync"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

var customEncoderType = reflect.TypeOf((*CustomEncoder)(nil)).Elem()
var customDecoderType = reflect.TypeOf((*CustomDecoder)(nil)).Elem()

var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()

type encoderFunc func(*Encoder, reflect.Value) error
type decoderFunc func(*Decoder, reflect.Value) error

var typEncMap = make(map[reflect.Type]encoderFunc)
var typDecMap = make(map[reflect.Type]decoderFunc)

// Register registers encoder and decoder functions for a type.
// In most cases you should prefer implementing CustomEncoder and
// CustomDecoder interfaces.
func Register(typ reflect.Type, enc encoderFunc, dec decoderFunc) {
	if enc != nil {
		typEncMap[typ] = enc
	}
	if dec != nil {
		typDecMap[typ] = dec
	}
}

//------------------------------------------------------------------------------

var structs = newStructCache()

type structCache struct {
	l sync.RWMutex
	m map[reflect.Type]*fields
}

func newStructCache() *structCache {
	return &structCache{
		m: make(map[reflect.Type]*fields),
	}
}

func (m *structCache) Fields(typ reflect.Type) *fields {
	m.l.RLock()
	fs, ok := m.m[typ]
	m.l.RUnlock()
	if !ok {
		m.l.Lock()
		fs, ok = m.m[typ]
		if !ok {
			fs = getFields(typ)
			m.m[typ] = fs
		}
		m.l.Unlock()
	}

	return fs
}

//------------------------------------------------------------------------------

type field struct {
	name      string
	index     []int
	omitEmpty bool

	encoder encoderFunc
	decoder decoderFunc
}

func (f *field) value(strct reflect.Value) reflect.Value {
	return strct.FieldByIndex(f.index)
}

func (f *field) Omit(strct reflect.Value) bool {
	return f.omitEmpty && isEmptyValue(f.value(strct))
}

func (f *field) EncodeValue(e *Encoder, strct reflect.Value) error {
	return f.encoder(e, f.value(strct))
}

func (f *field) DecodeValue(d *Decoder, strct reflect.Value) error {
	return f.decoder(d, f.value(strct))
}

//------------------------------------------------------------------------------

type fields struct {
	List  []*field
	Table map[string]*field

	asArray   bool
	omitEmpty bool
}

func newFields(numField int) *fields {
	return &fields{
		List:  make([]*field, 0, numField),
		Table: make(map[string]*field, numField),
	}
}

func (fs *fields) Len() int {
	return len(fs.List)
}

func (fs *fields) Add(field *field) {
	fs.List = append(fs.List, field)
	fs.Table[field.name] = field
	if field.omitEmpty {
		fs.omitEmpty = field.omitEmpty
	}
}

func (fs *fields) OmitEmpty(strct reflect.Value) []*field {
	if !fs.omitEmpty {
		return fs.List
	}

	fields := make([]*field, 0, fs.Len())
	for _, f := range fs.List {
		if !f.Omit(strct) {
			fields = append(fields, f)
		}
	}
	return fields
}

func getFields(typ reflect.Type) *fields {
	numField := typ.NumField()
	fs := newFields(numField)

	var omitEmpty bool
	for i := 0; i < numField; i++ {
		f := typ.Field(i)

		name, opt := parseTag(f.Tag.Get("msgpack"))
		if name == "-" {
			continue
		}

		if f.Name == "_msgpack" {
			if opt.Contains("asArray") {
				fs.asArray = true
			}
			if opt.Contains("omitempty") {
				omitEmpty = true
			}
		}

		if f.PkgPath != "" && !f.Anonymous {
			continue
		}

		if opt.Contains("inline") {
			inlineFields(fs, f)
			continue
		}

		if name == "" {
			name = f.Name
		}
		field := field{
			name:      name,
			index:     f.Index,
			omitEmpty: omitEmpty || opt.Contains("omitempty"),
			encoder:   getEncoder(f.Type),
			decoder:   getDecoder(f.Type),
		}
		fs.Add(&field)
	}
	return fs
}

func inlineFields(fs *fields, f reflect.StructField) {
	typ := f.Type
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	inlinedFields := getFields(typ).List
	for _, field := range inlinedFields {
		if _, ok := fs.Table[field.name]; ok {
			// Don't overwrite shadowed fields.
			continue
		}
		field.index = append(f.Index, field.index...)
		fs.Add(field)
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
