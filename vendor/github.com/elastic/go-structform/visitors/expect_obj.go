package visitors

import (
	"errors"

	structform "github.com/elastic/go-structform"
)

type ExpectObjVisitor struct {
	active structform.ExtVisitor
	depth  int
}

func NewExpectObjVisitor(target structform.ExtVisitor) *ExpectObjVisitor {
	return &ExpectObjVisitor{active: target}
}

func (e *ExpectObjVisitor) SetActive(a structform.ExtVisitor) {
	e.active = a
	e.depth = 0
}

func (e *ExpectObjVisitor) Done() bool {
	return e.depth == 0
}

func (v *ExpectObjVisitor) OnObjectStart(len int, baseType structform.BaseType) error {
	v.depth++
	if v.depth == 1 {
		return nil
	}
	return v.active.OnObjectStart(len, baseType)
}

func (v *ExpectObjVisitor) OnObjectFinished() error {
	v.depth--
	if v.depth == 0 {
		return nil
	}
	return v.active.OnObjectFinished()
}

func (v *ExpectObjVisitor) OnKey(s string) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnKey(s)
}

func (v *ExpectObjVisitor) OnArrayStart(len int, baseType structform.BaseType) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnArrayStart(len, baseType)
}

func (v *ExpectObjVisitor) OnArrayFinished() error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnArrayFinished()
}

func (v *ExpectObjVisitor) OnNil() error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnNil()
}

func (v *ExpectObjVisitor) OnBool(b bool) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnBool(b)
}

func (v *ExpectObjVisitor) OnString(s string) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnString(s)
}

func (v *ExpectObjVisitor) OnInt8(i int8) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt8(i)
}

func (v *ExpectObjVisitor) OnInt16(i int16) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt16(i)
}

func (v *ExpectObjVisitor) OnInt32(i int32) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt32(i)
}

func (v *ExpectObjVisitor) OnInt64(i int64) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt64(i)
}

func (v *ExpectObjVisitor) OnInt(i int) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt(i)
}

func (v *ExpectObjVisitor) OnByte(b byte) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnByte(b)
}

func (v *ExpectObjVisitor) OnUint8(u uint8) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint8(u)
}

func (v *ExpectObjVisitor) OnUint16(u uint16) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint16(u)
}

func (v *ExpectObjVisitor) OnUint32(u uint32) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint32(u)
}

func (v *ExpectObjVisitor) OnUint64(u uint64) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint64(u)
}

func (v *ExpectObjVisitor) OnUint(u uint) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint(u)
}

func (v *ExpectObjVisitor) OnFloat32(f float32) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnFloat32(f)
}

func (v *ExpectObjVisitor) OnFloat64(f float64) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnFloat64(f)
}

func (v *ExpectObjVisitor) OnStringRef(s []byte) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnStringRef(s)
}

func (v *ExpectObjVisitor) OnKeyRef(s []byte) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnKeyRef(s)
}

func (v *ExpectObjVisitor) check() error {
	if v.depth == 0 {
		return errors.New("inline object is no object")
	}
	return nil
}
