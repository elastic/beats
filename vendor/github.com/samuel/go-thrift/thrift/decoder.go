// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"reflect"
	"runtime"
)

// Decoder is the interface that allows types to deserialize themselves from a Thrift stream
type Decoder interface {
	DecodeThrift(ProtocolReader) error
}

type decoder struct {
	r ProtocolReader
}

// DecodeStruct tries to deserialize a struct from a Thrift stream
func DecodeStruct(r ProtocolReader, v interface{}) (err error) {
	if de, ok := v.(Decoder); ok {
		return de.DecodeThrift(r)
	}

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	d := &decoder{r}
	vo := reflect.ValueOf(v)
	for vo.Kind() != reflect.Ptr {
		d.error(&UnsupportedValueError{Value: vo, Str: "pointer to struct expected"})
	}
	if vo.Elem().Kind() != reflect.Struct {
		d.error(&UnsupportedValueError{Value: vo, Str: "expected a struct"})
	}
	d.readValue(TypeStruct, vo.Elem())
	return nil
}

func (d *decoder) error(err interface{}) {
	panic(err)
}

func (d *decoder) readValue(thriftType byte, rf reflect.Value) {
	v := rf
	kind := rf.Kind()
	if kind == reflect.Ptr {
		if rf.IsNil() {
			rf.Set(reflect.New(rf.Type().Elem()))
		}
		v = rf.Elem()
		kind = v.Kind()
	}

	if de, ok := rf.Interface().(Decoder); ok {
		if err := de.DecodeThrift(d.r); err != nil {
			d.error(err)
		}
		return
	}

	var err error
	switch thriftType {
	case TypeBool:
		if val, err := d.r.ReadBool(); err != nil {
			d.error(err)
		} else {
			v.SetBool(val)
		}
	case TypeByte:
		if val, err := d.r.ReadByte(); err != nil {
			d.error(err)
		} else {
			if kind == reflect.Uint8 {
				v.SetUint(uint64(val))
			} else {
				v.SetInt(int64(val))
			}
		}
	case TypeI16:
		if val, err := d.r.ReadI16(); err != nil {
			d.error(err)
		} else {
			v.SetInt(int64(val))
		}
	case TypeI32:
		if val, err := d.r.ReadI32(); err != nil {
			d.error(err)
		} else {
			if kind == reflect.Uint32 {
				v.SetUint(uint64(val))
			} else {
				v.SetInt(int64(val))
			}
		}
	case TypeI64:
		if val, err := d.r.ReadI64(); err != nil {
			d.error(err)
		} else {
			if kind == reflect.Uint64 {
				v.SetUint(uint64(val))
			} else {
				v.SetInt(val)
			}
		}
	case TypeDouble:
		if val, err := d.r.ReadDouble(); err != nil {
			d.error(err)
		} else {
			v.SetFloat(val)
		}
	case TypeString:
		if kind == reflect.Slice {
			elemType := v.Type().Elem()
			elemTypeName := elemType.Name()
			if elemType.Kind() == reflect.Uint8 && (elemTypeName == "byte" || elemTypeName == "uint8") {
				if val, err := d.r.ReadBytes(); err != nil {
					d.error(err)
				} else {
					v.SetBytes(val)
				}
			} else {
				err = &UnsupportedValueError{Value: v, Str: "decoder expected a byte array"}
			}
		} else {
			if val, err := d.r.ReadString(); err != nil {
				d.error(err)
			} else {
				v.SetString(val)
			}
		}
	case TypeStruct:
		if err := d.r.ReadStructBegin(); err != nil {
			d.error(err)
		}

		meta := encodeFields(v.Type())
		req := meta.required
		for {
			ftype, id, err := d.r.ReadFieldBegin()
			if err != nil {
				d.error(err)
			}
			if ftype == TypeStop {
				break
			}

			ef, ok := meta.fields[int(id)]
			if !ok {
				SkipValue(d.r, ftype)
			} else {
				req &= ^(uint64(1) << uint64(id))
				fieldValue := v.Field(ef.i)
				if ftype != ef.fieldType {
					d.error(&UnsupportedValueError{Value: fieldValue, Str: "type mismatch"})
				}
				d.readValue(ftype, fieldValue)
			}

			if err = d.r.ReadFieldEnd(); err != nil {
				d.error(err)
			}
		}

		if err := d.r.ReadStructEnd(); err != nil {
			d.error(err)
		}

		if req != 0 {
			for i := 0; req != 0; i, req = i+1, req>>1 {
				if req&1 != 0 {
					d.error(&MissingRequiredField{
						StructName: v.Type().Name(),
						FieldName:  meta.fields[i].name,
					})
				}
			}
		}
	case TypeMap:
		keyType := v.Type().Key()
		valueType := v.Type().Elem()
		ktype, vtype, n, err := d.r.ReadMapBegin()
		if err != nil {
			d.error(err)
		}
		v.Set(reflect.MakeMap(v.Type()))
		for i := 0; i < n; i++ {
			key := reflect.New(keyType).Elem()
			val := reflect.New(valueType).Elem()
			d.readValue(ktype, key)
			d.readValue(vtype, val)
			v.SetMapIndex(key, val)
		}
		if err := d.r.ReadMapEnd(); err != nil {
			d.error(err)
		}
	case TypeList:
		elemType := v.Type().Elem()
		et, n, err := d.r.ReadListBegin()
		if err != nil {
			d.error(err)
		}
		for i := 0; i < n; i++ {
			val := reflect.New(elemType)
			d.readValue(et, val.Elem())
			v.Set(reflect.Append(v, val.Elem()))
		}
		if err := d.r.ReadListEnd(); err != nil {
			d.error(err)
		}
	case TypeSet:
		if v.Type().Kind() == reflect.Slice {
			elemType := v.Type().Elem()
			et, n, err := d.r.ReadSetBegin()
			if err != nil {
				d.error(err)
			}
			for i := 0; i < n; i++ {
				val := reflect.New(elemType)
				d.readValue(et, val.Elem())
				v.Set(reflect.Append(v, val.Elem()))
			}
			if err := d.r.ReadSetEnd(); err != nil {
				d.error(err)
			}
		} else if v.Type().Kind() == reflect.Map {
			elemType := v.Type().Key()
			valueType := v.Type().Elem()
			et, n, err := d.r.ReadSetBegin()
			if err != nil {
				d.error(err)
			}
			v.Set(reflect.MakeMap(v.Type()))
			for i := 0; i < n; i++ {
				key := reflect.New(elemType).Elem()
				d.readValue(et, key)
				switch valueType.Kind() {
				case reflect.Bool:
					v.SetMapIndex(key, reflect.ValueOf(true))
				default:
					v.SetMapIndex(key, reflect.Zero(valueType))
				}
			}
			if err := d.r.ReadSetEnd(); err != nil {
				d.error(err)
			}
		} else {
			d.error(&UnsupportedTypeError{v.Type()})
		}
	default:
		d.error(&UnsupportedTypeError{v.Type()})
	}

	if err != nil {
		d.error(err)
	}

	return
}
