package fld

import (
	"math"
	"time"
)

type Field struct {
	Key          string
	Value        Value
	Standardized bool
}

type Value struct {
	Primitive uint64
	String    string
	Ifc       interface{}

	Reporter Reporter
}

type Reporter interface {
	Type() Type
	Ifc(*Value, func(interface{}))
}

type Type uint8

const (
	IfcType Type = iota
	IntType
	Int64Type
	Uint64Type
	Float64Type
	DurationType
	TimestampType
	StringType
)

func (v *Value) Interface() (ifc interface{}) {
	v.Reporter.Ifc(v, func(tmp interface{}) {
		ifc = tmp
	})
	return ifc
}

func userField(k string, v Value) Field {
	return Field{Key: k, Value: v}
}

func Int(key string, i int) Field { return userField(key, ValInt(i)) }
func ValInt(i int) Value          { return Value{Primitive: uint64(i), Reporter: _intReporter} }

type intReporter struct{}

var _intReporter Reporter = intReporter{}

func (intReporter) Type() Type {
	return IntType
}

func (intReporter) Ifc(v *Value, fn func(interface{})) {
	fn(int(v.Primitive))
}

func Int64(key string, i int64) Field { return userField(key, ValInt64(i)) }
func ValInt64(i int64) Value          { return Value{Primitive: uint64(i), Reporter: _int64Reporter} }

type int64Reporter struct{}

var _int64Reporter Reporter = int64Reporter{}

func (int64Reporter) Type() Type { return Int64Type }
func (int64Reporter) Ifc(v *Value, fn func(v interface{})) {
	fn(int64(v.Primitive))
}

func Uint(key string, i uint) Field { return userField(key, ValUint(i)) }
func ValUint(i uint) Value          { return ValUint64(uint64(i)) }

func Uint64(key string, i uint64) Field { return userField(key, ValUint64(i)) }
func ValUint64(u uint64) Value          { return Value{Primitive: u, Reporter: _uint64Reporter} }

type uint64Reporter struct{}

var _uint64Reporter Reporter = uint64Reporter{}

func (uint64Reporter) Type() Type { return Int64Type }
func (uint64Reporter) Ifc(v *Value, fn func(v interface{})) {
	fn(uint64(v.Primitive))
}

func Float(key string, f float64) Field { return userField(key, ValFloat(f)) }
func ValFloat(f float64) Value {
	return Value{Primitive: math.Float64bits(f), Reporter: _float64Reporter}
}

type float64Reporter struct{}

var _float64Reporter Reporter = float64Reporter{}

func (float64Reporter) Type() Type { return Float64Type }
func (float64Reporter) Ifc(v *Value, fn func(v interface{})) {
	fn(math.Float64frombits(v.Primitive))
}

func String(key, str string) Field                  { return userField(key, ValString(str)) }
func ValString(str string) Value                    { return Value{String: str, Reporter: _strReporter} }
func reportString(v *Value, fn func(v interface{})) { fn(v.String) }

type strReporter struct{}

var _strReporter Reporter = strReporter{}

func (strReporter) Type() Type { return StringType }
func (strReporter) Ifc(v *Value, fn func(v interface{})) {
	fn(v.String)
}

func Duration(key string, dur time.Duration) Field { return userField(key, ValDuration(dur)) }
func ValDuration(dur time.Duration) Value {
	return Value{Primitive: uint64(dur), Reporter: _durReporter}
}

type durReporter struct{}

var _durReporter Reporter = durReporter{}

func (durReporter) Type() Type { return DurationType }
func (durReporter) Ifc(v *Value, fn func(v interface{})) {
	fn(time.Duration(v.Primitive))
}

func Timestamp(key string, ts time.Time) Field { return userField(key, ValTime(ts)) }
func ValTime(ts time.Time) Value {
	return Value{Ifc: ts, Reporter: _timeReporter}
}

type timeReporter struct{}

var _timeReporter Reporter = timeReporter{}

func (timeReporter) Type() Type { return TimestampType }
func (timeReporter) Ifc(v *Value, fn func(v interface{})) {
	fn(v.Ifc)
}

func Any(key string, ifc interface{}) Field {
	// TODO: use type switch + reflection to select concrete Field
	return userField(key, ValAny(ifc))
}
func ValAny(ifc interface{}) Value { return Value{Ifc: ifc, Reporter: _anyReporter} }
func reportAny(v *Value, fn func(v interface{})) {
	fn(v.Ifc)
}

type anyReporter struct{}

var _anyReporter Reporter = anyReporter{}

func (anyReporter) Type() Type { return IfcType }
func (anyReporter) Ifc(v *Value, fn func(v interface{})) {
	fn(v.Ifc)
}
