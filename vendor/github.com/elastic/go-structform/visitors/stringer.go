package visitors

import (
	"fmt"

	structform "github.com/elastic/go-structform"
)

type StringConvVisitor struct {
	active structform.ExtVisitor
}

func NewStringConvVisitor(target structform.ExtVisitor) *StringConvVisitor {
	return &StringConvVisitor{target}
}

func (v *StringConvVisitor) SetActive(a structform.ExtVisitor) {
	v.active = a
}

func (v *StringConvVisitor) OnObjectStart(l int, t structform.BaseType) error {
	return v.active.OnObjectStart(l, t)
}

func (v *StringConvVisitor) OnObjectFinished() error {
	return v.active.OnObjectFinished()
}

func (v *StringConvVisitor) OnKey(s string) error {
	return v.active.OnKey(s)
}

func (v *StringConvVisitor) OnKeyRef(s []byte) error {
	return v.active.OnKeyRef(s)
}

func (v *StringConvVisitor) OnArrayStart(l int, t structform.BaseType) error {
	return v.active.OnArrayStart(l, t)
}

func (v *StringConvVisitor) OnArrayFinished() error {
	return v.active.OnArrayFinished()
}

func (v *StringConvVisitor) OnNil() error {
	return v.OnString("")
}

func (v *StringConvVisitor) OnBool(b bool) error {
	t := "false"
	if b {
		t = "true"
	}
	return v.OnString(t)
}

func (v *StringConvVisitor) OnString(s string) error {
	return v.active.OnString(s)
}

func (v *StringConvVisitor) OnStringRef(b []byte) error {
	return v.active.OnStringRef(b)
}

func (v *StringConvVisitor) OnInt8(i int8) error {
	return v.OnInt64(int64(i))
}

func (v *StringConvVisitor) OnInt16(i int16) error {
	return v.OnInt64(int64(i))
}

func (v *StringConvVisitor) OnInt32(i int32) error {
	return v.OnInt64(int64(i))
}

func (v *StringConvVisitor) OnInt64(i int64) error {
	return v.OnString(fmt.Sprintf("%v", i))
}

func (v *StringConvVisitor) OnInt(i int) error {
	return v.OnInt64(int64(i))
}

func (v *StringConvVisitor) OnUint8(i uint8) error {
	return v.OnUint64(uint64(i))
}

func (v *StringConvVisitor) OnUint16(i uint16) error {
	return v.OnUint64(uint64(i))
}

func (v *StringConvVisitor) OnUint32(i uint32) error {
	return v.OnUint64(uint64(i))
}

func (v *StringConvVisitor) OnUint64(i uint64) error {
	return v.OnString(fmt.Sprintf("%v", i))
}

func (v *StringConvVisitor) OnUint(i uint) error {
	return v.OnUint64(uint64(i))
}

func (v *StringConvVisitor) OnFloat32(f float32) error {
	return v.OnString(fmt.Sprintf("%v", f))
}

func (v *StringConvVisitor) OnFloat64(f float64) error {
	return v.OnString(fmt.Sprintf("%v", f))
}
