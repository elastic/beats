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
	reFoldMapBool    = liftFold(map[string]bool(nil), foldMapBool)
	reFoldMapInt     = liftFold(map[string]int(nil), foldMapInt)
	reFoldMapInt8    = liftFold(map[string]int8(nil), foldMapInt8)
	reFoldMapInt16   = liftFold(map[string]int16(nil), foldMapInt16)
	reFoldMapInt32   = liftFold(map[string]int32(nil), foldMapInt32)
	reFoldMapInt64   = liftFold(map[string]int64(nil), foldMapInt64)
	reFoldMapUint    = liftFold(map[string]uint(nil), foldMapUint)
	reFoldMapUint8   = liftFold(map[string]uint8(nil), foldMapUint8)
	reFoldMapUint16  = liftFold(map[string]uint16(nil), foldMapUint16)
	reFoldMapUint32  = liftFold(map[string]uint32(nil), foldMapUint32)
	reFoldMapUint64  = liftFold(map[string]uint64(nil), foldMapUint64)
	reFoldMapFloat32 = liftFold(map[string]float32(nil), foldMapFloat32)
	reFoldMapFloat64 = liftFold(map[string]float64(nil), foldMapFloat64)
	reFoldMapString  = liftFold(map[string]string(nil), foldMapString)
)

var tMapAny = reflect.TypeOf(map[string]interface{}(nil))

func foldMapInterface(C *foldContext, v interface{}) error {
	m := v.(map[string]interface{})
	if err := C.OnObjectStart(len(m), structform.AnyType); err != nil {
		return err
	}

	for k, v := range m {
		if err := C.OnKey(k); err != nil {
			return err
		}
		if err := foldInterfaceValue(C, v); err != nil {
			return err
		}
	}
	return C.OnObjectFinished()
}

func foldMapBool(C *foldContext, v interface{}) error {
	return C.visitor.OnBoolObject(v.(map[string]bool))
}

func foldMapString(C *foldContext, v interface{}) error {
	return C.visitor.OnStringObject(v.(map[string]string))
}

func foldMapInt8(C *foldContext, v interface{}) error {
	return C.visitor.OnInt8Object(v.(map[string]int8))
}

func foldMapInt16(C *foldContext, v interface{}) error {
	return C.visitor.OnInt16Object(v.(map[string]int16))
}

func foldMapInt32(C *foldContext, v interface{}) error {
	return C.visitor.OnInt32Object(v.(map[string]int32))
}

func foldMapInt64(C *foldContext, v interface{}) error {
	return C.visitor.OnInt64Object(v.(map[string]int64))
}

func foldMapInt(C *foldContext, v interface{}) error {
	return C.visitor.OnIntObject(v.(map[string]int))
}

func foldMapUint8(C *foldContext, v interface{}) error {
	return C.visitor.OnUint8Object(v.(map[string]uint8))
}

func foldMapUint16(C *foldContext, v interface{}) error {
	return C.visitor.OnUint16Object(v.(map[string]uint16))
}

func foldMapUint32(C *foldContext, v interface{}) error {
	return C.visitor.OnUint32Object(v.(map[string]uint32))
}

func foldMapUint64(C *foldContext, v interface{}) error {
	return C.visitor.OnUint64Object(v.(map[string]uint64))
}

func foldMapUint(C *foldContext, v interface{}) error {
	return C.visitor.OnUintObject(v.(map[string]uint))
}

func foldMapFloat32(C *foldContext, v interface{}) error {
	return C.visitor.OnFloat32Object(v.(map[string]float32))
}

func foldMapFloat64(C *foldContext, v interface{}) error {
	return C.visitor.OnFloat64Object(v.(map[string]float64))
}
