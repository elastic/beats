// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"reflect"
	"runtime"
)

// Encoder is the interface that allows types to serialize themselves to a Thrift stream
type Encoder interface {
	EncodeThrift(ProtocolWriter) error
}

type encoder struct {
	w ProtocolWriter
}

// EncodeStruct tries to serialize a struct to a Thrift stream
func EncodeStruct(w ProtocolWriter, v interface{}) (err error) {
	if en, ok := v.(Encoder); ok {
		return en.EncodeThrift(w)
	}

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	e := &encoder{w}
	vo := reflect.ValueOf(v)
	e.writeStruct(vo)
	return nil
}

func (e *encoder) error(err interface{}) {
	panic(err)
}

func (e *encoder) writeStruct(v reflect.Value) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		e.error(&UnsupportedValueError{Value: v, Str: "expected a struct"})
	}
	if err := e.w.WriteStructBegin(v.Type().Name()); err != nil {
		e.error(err)
	}
	for _, ef := range encodeFields(v.Type()).fields {
		structField := v.Type().Field(ef.i)
		fieldValue := v.Field(ef.i)

		if !ef.required && !ef.keepEmpty && isEmptyValue(fieldValue) {
			continue
		}

		if fieldValue.Kind() == reflect.Ptr {
			if ef.required && fieldValue.IsNil() {
				e.error(&MissingRequiredField{v.Type().Name(), structField.Name})
			}
		}

		ftype := ef.fieldType

		if err := e.w.WriteFieldBegin(structField.Name, ftype, int16(ef.id)); err != nil {
			e.error(err)
		}
		e.writeValue(fieldValue, ftype)
		if err := e.w.WriteFieldEnd(); err != nil {
			e.error(err)
		}
	}
	e.w.WriteFieldStop()
	if err := e.w.WriteStructEnd(); err != nil {
		e.error(err)
	}
}

func (e *encoder) writeValue(v reflect.Value, thriftType byte) {
	if en, ok := v.Interface().(Encoder); ok {
		if err := en.EncodeThrift(e.w); err != nil {
			e.error(err)
		}
		return
	}

	kind := v.Kind()
	if kind == reflect.Ptr || kind == reflect.Interface {
		v = v.Elem()
		kind = v.Kind()
	}

	var err error
	switch thriftType {
	case TypeBool:
		err = e.w.WriteBool(v.Bool())
	case TypeByte:
		if kind == reflect.Uint8 {
			err = e.w.WriteByte(byte(v.Uint()))
		} else {
			err = e.w.WriteByte(byte(v.Int()))
		}
	case TypeI16:
		err = e.w.WriteI16(int16(v.Int()))
	case TypeI32:
		if kind == reflect.Uint32 {
			err = e.w.WriteI32(int32(v.Uint()))
		} else {
			err = e.w.WriteI32(int32(v.Int()))
		}
	case TypeI64:
		if kind == reflect.Uint64 {
			err = e.w.WriteI64(int64(v.Uint()))
		} else {
			err = e.w.WriteI64(v.Int())
		}
	case TypeDouble:
		err = e.w.WriteDouble(v.Float())
	case TypeString:
		if kind == reflect.Slice {
			elemType := v.Type().Elem()
			if elemType.Kind() == reflect.Uint8 {
				err = e.w.WriteBytes(v.Bytes())
			} else {
				err = &UnsupportedValueError{Value: v, Str: "encoder expected a byte array"}
			}
		} else {
			err = e.w.WriteString(v.String())
		}
	case TypeStruct:
		e.writeStruct(v)
	case TypeMap:
		keyType := v.Type().Key()
		valueType := v.Type().Elem()
		keyThriftType := fieldType(keyType)
		valueThriftType := fieldType(valueType)
		if er := e.w.WriteMapBegin(keyThriftType, valueThriftType, v.Len()); er != nil {
			e.error(er)
		}
		for _, k := range v.MapKeys() {
			e.writeValue(k, keyThriftType)
			e.writeValue(v.MapIndex(k), valueThriftType)
		}
		err = e.w.WriteMapEnd()
	case TypeList:
		elemType := v.Type().Elem()
		if elemType.Kind() == reflect.Uint8 {
			err = e.w.WriteBytes(v.Bytes())
		} else {
			elemThriftType := fieldType(elemType)
			if er := e.w.WriteListBegin(elemThriftType, v.Len()); er != nil {
				e.error(er)
			}
			n := v.Len()
			for i := 0; i < n; i++ {
				e.writeValue(v.Index(i), elemThriftType)
			}
			err = e.w.WriteListEnd()
		}
	case TypeSet:
		if v.Type().Kind() == reflect.Slice {
			elemType := v.Type().Elem()
			elemThriftType := fieldType(elemType)
			if er := e.w.WriteSetBegin(elemThriftType, v.Len()); er != nil {
				e.error(er)
			}
			n := v.Len()
			for i := 0; i < n; i++ {
				e.writeValue(v.Index(i), elemThriftType)
			}
			err = e.w.WriteSetEnd()
		} else if v.Type().Kind() == reflect.Map {
			elemType := v.Type().Key()
			valueType := v.Type().Elem()
			elemThriftType := fieldType(elemType)
			if valueType.Kind() == reflect.Bool {
				n := 0
				for _, k := range v.MapKeys() {
					if v.MapIndex(k).Bool() {
						n++
					}
				}
				if er := e.w.WriteSetBegin(elemThriftType, n); er != nil {
					e.error(er)
				}
				for _, k := range v.MapKeys() {
					if v.MapIndex(k).Bool() {
						e.writeValue(k, elemThriftType)
					}
				}
			} else {
				if er := e.w.WriteSetBegin(elemThriftType, v.Len()); er != nil {
					e.error(er)
				}
				for _, k := range v.MapKeys() {
					e.writeValue(k, elemThriftType)
				}
			}
			err = e.w.WriteSetEnd()
		} else {
			e.error(&UnsupportedTypeError{v.Type()})
		}
	default:
		e.error(&UnsupportedTypeError{v.Type()})
	}

	if err != nil {
		e.error(err)
	}
}
