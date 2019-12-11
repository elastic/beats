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

package ucfg

import "reflect"

// Unpacker type used by Unpack to allow types to implement custom configuration
// unpacking.
type Unpacker interface {
	// Unpack is called if a setting of field has a type implementing Unpacker.
	//
	// The interface{} value passed to Unpack can be of type: bool, int64, uint64,
	// float64, string, []interface{} or map[string]interface{}.
	Unpack(interface{}) error
}

// BoolUnpacker interface specializes the Unpacker interface
// by casting values to bool when calling Unpack.
type BoolUnpacker interface {
	Unpack(b bool) error
}

// IntUnpacker interface specializes the Unpacker interface
// by casting values to int64 when calling Unpack.
type IntUnpacker interface {
	Unpack(i int64) error
}

// UintUnpacker interface specializes the Unpacker interface
// by casting values to uint64 when calling Unpack.
type UintUnpacker interface {
	Unpack(u uint64) error
}

// FloatUnpacker interface specializes the Unpacker interface
// by casting values to float64 when calling Unpack.
type FloatUnpacker interface {
	Unpack(f float64) error
}

// StringUnpacker interface specializes the Unpacker interface
// by casting values to string when calling Unpack.
type StringUnpacker interface {
	Unpack(s string) error
}

// ConfigUnpacker interface specializes the Unpacker interface
// by passing the the *Config object directly instead of
// transforming the *Config object into map[string]interface{}.
type ConfigUnpacker interface {
	Unpack(c *Config) error
}

var (
	// unpacker interface types
	tUnpacker       = reflect.TypeOf((*Unpacker)(nil)).Elem()
	tBoolUnpacker   = reflect.TypeOf((*BoolUnpacker)(nil)).Elem()
	tIntUnpacker    = reflect.TypeOf((*IntUnpacker)(nil)).Elem()
	tUintUnpacker   = reflect.TypeOf((*UintUnpacker)(nil)).Elem()
	tFloatUnpacker  = reflect.TypeOf((*FloatUnpacker)(nil)).Elem()
	tStringUnpacker = reflect.TypeOf((*StringUnpacker)(nil)).Elem()
	tConfigUnpacker = reflect.TypeOf((*ConfigUnpacker)(nil)).Elem()

	tUnpackers = [...]reflect.Type{
		tUnpacker,
		tBoolUnpacker,
		tIntUnpacker,
		tUintUnpacker,
		tFloatUnpacker,
		tStringUnpacker,
		tConfigUnpacker,
	}
)

// valueIsUnpacker checks if v implements the Unpacker interface.
// If there exists a pointer to v, the pointer to v is also tested.
func valueIsUnpacker(v reflect.Value) (reflect.Value, bool) {
	for {
		if implementsUnpacker(v.Type()) {
			return v, true
		}

		if !v.CanAddr() {
			break
		}
		v = v.Addr()
	}

	return reflect.Value{}, false
}

func typeIsUnpacker(t reflect.Type) (reflect.Value, bool) {
	if implementsUnpacker(t) {
		return reflect.New(t).Elem(), true
	}

	if implementsUnpacker(reflect.PtrTo(t)) {
		return reflect.New(t), true
	}

	return reflect.Value{}, false
}

func implementsUnpacker(t reflect.Type) bool {
	// ucfg.Config or structures that can be casted to ucfg.Config are not
	// Unpackers.
	if tConfig.ConvertibleTo(chaseTypePointers(t)) {
		return false
	}

	for _, tUnpack := range tUnpackers {
		if t.Implements(tUnpack) {
			return true
		}
	}

	if t.NumMethod() == 0 {
		return false
	}

	// test if object has 'Unpack' method
	method, ok := t.MethodByName("Unpack")
	if !ok {

		return false
	}

	// check method input and output parameters to match the ConfigUnpacker interface:
	// func (to *T) Unpack(cfg *TConfig) error
	//   with T being the method receiver (input paramter 0)
	//   and TConfig being the aliased config type to convert to (input parameter 1)
	paramCountCheck := method.Type.NumIn() == 2 && method.Type.NumOut() == 1
	if !paramCountCheck {
		return false
	}
	if !method.Type.Out(0).Implements(tError) {
		// return variable is not compatible to `error` type
		return false
	}

	// method receiver is known, check config parameters being compatible
	tIn := method.Type.In(1)
	return tConfig.ConvertibleTo(tIn) || tConfigPtr.ConvertibleTo(tIn)
}

func unpackWith(opts *options, v reflect.Value, with value) Error {
	ctx := with.Context()
	meta := with.meta()

	var err error
	value := v.Interface()
	switch u := value.(type) {
	case Unpacker:
		var reified interface{}
		if reified, err = with.reify(opts); err == nil {
			err = u.Unpack(reified)
		}

	case BoolUnpacker:
		var b bool
		if b, err = with.toBool(opts); err == nil {
			err = u.Unpack(b)
		}

	case IntUnpacker:
		var n int64
		if n, err = with.toInt(opts); err == nil {
			err = u.Unpack(n)
		}

	case UintUnpacker:
		var n uint64
		if n, err = with.toUint(opts); err == nil {
			err = u.Unpack(n)
		}

	case FloatUnpacker:
		var f float64
		if f, err = with.toFloat(opts); err == nil {
			err = u.Unpack(f)
		}

	case StringUnpacker:
		var s string
		if s, err = with.toString(opts); err == nil {
			err = u.Unpack(s)
		}

	case ConfigUnpacker:
		var c *Config
		if c, err = with.toConfig(opts); err == nil {
			err = u.Unpack(c)
		}

	default:
		var c *Config
		if c, err = with.toConfig(opts); err == nil {
			err = reflectUnpackWithConfig(v, c)
		}

	}

	if err != nil {
		return raisePathErr(err, meta, "", ctx.path("."))
	}
	return nil
}

func reflectUnpackWithConfig(v reflect.Value, c *Config) error {
	method, _ := v.Type().MethodByName("Unpack")
	tIn := method.Type.In(1)

	var rc reflect.Value
	switch {
	case tConfig.ConvertibleTo(tIn):
		rc = reflect.ValueOf(*c)
	case tConfigPtr.ConvertibleTo(tIn):
		rc = reflect.ValueOf(c)
	}

	results := method.Func.Call([]reflect.Value{v, rc.Convert(tIn)})
	ifc := results[0].Convert(tError).Interface()
	if ifc == nil {
		return nil
	}
	return ifc.(error)
}
