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

// This file has been generated from 'unfold_lookup_go.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"
)

func lookupGoTypeUnfolder(to interface{}) (unsafe.Pointer, ptrUnfolder) {
	switch ptr := to.(type) {
	case *interface{}:
		return unsafe.Pointer(ptr), newUnfolderIfc()
	case *[]interface{}:
		return unsafe.Pointer(ptr), newUnfolderArrIfc()
	case *map[string]interface{}:
		return unsafe.Pointer(ptr), newUnfolderMapIfc()

	case *bool:
		return unsafe.Pointer(ptr), newUnfolderBool()
	case *[]bool:
		return unsafe.Pointer(ptr), newUnfolderArrBool()
	case *map[string]bool:
		return unsafe.Pointer(ptr), newUnfolderMapBool()

	case *string:
		return unsafe.Pointer(ptr), newUnfolderString()
	case *[]string:
		return unsafe.Pointer(ptr), newUnfolderArrString()
	case *map[string]string:
		return unsafe.Pointer(ptr), newUnfolderMapString()

	case *uint:
		return unsafe.Pointer(ptr), newUnfolderUint()
	case *[]uint:
		return unsafe.Pointer(ptr), newUnfolderArrUint()
	case *map[string]uint:
		return unsafe.Pointer(ptr), newUnfolderMapUint()

	case *uint8:
		return unsafe.Pointer(ptr), newUnfolderUint8()
	case *[]uint8:
		return unsafe.Pointer(ptr), newUnfolderArrUint8()
	case *map[string]uint8:
		return unsafe.Pointer(ptr), newUnfolderMapUint8()

	case *uint16:
		return unsafe.Pointer(ptr), newUnfolderUint16()
	case *[]uint16:
		return unsafe.Pointer(ptr), newUnfolderArrUint16()
	case *map[string]uint16:
		return unsafe.Pointer(ptr), newUnfolderMapUint16()

	case *uint32:
		return unsafe.Pointer(ptr), newUnfolderUint32()
	case *[]uint32:
		return unsafe.Pointer(ptr), newUnfolderArrUint32()
	case *map[string]uint32:
		return unsafe.Pointer(ptr), newUnfolderMapUint32()

	case *uint64:
		return unsafe.Pointer(ptr), newUnfolderUint64()
	case *[]uint64:
		return unsafe.Pointer(ptr), newUnfolderArrUint64()
	case *map[string]uint64:
		return unsafe.Pointer(ptr), newUnfolderMapUint64()

	case *int:
		return unsafe.Pointer(ptr), newUnfolderInt()
	case *[]int:
		return unsafe.Pointer(ptr), newUnfolderArrInt()
	case *map[string]int:
		return unsafe.Pointer(ptr), newUnfolderMapInt()

	case *int8:
		return unsafe.Pointer(ptr), newUnfolderInt8()
	case *[]int8:
		return unsafe.Pointer(ptr), newUnfolderArrInt8()
	case *map[string]int8:
		return unsafe.Pointer(ptr), newUnfolderMapInt8()

	case *int16:
		return unsafe.Pointer(ptr), newUnfolderInt16()
	case *[]int16:
		return unsafe.Pointer(ptr), newUnfolderArrInt16()
	case *map[string]int16:
		return unsafe.Pointer(ptr), newUnfolderMapInt16()

	case *int32:
		return unsafe.Pointer(ptr), newUnfolderInt32()
	case *[]int32:
		return unsafe.Pointer(ptr), newUnfolderArrInt32()
	case *map[string]int32:
		return unsafe.Pointer(ptr), newUnfolderMapInt32()

	case *int64:
		return unsafe.Pointer(ptr), newUnfolderInt64()
	case *[]int64:
		return unsafe.Pointer(ptr), newUnfolderArrInt64()
	case *map[string]int64:
		return unsafe.Pointer(ptr), newUnfolderMapInt64()

	case *float32:
		return unsafe.Pointer(ptr), newUnfolderFloat32()
	case *[]float32:
		return unsafe.Pointer(ptr), newUnfolderArrFloat32()
	case *map[string]float32:
		return unsafe.Pointer(ptr), newUnfolderMapFloat32()

	case *float64:
		return unsafe.Pointer(ptr), newUnfolderFloat64()
	case *[]float64:
		return unsafe.Pointer(ptr), newUnfolderArrFloat64()
	case *map[string]float64:
		return unsafe.Pointer(ptr), newUnfolderMapFloat64()

	default:
		return nil, nil
	}
}

