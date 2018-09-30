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

	structform "github.com/elastic/go-structform"
)

var (
	reFoldArrBool    = liftFold([]bool(nil), foldArrBool)
	reFoldArrInt     = liftFold([]int(nil), foldArrInt)
	reFoldArrInt8    = liftFold([]int8(nil), foldArrInt8)
	reFoldArrInt16   = liftFold([]int16(nil), foldArrInt16)
	reFoldArrInt32   = liftFold([]int32(nil), foldArrInt32)
	reFoldArrInt64   = liftFold([]int64(nil), foldArrInt64)
	reFoldArrUint    = liftFold([]uint(nil), foldArrUint)
	reFoldArrUint8   = liftFold([]uint8(nil), foldArrUint8)
	reFoldArrUint16  = liftFold([]uint16(nil), foldArrUint16)
	reFoldArrUint32  = liftFold([]uint32(nil), foldArrUint32)
	reFoldArrUint64  = liftFold([]uint64(nil), foldArrUint64)
	reFoldArrFloat32 = liftFold([]float32(nil), foldArrFloat32)
	reFoldArrFloat64 = liftFold([]float64(nil), foldArrFloat64)
	reFoldArrString  = liftFold([]string(nil), foldArrString)
)

var tArrayAny = reflect.TypeOf([]interface{}(nil))

func foldArrInterface(C *foldContext, v interface{}) error {
	a := v.([]interface{})
	if err := C.OnArrayStart(len(a), structform.AnyType); err != nil {
		return err
	}

	for _, v := range a {
		if err := foldInterfaceValue(C, v); err != nil {
			return err
		}
	}
	return C.OnArrayFinished()
}

func foldArrBool(C *foldContext, v interface{}) error   { return C.visitor.OnBoolArray(v.([]bool)) }
func foldArrString(C *foldContext, v interface{}) error { return C.visitor.OnStringArray(v.([]string)) }
func foldArrInt8(C *foldContext, v interface{}) error   { return C.visitor.OnInt8Array(v.([]int8)) }
func foldArrInt16(C *foldContext, v interface{}) error  { return C.visitor.OnInt16Array(v.([]int16)) }
func foldArrInt32(C *foldContext, v interface{}) error  { return C.visitor.OnInt32Array(v.([]int32)) }
func foldArrInt64(C *foldContext, v interface{}) error  { return C.visitor.OnInt64Array(v.([]int64)) }
func foldArrInt(C *foldContext, v interface{}) error    { return C.visitor.OnIntArray(v.([]int)) }
func foldBytes(C *foldContext, v interface{}) error     { return C.visitor.OnBytes(v.([]byte)) }
func foldArrUint8(C *foldContext, v interface{}) error  { return C.visitor.OnUint8Array(v.([]uint8)) }
func foldArrUint16(C *foldContext, v interface{}) error { return C.visitor.OnUint16Array(v.([]uint16)) }
func foldArrUint32(C *foldContext, v interface{}) error { return C.visitor.OnUint32Array(v.([]uint32)) }
func foldArrUint64(C *foldContext, v interface{}) error { return C.visitor.OnUint64Array(v.([]uint64)) }
func foldArrUint(C *foldContext, v interface{}) error   { return C.visitor.OnUintArray(v.([]uint)) }
func foldArrFloat32(C *foldContext, v interface{}) error {
	return C.visitor.OnFloat32Array(v.([]float32))
}
func foldArrFloat64(C *foldContext, v interface{}) error {
	return C.visitor.OnFloat64Array(v.([]float64))
}
