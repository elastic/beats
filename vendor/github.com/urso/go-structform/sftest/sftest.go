package sftest

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/json-iterator/go/assert"
	structform "github.com/urso/go-structform"
)

type Recording []Record

type Record interface {
	Replay(v structform.ExtVisitor) error
}

type NilRec struct{}
type BoolRec struct{ Value bool }
type StringRec struct{ Value string }
type IntRec struct{ Value int }
type Int8Rec struct{ Value int8 }
type Int16Rec struct{ Value int16 }
type Int32Rec struct{ Value int32 }
type Int64Rec struct{ Value int64 }
type UintRec struct{ Value uint }
type ByteRec struct{ Value byte }
type Uint8Rec struct{ Value uint8 }
type Uint16Rec struct{ Value uint16 }
type Uint32Rec struct{ Value uint32 }
type Uint64Rec struct{ Value uint64 }
type Float32Rec struct{ Value float32 }
type Float64Rec struct{ Value float64 }

// extended (yet) non-recordible records
type Int8ArrRec struct{ Value []int8 }
type StringArrRec struct{ Value []string }
type StringObjRec struct{ Value map[string]string }
type UintObjRec struct{ Value map[string]uint }

type ObjectStartRec struct {
	Len int
	T   structform.BaseType
}
type ObjectFinishRec struct{}
type ObjectKeyRec struct{ Value string }

type ArrayStartRec struct {
	Len int
	T   structform.BaseType
}
type ArrayFinishRec struct{}

func (NilRec) Replay(vs structform.ExtVisitor) error            { return vs.OnNil() }
func (r BoolRec) Replay(vs structform.ExtVisitor) error         { return vs.OnBool(r.Value) }
func (r StringRec) Replay(vs structform.ExtVisitor) error       { return vs.OnString(r.Value) }
func (r IntRec) Replay(vs structform.ExtVisitor) error          { return vs.OnInt(r.Value) }
func (r Int8Rec) Replay(vs structform.ExtVisitor) error         { return vs.OnInt8(r.Value) }
func (r Int16Rec) Replay(vs structform.ExtVisitor) error        { return vs.OnInt16(r.Value) }
func (r Int32Rec) Replay(vs structform.ExtVisitor) error        { return vs.OnInt32(r.Value) }
func (r Int64Rec) Replay(vs structform.ExtVisitor) error        { return vs.OnInt64(r.Value) }
func (r UintRec) Replay(vs structform.ExtVisitor) error         { return vs.OnUint(r.Value) }
func (r ByteRec) Replay(vs structform.ExtVisitor) error         { return vs.OnByte(r.Value) }
func (r Uint8Rec) Replay(vs structform.ExtVisitor) error        { return vs.OnUint8(r.Value) }
func (r Uint16Rec) Replay(vs structform.ExtVisitor) error       { return vs.OnUint16(r.Value) }
func (r Uint32Rec) Replay(vs structform.ExtVisitor) error       { return vs.OnUint32(r.Value) }
func (r Uint64Rec) Replay(vs structform.ExtVisitor) error       { return vs.OnUint64(r.Value) }
func (r Float32Rec) Replay(vs structform.ExtVisitor) error      { return vs.OnFloat32(r.Value) }
func (r Float64Rec) Replay(vs structform.ExtVisitor) error      { return vs.OnFloat64(r.Value) }
func (r ObjectStartRec) Replay(vs structform.ExtVisitor) error  { return vs.OnObjectStart(r.Len, r.T) }
func (r ObjectFinishRec) Replay(vs structform.ExtVisitor) error { return vs.OnObjectFinished() }
func (r ObjectKeyRec) Replay(vs structform.ExtVisitor) error    { return vs.OnKey(r.Value) }
func (r ArrayStartRec) Replay(vs structform.ExtVisitor) error   { return vs.OnArrayStart(r.Len, r.T) }
func (r ArrayFinishRec) Replay(vs structform.ExtVisitor) error  { return vs.OnArrayFinished() }

func (r Int8ArrRec) Replay(vs structform.ExtVisitor) error   { return vs.OnInt8Array(r.Value) }
func (r StringArrRec) Replay(vs structform.ExtVisitor) error { return vs.OnStringArray(r.Value) }
func (r UintObjRec) Replay(vs structform.ExtVisitor) error   { return vs.OnUintObject(r.Value) }
func (r StringObjRec) Replay(vs structform.ExtVisitor) error { return vs.OnStringObject(r.Value) }

func (rec *Recording) Replay(vs structform.Visitor) error {
	evs := structform.EnsureExtVisitor(vs)
	for _, r := range *rec {
		if err := r.Replay(evs); err != nil {
			return err
		}
	}
	return nil
}

func (r *Recording) add(v Record) error {
	*r = append(*r, v)
	return nil
}

