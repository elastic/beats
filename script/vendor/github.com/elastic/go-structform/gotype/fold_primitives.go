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

import "reflect"

func foldNil(C *foldContext, v interface{}) error     { return C.OnNil() }
func foldBool(C *foldContext, v interface{}) error    { return C.OnBool(v.(bool)) }
func foldInt8(C *foldContext, v interface{}) error    { return C.OnInt8(v.(int8)) }
func foldInt16(C *foldContext, v interface{}) error   { return C.OnInt16(v.(int16)) }
func foldInt32(C *foldContext, v interface{}) error   { return C.OnInt32(v.(int32)) }
func foldInt64(C *foldContext, v interface{}) error   { return C.OnInt64(v.(int64)) }
func foldInt(C *foldContext, v interface{}) error     { return C.OnInt64(int64(v.(int))) }
func foldByte(C *foldContext, v interface{}) error    { return C.OnByte(v.(byte)) }
func foldUint8(C *foldContext, v interface{}) error   { return C.OnUint8(v.(uint8)) }
func foldUint16(C *foldContext, v interface{}) error  { return C.OnUint16(v.(uint16)) }
func foldUint32(C *foldContext, v interface{}) error  { return C.OnUint32(v.(uint32)) }
func foldUint64(C *foldContext, v interface{}) error  { return C.OnUint64(v.(uint64)) }
func foldUint(C *foldContext, v interface{}) error    { return C.OnUint(v.(uint)) }
func foldFloat32(C *foldContext, v interface{}) error { return C.OnFloat32(v.(float32)) }
func foldFloat64(C *foldContext, v interface{}) error { return C.OnFloat64(v.(float64)) }
func foldString(C *foldContext, v interface{}) error  { return C.OnString(v.(string)) }

func reFoldNil(C *foldContext, v reflect.Value) error    { return C.OnNil() }
func reFoldBool(C *foldContext, v reflect.Value) error   { return C.OnBool(v.Bool()) }
func reFoldInt8(C *foldContext, v reflect.Value) error   { return C.OnInt8(int8(v.Int())) }
func reFoldInt16(C *foldContext, v reflect.Value) error  { return C.OnInt16(int16(v.Int())) }
func reFoldInt32(C *foldContext, v reflect.Value) error  { return C.OnInt32(int32(v.Int())) }
func reFoldInt64(C *foldContext, v reflect.Value) error  { return C.OnInt64(v.Int()) }
func reFoldInt(C *foldContext, v reflect.Value) error    { return C.OnInt64(int64(int(v.Int()))) }
func reFoldUint8(C *foldContext, v reflect.Value) error  { return C.OnUint8(uint8(v.Uint())) }
func reFoldUint16(C *foldContext, v reflect.Value) error { return C.OnUint16(uint16(v.Uint())) }
func reFoldUint32(C *foldContext, v reflect.Value) error { return C.OnUint32(uint32(v.Uint())) }
func reFoldUint64(C *foldContext, v reflect.Value) error { return C.OnUint64(v.Uint()) }
func reFoldUint(C *foldContext, v reflect.Value) error   { return C.OnUint(uint(v.Uint())) }
func reFoldFloat32(C *foldContext, v reflect.Value) error {
	return C.OnFloat32(float32(v.Float()))
}
func reFoldFloat64(C *foldContext, v reflect.Value) error { return C.OnFloat64(v.Float()) }
func reFoldString(C *foldContext, v reflect.Value) error  { return C.OnString(v.String()) }

func reFoldFolderIfc(C *foldContext, v reflect.Value) error {
	return v.Interface().(Folder).Fold(C.visitor)
}
