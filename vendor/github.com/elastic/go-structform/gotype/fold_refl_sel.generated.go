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

// This file has been generated from 'fold_refl_sel.yml', do not edit
package gotype

import "reflect"

var _reflPrimitivesMapping = map[reflect.Type]reFoldFn{

	tBool:                         reFoldBool,
	reflect.SliceOf(tBool):        reFoldArrBool,
	reflect.MapOf(tString, tBool): reFoldMapBool,

	tString:                         reFoldString,
	reflect.SliceOf(tString):        reFoldArrString,
	reflect.MapOf(tString, tString): reFoldMapString,

	tUint:                         reFoldUint,
	reflect.SliceOf(tUint):        reFoldArrUint,
	reflect.MapOf(tString, tUint): reFoldMapUint,

	tUint8:                         reFoldUint8,
	reflect.SliceOf(tUint8):        reFoldArrUint8,
	reflect.MapOf(tString, tUint8): reFoldMapUint8,

	tUint16:                         reFoldUint16,
	reflect.SliceOf(tUint16):        reFoldArrUint16,
	reflect.MapOf(tString, tUint16): reFoldMapUint16,

	tUint32:                         reFoldUint32,
	reflect.SliceOf(tUint32):        reFoldArrUint32,
	reflect.MapOf(tString, tUint32): reFoldMapUint32,

	tUint64:                         reFoldUint64,
	reflect.SliceOf(tUint64):        reFoldArrUint64,
	reflect.MapOf(tString, tUint64): reFoldMapUint64,

	tInt:                         reFoldInt,
	reflect.SliceOf(tInt):        reFoldArrInt,
	reflect.MapOf(tString, tInt): reFoldMapInt,

	tInt8:                         reFoldInt8,
	reflect.SliceOf(tInt8):        reFoldArrInt8,
	reflect.MapOf(tString, tInt8): reFoldMapInt8,

	tInt16:                         reFoldInt16,
	reflect.SliceOf(tInt16):        reFoldArrInt16,
	reflect.MapOf(tString, tInt16): reFoldMapInt16,

	tInt32:                         reFoldInt32,
	reflect.SliceOf(tInt32):        reFoldArrInt32,
	reflect.MapOf(tString, tInt32): reFoldMapInt32,

	tInt64:                         reFoldInt64,
	reflect.SliceOf(tInt64):        reFoldArrInt64,
	reflect.MapOf(tString, tInt64): reFoldMapInt64,

	tFloat32:                         reFoldFloat32,
	reflect.SliceOf(tFloat32):        reFoldArrFloat32,
	reflect.MapOf(tString, tFloat32): reFoldMapFloat32,

	tFloat64:                         reFoldFloat64,
	reflect.SliceOf(tFloat64):        reFoldArrFloat64,
	reflect.MapOf(tString, tFloat64): reFoldMapFloat64,
}

func getReflectFoldPrimitive(t reflect.Type) reFoldFn {
	return _reflPrimitivesMapping[t]
}

func getReflectFoldPrimitiveKind(t reflect.Type) (reFoldFn, error) {
	switch t.Kind() {

	case reflect.Bool:
		return reFoldBool, nil

	case reflect.String:
		return reFoldString, nil

	case reflect.Uint:
		return reFoldUint, nil

	case reflect.Uint8:
		return reFoldUint8, nil

	case reflect.Uint16:
		return reFoldUint16, nil

	case reflect.Uint32:
		return reFoldUint32, nil

	case reflect.Uint64:
		return reFoldUint64, nil

	case reflect.Int:
		return reFoldInt, nil

	case reflect.Int8:
		return reFoldInt8, nil

	case reflect.Int16:
		return reFoldInt16, nil

	case reflect.Int32:
		return reFoldInt32, nil

	case reflect.Int64:
		return reFoldInt64, nil

	case reflect.Float32:
		return reFoldFloat32, nil

	case reflect.Float64:
		return reFoldFloat64, nil

	default:
		return nil, errUnsupported
	}
}