func lookupGoPtrUnfolder(t reflect.Type) ptrUnfolder {
	switch t.Kind() {
	case reflect.Interface:
		return newUnfolderIfc()

	case reflect.Bool:
		return newUnfolderBool()

	case reflect.String:
		return newUnfolderString()

	case reflect.Uint:
		return newUnfolderUint()

	case reflect.Uint8:
		return newUnfolderUint8()

	case reflect.Uint16:
		return newUnfolderUint16()

	case reflect.Uint32:
		return newUnfolderUint32()

	case reflect.Uint64:
		return newUnfolderUint64()

	case reflect.Int:
		return newUnfolderInt()

	case reflect.Int8:
		return newUnfolderInt8()

	case reflect.Int16:
		return newUnfolderInt16()

	case reflect.Int32:
		return newUnfolderInt32()

	case reflect.Int64:
		return newUnfolderInt64()

	case reflect.Float32:
		return newUnfolderFloat32()

	case reflect.Float64:
		return newUnfolderFloat64()

	case reflect.Slice:
		et := t.Elem()
		switch et.Kind() {
		case reflect.Interface:
			return newUnfolderArrIfc()

		case reflect.Bool:
			return newUnfolderArrBool()

		case reflect.String:
			return newUnfolderArrString()

		case reflect.Uint:
			return newUnfolderArrUint()

		case reflect.Uint8:
			return newUnfolderArrUint8()

		case reflect.Uint16:
			return newUnfolderArrUint16()

		case reflect.Uint32:
			return newUnfolderArrUint32()

		case reflect.Uint64:
			return newUnfolderArrUint64()

		case reflect.Int:
			return newUnfolderArrInt()

		case reflect.Int8:
			return newUnfolderArrInt8()

		case reflect.Int16:
			return newUnfolderArrInt16()

		case reflect.Int32:
			return newUnfolderArrInt32()

		case reflect.Int64:
			return newUnfolderArrInt64()

		case reflect.Float32:
			return newUnfolderArrFloat32()

		case reflect.Float64:
			return newUnfolderArrFloat64()

		}

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil
		}

		et := t.Elem()
		switch et.Kind() {
		case reflect.Interface:
			return newUnfolderMapIfc()

		case reflect.Bool:
			return newUnfolderMapBool()

		case reflect.String:
			return newUnfolderMapString()

		case reflect.Uint:
			return newUnfolderMapUint()

		case reflect.Uint8:
			return newUnfolderMapUint8()

		case reflect.Uint16:
			return newUnfolderMapUint16()

		case reflect.Uint32:
			return newUnfolderMapUint32()

		case reflect.Uint64:
			return newUnfolderMapUint64()

		case reflect.Int:
			return newUnfolderMapInt()

		case reflect.Int8:
			return newUnfolderMapInt8()

		case reflect.Int16:
			return newUnfolderMapInt16()

		case reflect.Int32:
			return newUnfolderMapInt32()

		case reflect.Int64:
			return newUnfolderMapInt64()

		case reflect.Float32:
			return newUnfolderMapFloat32()

		case reflect.Float64:
			return newUnfolderMapFloat64()

		}

	}

	return nil
}

func lookupReflUnfolder(ctx *unfoldCtx, t reflect.Type) (reflUnfolder, error) {
	if f := unfoldRegistry.find(t); f != nil {
		return f, nil
	}

	f, err := buildReflUnfolder(ctx, t)
	if err != nil {
		return nil, err
	}

	unfoldRegistry.set(t, f)
	return f, nil
}

