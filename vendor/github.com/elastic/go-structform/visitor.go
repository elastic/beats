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

package structform

// Visitor interface for accepting events. The Vistor defined the common Data
// Model all serializers should accept and all deserializers must implement.
type Visitor interface {
	ObjectVisitor
	ArrayVisitor
	ValueVisitor
}

// ExtVisitor interface defines the Extended Data Model. Usage and
// implementation of the Extended Data Model is optional, but can speed up
// operations.
type ExtVisitor interface {
	Visitor
	ArrayValueVisitor
	ObjectValueVisitor
	StringRefVisitor
}

//go:generate stringer -type=BaseType
type BaseType uint8

const (
	AnyType BaseType = iota
	ByteType
	StringType
	BoolType
	ZeroType
	IntType
	Int8Type
	Int16Type
	Int32Type
	Int64Type
	UintType
	Uint8Type
	Uint16Type
	Uint32Type
	Uint64Type
	Float32Type
	Float64Type
)

// ObjectVisitor iterates all fields in a dictionary like structure.
type ObjectVisitor interface {
	// OnObjectStart is called when a new object (key-value pairs) is going to be reported.
	// A call to OnKey or OnObjectFinished must follow directly.
	OnObjectStart(len int, baseType BaseType) error

	// OnArrayFinished indicates that there are no more key value pairs to report.
	OnObjectFinished() error

	// OnKey adds a new key to the object. A value must directly follow a call to OnKey.
	OnKey(s string) error
}

// ArrayVisitor defines the support for arrays/slices in the Data Model.
type ArrayVisitor interface {
	// OnArrayStart is called whan a new array is going to be reported.
	//
	// The `len` argument should report the length if known. `len` MUST BE -1 if
	// the length of the array is not known.  The BaseType should indicate the
	// element type of the array. If the element type is unknown or can be any
	// type (e.g. interface{}), AnyType must be used.
	OnArrayStart(len int, baseType BaseType) error

	// OnArrayFinished indicates that there are no more elements in the array.
	OnArrayFinished() error
}

// ValueVisitor defines the set of supported primitive types in the Data Model.
type ValueVisitor interface {
	// untyped nil value
	OnNil() error

	OnBool(b bool) error

	OnString(s string) error

	// int
	OnInt8(i int8) error
	OnInt16(i int16) error
	OnInt32(i int32) error
	OnInt64(i int64) error
	OnInt(i int) error

	// uint
	OnByte(b byte) error
	OnUint8(u uint8) error
	OnUint16(u uint16) error
	OnUint32(u uint32) error
	OnUint64(u uint64) error
	OnUint(u uint) error

	// float
	OnFloat32(f float32) error
	OnFloat64(f float64) error
}

// ArrayValueVisitor passes arrays with known type. Implementation
// of ArrayValueVisitor is optional.
type ArrayValueVisitor interface {
	OnBoolArray([]bool) error

	OnStringArray([]string) error

	// int
	OnInt8Array([]int8) error
	OnInt16Array([]int16) error
	OnInt32Array([]int32) error
	OnInt64Array([]int64) error
	OnIntArray([]int) error

	// uint
	OnBytes([]byte) error
	OnUint8Array([]uint8) error
	OnUint16Array([]uint16) error
	OnUint32Array([]uint32) error
	OnUint64Array([]uint64) error
	OnUintArray([]uint) error

	// float
	OnFloat32Array([]float32) error
	OnFloat64Array([]float64) error
}

// ObjectValueVisitor passes map[string]T. Implementation
// of ObjectValueVisitor is optional.
type ObjectValueVisitor interface {
	OnBoolObject(map[string]bool) error

	OnStringObject(map[string]string) error

	// int
	OnInt8Object(map[string]int8) error
	OnInt16Object(map[string]int16) error
	OnInt32Object(map[string]int32) error
	OnInt64Object(map[string]int64) error
	OnIntObject(map[string]int) error

	// uint
	OnUint8Object(map[string]uint8) error
	OnUint16Object(map[string]uint16) error
	OnUint32Object(map[string]uint32) error
	OnUint64Object(map[string]uint64) error
	OnUintObject(map[string]uint) error

	// float
	OnFloat32Object(map[string]float32) error
	OnFloat64Object(map[string]float64) error
}

// StringRefVisitor handles strings by reference into a byte string.
// The reference must be processed immediately, as the string passed
// might get modified after the callback returns.
type StringRefVisitor interface {
	OnStringRef(s []byte) error
	OnKeyRef(s []byte) error
}

type extVisitor struct {
	Visitor
	ObjectValueVisitor
	ArrayValueVisitor
	StringRefVisitor
}

// EnsureExtVisitor converts a Visitor into an ExtVisitor.  If v already
// implements ExtVisitor, it is directly implemented. If v only implements a
// subset of ExtVisitor, then conversions for the missing interfaces will be
// created.
func EnsureExtVisitor(v Visitor) ExtVisitor {
	if ev, ok := v.(ExtVisitor); ok {
		return ev
	}

	e := &extVisitor{
		Visitor: v,
	}
	if ov, ok := v.(ObjectValueVisitor); ok {
		e.ObjectValueVisitor = ov
	} else {
		e.ObjectValueVisitor = extObjVisitor{v}
	}
	if av, ok := v.(ArrayValueVisitor); ok {
		e.ArrayValueVisitor = av
	} else {
		e.ArrayValueVisitor = extArrVisitor{v}
	}
	e.StringRefVisitor = MakeStringRefVisitor(v)

	return e
}
