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

package unsafe

import (
	"reflect"
	"runtime"
	"unsafe"
)

type emptyInterface struct {
	typ  unsafe.Pointer
	word unsafe.Pointer
}

func Str2Bytes(s string) (b []byte) {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	runtime.KeepAlive(s)
	return
}

func Bytes2Str(b []byte) (s string) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sh.Data = bh.Data
	sh.Len = bh.Len
	runtime.KeepAlive(b)
	return
}

// IfcValuePtr extracts the underlying values pointer from an empty interface{}
// value.
// Note: this might beome more unsafe in future go-versions,
// if primitive values < pointer size will be stored by value in the
// `interface{}` type.
func IfcValuePtr(v interface{}) unsafe.Pointer {
	ifc := (*emptyInterface)(unsafe.Pointer(&v))
	return ifc.word
}

// ReflValuePtr extracts the pointer value from a reflect.Value instance.
// With reflect.Value basically being similar to `interface{}` augmented with additional
// flags to execute checks, we map the value into an empty interface value (no methods)
// and extract the actual values pointer.
// Note: this might beome more unsafe in future go-versions,
// if primitive values < pointer size will be stored by value in the
// `interface{}` type.
func ReflValuePtr(v reflect.Value) unsafe.Pointer {
	ifc := (*emptyInterface)(unsafe.Pointer(&v))
	return ifc.word
}

// Returns a newly (allocated on heap) function pointer. The unsafe.Pointer returned
// can be used to cast a function type into a function with other(compatible)
// type (e.g. passing pointers only).
func UnsafeFnPtr(fn interface{}) unsafe.Pointer {
	var v reflect.Value
	if tmp, ok := fn.(reflect.Value); ok {
		v = tmp
	} else {
		v = reflect.ValueOf(fn)
	}

	tmp := reflect.New(v.Type())
	tmp.Elem().Set(v)
	return unsafe.Pointer(tmp.Pointer())
}