func buildReflUnfolder(ctx *unfoldCtx, t reflect.Type) (reflUnfolder, error) {
	// we always expect a pointer
	bt := t.Elem()

	switch bt.Kind() {
	case reflect.Interface:
		return unfolderReflIfc, nil

	case reflect.Bool:
		return unfolderReflBool, nil

	case reflect.String:
		return unfolderReflString, nil

	case reflect.Uint:
		return unfolderReflUint, nil

	case reflect.Uint8:
		return unfolderReflUint8, nil

	case reflect.Uint16:
		return unfolderReflUint16, nil

	case reflect.Uint32:
		return unfolderReflUint32, nil

	case reflect.Uint64:
		return unfolderReflUint64, nil

	case reflect.Int:
		return unfolderReflInt, nil

	case reflect.Int8:
		return unfolderReflInt8, nil

	case reflect.Int16:
		return unfolderReflInt16, nil

	case reflect.Int32:
		return unfolderReflInt32, nil

	case reflect.Int64:
		return unfolderReflInt64, nil

	case reflect.Float32:
		return unfolderReflFloat32, nil

	case reflect.Float64:
		return unfolderReflFloat64, nil

	case reflect.Array:
		return nil, errTODO()

	case reflect.Ptr:
		unfolderElem, err := lookupReflUnfolder(ctx, bt)
		if err != nil {
			return nil, err
		}
		return newUnfolderReflPtr(unfolderElem), nil

	case reflect.Slice:
		et := bt.Elem()
		switch et.Kind() {
		case reflect.Interface:
			return unfolderReflArrIfc, nil

		case reflect.Bool:
			return unfolderReflArrBool, nil

		case reflect.String:
			return unfolderReflArrString, nil

		case reflect.Uint:
			return unfolderReflArrUint, nil

		case reflect.Uint8:
			return unfolderReflArrUint8, nil

		case reflect.Uint16:
			return unfolderReflArrUint16, nil

		case reflect.Uint32:
			return unfolderReflArrUint32, nil

		case reflect.Uint64:
			return unfolderReflArrUint64, nil

		case reflect.Int:
			return unfolderReflArrInt, nil

		case reflect.Int8:
			return unfolderReflArrInt8, nil

		case reflect.Int16:
			return unfolderReflArrInt16, nil

		case reflect.Int32:
			return unfolderReflArrInt32, nil

		case reflect.Int64:
			return unfolderReflArrInt64, nil

		case reflect.Float32:
			return unfolderReflArrFloat32, nil

		case reflect.Float64:
			return unfolderReflArrFloat64, nil

		}

		unfolderElem, err := lookupReflUnfolder(ctx, reflect.PtrTo(et))
		if err != nil {
			return nil, err
		}
		return newUnfolderReflSlice(unfolderElem), nil

	case reflect.Map:
		et := bt.Elem()
		switch et.Kind() {
		case reflect.Interface:
			return unfolderReflMapIfc, nil

		case reflect.Bool:
			return unfolderReflMapBool, nil

		case reflect.String:
			return unfolderReflMapString, nil

		case reflect.Uint:
			return unfolderReflMapUint, nil

		case reflect.Uint8:
			return unfolderReflMapUint8, nil

		case reflect.Uint16:
			return unfolderReflMapUint16, nil

		case reflect.Uint32:
			return unfolderReflMapUint32, nil

		case reflect.Uint64:
			return unfolderReflMapUint64, nil

		case reflect.Int:
			return unfolderReflMapInt, nil

		case reflect.Int8:
			return unfolderReflMapInt8, nil

		case reflect.Int16:
			return unfolderReflMapInt16, nil

		case reflect.Int32:
			return unfolderReflMapInt32, nil

		case reflect.Int64:
			return unfolderReflMapInt64, nil

		case reflect.Float32:
			return unfolderReflMapFloat32, nil

		case reflect.Float64:
			return unfolderReflMapFloat64, nil

		}

		unfolderElem, err := lookupReflUnfolder(ctx, reflect.PtrTo(et))
		if err != nil {
			return nil, err
		}
		return newUnfolderReflMap(unfolderElem), nil

	case reflect.Struct:
		return createUnfolderReflStruct(ctx, t)

	default:
		return nil, errTODO()
	}
}
