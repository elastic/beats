// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package diag

import (
	"math"
	"time"
)

// Value represents a reportable value to be stored in a Field.
// The Value struct provides a slot for primitive values that require only
// 64bits, a string, or an arbitrary interface. The interpretation of the slots is up to the Reporter.
type Value struct {
	Primitive uint64
	String    string
	Ifc       interface{}

	Reporter Reporter
}

// Reporter defines the type and supports unpacking, querying the decoded Value.
type Reporter interface {
	Type() Type

	// Ifc decodes the Value and reports the decoded value to as `interface{}`
	// to the provided callback.
	Ifc(*Value, func(interface{}))
}

// Type represents the possible types a Value can have.
type Type uint8

const (
	IfcType Type = iota
	BoolType
	IntType
	Int64Type
	Uint64Type
	Float64Type
	DurationType
	TimestampType
	StringType
)

// Interface decodes and returns the value stored in Value.
func (v *Value) Interface() (ifc interface{}) {
	v.Reporter.Ifc(v, func(tmp interface{}) {
		ifc = tmp
	})
	return ifc
}

// ValBool creates a new Value representing a bool.
func ValBool(b bool) Value {
	var x uint64
	if b {
		x = 1
	}
	return Value{Primitive: x, Reporter: _boolReporter}
}

type boolReporter struct{}

var _boolReporter Reporter = boolReporter{}

func (boolReporter) Type() Type                         { return BoolType }
func (boolReporter) Ifc(v *Value, fn func(interface{})) { fn(bool(v.Primitive != 0)) }

// ValInt create a new Value representing an int.
func ValInt(i int) Value { return Value{Primitive: uint64(i), Reporter: _intReporter} }

type intReporter struct{}

var _intReporter Reporter = intReporter{}

func (intReporter) Type() Type                         { return IntType }
func (intReporter) Ifc(v *Value, fn func(interface{})) { fn(int(v.Primitive)) }

// ValInt64 creates a new Value representing an int64.
func ValInt64(i int64) Value { return Value{Primitive: uint64(i), Reporter: _int64Reporter} }

type int64Reporter struct{}

var _int64Reporter Reporter = int64Reporter{}

func (int64Reporter) Type() Type                           { return Int64Type }
func (int64Reporter) Ifc(v *Value, fn func(v interface{})) { fn(int64(v.Primitive)) }

// ValUint creates a new Value representing an uint.
func ValUint(i uint) Value { return ValUint64(uint64(i)) }

// ValUint64 creates a new Value representing an uint64.
func ValUint64(u uint64) Value { return Value{Primitive: u, Reporter: _uint64Reporter} }

type uint64Reporter struct{}

var _uint64Reporter Reporter = uint64Reporter{}

func (uint64Reporter) Type() Type                           { return Int64Type }
func (uint64Reporter) Ifc(v *Value, fn func(v interface{})) { fn(uint64(v.Primitive)) }

// ValFloat creates a new Value representing a float.
func ValFloat(f float64) Value {
	return Value{Primitive: math.Float64bits(f), Reporter: _float64Reporter}
}

type float64Reporter struct{}

var _float64Reporter Reporter = float64Reporter{}

func (float64Reporter) Type() Type                           { return Float64Type }
func (float64Reporter) Ifc(v *Value, fn func(v interface{})) { fn(math.Float64frombits(v.Primitive)) }

// ValString creates a new Value representing a string.
func ValString(str string) Value { return Value{String: str, Reporter: _strReporter} }

type strReporter struct{}

var _strReporter Reporter = strReporter{}

func (strReporter) Type() Type                           { return StringType }
func (strReporter) Ifc(v *Value, fn func(v interface{})) { fn(v.String) }

// ValDuration creates a new Value representing a duration.
func ValDuration(dur time.Duration) Value {
	return Value{Primitive: uint64(dur), Reporter: _durReporter}
}

type durReporter struct{}

var _durReporter Reporter = durReporter{}

func (durReporter) Type() Type                           { return DurationType }
func (durReporter) Ifc(v *Value, fn func(v interface{})) { fn(time.Duration(v.Primitive)) }

// ValTime creates a new Value representing a timestamp.
func ValTime(ts time.Time) Value {
	return Value{Ifc: ts, Reporter: _timeReporter}
}

type timeReporter struct{}

var _timeReporter Reporter = timeReporter{}

func (timeReporter) Type() Type { return TimestampType }
func (timeReporter) Ifc(v *Value, fn func(v interface{})) {
	fn(v.Ifc)
}

// ValAny creates a new Value representing any value as interface.
func ValAny(ifc interface{}) Value { return Value{Ifc: ifc, Reporter: _anyReporter} }
func reportAny(v *Value, fn func(v interface{})) {
	fn(v.Ifc)
}

type anyReporter struct{}

var _anyReporter Reporter = anyReporter{}

func (anyReporter) Type() Type                           { return IfcType }
func (anyReporter) Ifc(v *Value, fn func(v interface{})) { fn(v.Ifc) }
