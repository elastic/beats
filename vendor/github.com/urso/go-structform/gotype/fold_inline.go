package gotype

import (
	"errors"
	"reflect"

	structform "github.com/urso/go-structform"
)

type expectObjVisitor struct {
	active visitor
	depth  int
}

// getReflectFoldMapKeys implements inline fold of a map[string]X type,
// not reporting object start/end events
func getReflectFoldMapKeys(c *foldContext, t reflect.Type) (reFoldFn, error) {
	if t.Key().Kind() != reflect.String {
		return nil, errMapRequiresStringKey
	}

	elemVisitor, err := getReflectFold(c, t.Elem())
	if err != nil {
		return nil, err
	}

	return makeMapKeysFold(elemVisitor), nil
}

func makeMapKeysFold(elemVisitor reFoldFn) reFoldFn {
	return func(C *foldContext, rv reflect.Value) error {
		if rv.IsNil() || !rv.IsValid() {
			return nil
		}

		for _, k := range rv.MapKeys() {
			if err := C.OnKey(k.String()); err != nil {
				return err
			}
			if err := elemVisitor(C, rv.MapIndex(k)); err != nil {
				return err
			}
		}
		return nil
	}
}

// getReflectFoldInlineInterface create an inline folder for an yet unknown type.
// The actual types folder must open/close an object
func getReflectFoldInlineInterface(C *foldContext, t reflect.Type) (reFoldFn, error) {

	var (
		ctx = *C
		vs  = &expectObjVisitor{}

		// cache last used folder
		lastType    reflect.Type
		lastVisitor reFoldFn
	)

	ctx.visitor = structform.EnsureExtVisitor(vs).(visitor)
	return func(C *foldContext, rv reflect.Value) error {
		vs.active = C.visitor

		// don't inline missing object
		if rv.IsNil() || !rv.IsValid() {
			return nil
		}

		if rv.Type() != lastType {
			elemVisitor, err := getReflectFold(&ctx, rv.Type())
			if err != nil {
				return err
			}

			lastVisitor = elemVisitor
			lastType = rv.Type()
		}

		vs.depth = 0
		if err := lastVisitor(&ctx, rv); err != nil {
			return err
		}

		if vs.depth != 0 {
			return errors.New("missing object close")
		}

		return nil
	}, nil
}

func (v *expectObjVisitor) OnObjectStart(len int, baseType structform.BaseType) error {
	v.depth++
	if v.depth == 1 {
		return nil
	}
	return v.active.OnObjectStart(len, baseType)
}

func (v *expectObjVisitor) OnObjectFinished() error {
	v.depth--
	if v.depth == 0 {
		return nil
	}
	return v.active.OnObjectFinished()
}

func (v *expectObjVisitor) OnKey(s string) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnKey(s)
}

func (v *expectObjVisitor) OnArrayStart(len int, baseType structform.BaseType) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnArrayStart(len, baseType)
}

func (v *expectObjVisitor) OnArrayFinished() error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnArrayFinished()
}

func (v *expectObjVisitor) OnNil() error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnNil()
}

func (v *expectObjVisitor) OnBool(b bool) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnBool(b)
}

func (v *expectObjVisitor) OnString(s string) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnString(s)
}

func (v *expectObjVisitor) OnInt8(i int8) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt8(i)
}

func (v *expectObjVisitor) OnInt16(i int16) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt16(i)
}

func (v *expectObjVisitor) OnInt32(i int32) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt32(i)
}

func (v *expectObjVisitor) OnInt64(i int64) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt64(i)
}

func (v *expectObjVisitor) OnInt(i int) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnInt(i)
}

func (v *expectObjVisitor) OnByte(b byte) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnByte(b)
}

func (v *expectObjVisitor) OnUint8(u uint8) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint8(u)
}

func (v *expectObjVisitor) OnUint16(u uint16) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint16(u)
}

func (v *expectObjVisitor) OnUint32(u uint32) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint32(u)
}

func (v *expectObjVisitor) OnUint64(u uint64) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint64(u)
}

func (v *expectObjVisitor) OnUint(u uint) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnUint(u)
}

func (v *expectObjVisitor) OnFloat32(f float32) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnFloat32(f)
}

func (v *expectObjVisitor) OnFloat64(f float64) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnFloat64(f)
}

func (v *expectObjVisitor) OnStringRef(s []byte) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnStringRef(s)
}

func (v *expectObjVisitor) OnKeyRef(s []byte) error {
	if err := v.check(); err != nil {
		return err
	}
	return v.active.OnKeyRef(s)
}

func (v *expectObjVisitor) check() error {
	if v.depth == 0 {
		return errors.New("inline object is no object")
	}
	return nil
}