func (r *Recording) OnNil() error              { return r.add(NilRec{}) }
func (r *Recording) OnBool(b bool) error       { return r.add(BoolRec{b}) }
func (r *Recording) OnString(s string) error   { return r.add(StringRec{s}) }
func (r *Recording) OnInt8(i int8) error       { return r.add(Int8Rec{i}) }
func (r *Recording) OnInt16(i int16) error     { return r.add(Int16Rec{i}) }
func (r *Recording) OnInt32(i int32) error     { return r.add(Int32Rec{i}) }
func (r *Recording) OnInt64(i int64) error     { return r.add(Int64Rec{i}) }
func (r *Recording) OnInt(i int) error         { return r.add(IntRec{i}) }
func (r *Recording) OnUint8(i uint8) error     { return r.add(Uint8Rec{i}) }
func (r *Recording) OnByte(i byte) error       { return r.add(ByteRec{i}) }
func (r *Recording) OnUint16(i uint16) error   { return r.add(Uint16Rec{i}) }
func (r *Recording) OnUint32(i uint32) error   { return r.add(Uint32Rec{i}) }
func (r *Recording) OnUint64(i uint64) error   { return r.add(Uint64Rec{i}) }
func (r *Recording) OnUint(i uint) error       { return r.add(UintRec{i}) }
func (r *Recording) OnFloat32(i float32) error { return r.add(Float32Rec{i}) }
func (r *Recording) OnFloat64(i float64) error { return r.add(Float64Rec{i}) }

func (r *Recording) OnArrayStart(len int, baseType structform.BaseType) error {
	return r.add(ArrayStartRec{len, baseType})
}
func (r *Recording) OnArrayFinished() error {
	return r.add(ArrayFinishRec{})
}

func (r *Recording) OnObjectStart(len int, baseType structform.BaseType) error {
	return r.add(ObjectStartRec{len, baseType})
}
func (r *Recording) OnObjectFinished() error {
	return r.add(ObjectFinishRec{})
}
func (r *Recording) OnKey(s string) error {
	return r.add(ObjectKeyRec{s})
}

func (r *Recording) expand() Recording {
	var to Recording
	r.Replay(&to)
	return to
}

func (r Recording) Assert(t *testing.T, expected Recording) {
	exp, err := expected.ToJSON()
	if err != nil {
		t.Error("Assert (expected): ", err)
		t.Logf("  recording: %#v", exp)
		return
	}

	act, err := r.ToJSON()
	if err != nil {
		t.Error("Assert (actual): ", err)
		t.Logf("  recording: %#v", r)
		return
	}

	assert.Equal(t, exp, act)
}

func (r Recording) ToJSON() (string, error) {
	v, _, err := buildValue(r.expand())
	if err != nil {
		return "", err
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type builder struct {
	stack []interface{}
	value interface{}
}

func buildValue(rec Recording) (interface{}, Recording, error) {
	if len(rec) == 0 {
		return nil, nil, errors.New("empty recording")
	}

	switch v := rec[0].(type) {
	case NilRec:
		return nil, rec[1:], nil
	case BoolRec:
		return v.Value, rec[1:], nil
	case StringRec:
		return v.Value, rec[1:], nil
	case IntRec:
		return v.Value, rec[1:], nil
	case Int8Rec:
		return v.Value, rec[1:], nil
	case Int16Rec:
		return v.Value, rec[1:], nil
	case Int32Rec:
		return v.Value, rec[1:], nil
	case Int64Rec:
		return v.Value, rec[1:], nil
	case UintRec:
		return v.Value, rec[1:], nil
	case ByteRec:
		return v.Value, rec[1:], nil
	case Uint8Rec:
		return v.Value, rec[1:], nil
	case Uint16Rec:
		return v.Value, rec[1:], nil
	case Uint32Rec:
		return v.Value, rec[1:], nil
	case Uint64Rec:
		return v.Value, rec[1:], nil
	case Float32Rec:
		return v.Value, rec[1:], nil
	case Float64Rec:
		return v.Value, rec[1:], nil
	case ArrayStartRec:
		return buildArray(rec[1:])
	case ObjectStartRec:
		return buildObject(rec[1:])

	default:
		return nil, nil, fmt.Errorf("Invalid record entry: %v", v)
	}
}

func buildArray(rec Recording) (interface{}, Recording, error) {
	a := []interface{}{}

	for len(rec) > 0 {
		var (
			v   interface{}
			err error
		)

		if _, end := rec[0].(ArrayFinishRec); end {
			return a, rec[1:], nil
		}

		v, rec, err = buildValue(rec)
		if err != nil {
			return nil, nil, err
		}

		a = append(a, v)
	}

	return nil, nil, errors.New("missing array finish record")
}

func buildObject(rec Recording) (interface{}, Recording, error) {
	obj := map[string]interface{}{}

	for len(rec) > 0 {
		var (
			key string
			v   interface{}
			err error
		)

		switch v := rec[0].(type) {
		case ObjectFinishRec:
			return obj, rec[1:], nil
		case ObjectKeyRec:
			key = v.Value
		}

		v, rec, err = buildValue(rec[1:])
		if err != nil {
			return nil, nil, err
		}

		obj[key] = v
	}

	return nil, nil, errors.New("missing object finish record")
}

func TestEncodeParseConsistent(
	t *testing.T,
	samples []Recording,
	constr func() (structform.Visitor, func(structform.Visitor) error),
) {
	for i, sample := range samples {
		expected, err := sample.ToJSON()
		if err != nil {
			panic(err)
		}

		t.Logf("test %v: %#v => %v", i, sample, expected)

		enc, dec := constr()

		err = sample.Replay(enc)
		if err != nil {
			t.Errorf("Failed to encode %#v with %v", sample, err)
			return
		}

		var target Recording
		err = dec(&target)
		if err != nil {
			t.Errorf("Failed to decode %#v with %v", target, err)
			t.Logf("  recording: %#v", target)
		}

		target.Assert(t, sample)
	}
}

func concatSamples(recs ...[]Recording) []Recording {
	var out []Recording
	for _, r := range recs {
		out = append(out, r...)
	}
	return out
}
