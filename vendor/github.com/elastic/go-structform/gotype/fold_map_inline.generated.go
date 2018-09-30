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

// This file has been generated from 'fold_map_inline.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"
)

var _mapInlineMapping = map[reflect.Type]reFoldFn{
	tBool:    foldMapInlineBool,
	tString:  foldMapInlineString,
	tUint:    foldMapInlineUint,
	tUint8:   foldMapInlineUint8,
	tUint16:  foldMapInlineUint16,
	tUint32:  foldMapInlineUint32,
	tUint64:  foldMapInlineUint64,
	tInt:     foldMapInlineInt,
	tInt8:    foldMapInlineInt8,
	tInt16:   foldMapInlineInt16,
	tInt32:   foldMapInlineInt32,
	tInt64:   foldMapInlineInt64,
	tFloat32: foldMapInlineFloat32,
	tFloat64: foldMapInlineFloat64,
}

func getMapInlineByPrimitiveElem(t reflect.Type) reFoldFn {
	if t == tInterface {
		return foldMapInlineInterface
	}
	return _mapInlineMapping[t]
}

func foldMapInlineInterface(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]interface{})(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = foldInterfaceValue(C, v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineBool(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]bool)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnBool(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineString(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]string)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnString(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint8(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint8)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint8(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint16(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint16)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint16(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint32(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint32)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint32(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineUint64(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]uint64)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnUint64(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt8(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int8)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt8(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt16(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int16)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt16(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt32(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int32)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt32(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineInt64(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]int64)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnInt64(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineFloat32(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]float32)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnFloat32(v); err != nil {
			return err
		}
	}
	return
}

func foldMapInlineFloat64(C *foldContext, v reflect.Value) (err error) {
	ptr := unsafe.Pointer(v.Pointer())
	if ptr == nil {
		return nil
	}

	m := *((*map[string]float64)(unsafe.Pointer(&ptr)))
	for k, v := range m {
		if err = C.OnKey(k); err != nil {
			return err
		}
		if err = C.OnFloat64(v); err != nil {
			return err
		}
	}
	return
}
