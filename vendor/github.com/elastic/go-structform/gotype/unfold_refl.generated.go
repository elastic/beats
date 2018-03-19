// This file has been generated from 'unfold_refl.yml', do not edit
package gotype

import (
	"reflect"

	structform "github.com/elastic/go-structform"
)

func (u *unfolderReflSlice) OnNil(ctx *unfoldCtx) error {
	u.prepare(ctx)
	return nil
}

func (u *unfolderReflSlice) OnByte(ctx *unfoldCtx, v byte) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnByte(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderReflSlice) OnBool(ctx *unfoldCtx, v bool) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnBool(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnString(ctx *unfoldCtx, v string) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnString(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnUint(ctx *unfoldCtx, v uint) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnUint8(ctx *unfoldCtx, v uint8) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint8(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnUint16(ctx *unfoldCtx, v uint16) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint16(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnUint32(ctx *unfoldCtx, v uint32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint32(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnUint64(ctx *unfoldCtx, v uint64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint64(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnInt(ctx *unfoldCtx, v int) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnInt8(ctx *unfoldCtx, v int8) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt8(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnInt16(ctx *unfoldCtx, v int16) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt16(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnInt32(ctx *unfoldCtx, v int32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt32(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnInt64(ctx *unfoldCtx, v int64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt64(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnFloat32(ctx *unfoldCtx, v float32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnFloat32(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnFloat64(ctx *unfoldCtx, v float64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnFloat64(ctx, v)

	return err
}

func (u *unfolderReflSlice) OnArrayStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	return ctx.unfolder.current.OnArrayStart(ctx, l, bt)
}

func (u *unfolderReflSlice) OnChildArrayDone(ctx *unfoldCtx) error {

	return nil
}

func (u *unfolderReflSlice) OnObjectStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	return ctx.unfolder.current.OnObjectStart(ctx, l, bt)
}

func (u *unfolderReflSlice) OnKey(_ *unfoldCtx, _ string) error {
	return errUnsupported
}

func (u *unfolderReflSlice) OnKeyRef(_ *unfoldCtx, _ []byte) error {
	return errUnsupported
}

func (u *unfolderReflSlice) OnChildObjectDone(ctx *unfoldCtx) error {

	return nil
}

func (u *unfolderReflMapOnElem) OnNil(ctx *unfoldCtx) error {
	ptr := ctx.value.current
	m := ptr.Elem()
	v := reflect.Zero(m.Type().Elem())
	m.SetMapIndex(reflect.ValueOf(ctx.key.pop()), v)

	ctx.unfolder.current = u.shared.waitKey
	return nil
}

func (u *unfolderReflMapOnElem) OnByte(ctx *unfoldCtx, v byte) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnByte(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderReflMapOnElem) OnBool(ctx *unfoldCtx, v bool) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnBool(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnString(ctx *unfoldCtx, v string) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnString(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnUint(ctx *unfoldCtx, v uint) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnUint8(ctx *unfoldCtx, v uint8) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint8(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnUint16(ctx *unfoldCtx, v uint16) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint16(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnUint32(ctx *unfoldCtx, v uint32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint32(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnUint64(ctx *unfoldCtx, v uint64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint64(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnInt(ctx *unfoldCtx, v int) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnInt8(ctx *unfoldCtx, v int8) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt8(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnInt16(ctx *unfoldCtx, v int16) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt16(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnInt32(ctx *unfoldCtx, v int32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt32(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnInt64(ctx *unfoldCtx, v int64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt64(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnFloat32(ctx *unfoldCtx, v float32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnFloat32(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnFloat64(ctx *unfoldCtx, v float64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnFloat64(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflMapOnElem) OnArrayStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	return ctx.unfolder.current.OnArrayStart(ctx, l, bt)
}

func (u *unfolderReflMapOnElem) OnChildArrayDone(ctx *unfoldCtx) error {

	u.process(ctx)

	return nil
}

func (u *unfolderReflMapOnElem) OnObjectStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	return ctx.unfolder.current.OnObjectStart(ctx, l, bt)
}

func (u *unfolderReflMapOnElem) OnKey(_ *unfoldCtx, _ string) error {
	return errExpectedObjectValue
}

func (u *unfolderReflMapOnElem) OnKeyRef(_ *unfoldCtx, _ []byte) error {
	return errExpectedObjectValue
}

func (u *unfolderReflMapOnElem) OnChildObjectDone(ctx *unfoldCtx) error {

	u.process(ctx)

	return nil
}

func (u *unfolderReflPtr) OnNil(ctx *unfoldCtx) error {
	ptr := ctx.value.current
	v := ptr.Elem()
	v.Set(reflect.Zero(v.Type()))
	u.cleanup(ctx)
	return nil
}

func (u *unfolderReflPtr) OnByte(ctx *unfoldCtx, v byte) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnByte(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderReflPtr) OnBool(ctx *unfoldCtx, v bool) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnBool(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnString(ctx *unfoldCtx, v string) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnString(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnUint(ctx *unfoldCtx, v uint) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnUint8(ctx *unfoldCtx, v uint8) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint8(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnUint16(ctx *unfoldCtx, v uint16) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint16(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnUint32(ctx *unfoldCtx, v uint32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint32(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnUint64(ctx *unfoldCtx, v uint64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnUint64(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnInt(ctx *unfoldCtx, v int) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnInt8(ctx *unfoldCtx, v int8) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt8(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnInt16(ctx *unfoldCtx, v int16) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt16(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnInt32(ctx *unfoldCtx, v int32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt32(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnInt64(ctx *unfoldCtx, v int64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnInt64(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnFloat32(ctx *unfoldCtx, v float32) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnFloat32(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnFloat64(ctx *unfoldCtx, v float64) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	err := ctx.unfolder.current.OnFloat64(ctx, v)

	if err == nil {
		u.process(ctx)
	}

	return err
}

func (u *unfolderReflPtr) OnArrayStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	return ctx.unfolder.current.OnArrayStart(ctx, l, bt)
}

func (u *unfolderReflPtr) OnChildArrayDone(ctx *unfoldCtx) error {

	u.process(ctx)

	return nil
}

func (u *unfolderReflPtr) OnObjectStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	elem := u.prepare(ctx)
	u.elem.initState(ctx, elem)
	return ctx.unfolder.current.OnObjectStart(ctx, l, bt)
}

func (u *unfolderReflPtr) OnKey(_ *unfoldCtx, _ string) error {
	return errUnsupported
}

func (u *unfolderReflPtr) OnKeyRef(_ *unfoldCtx, _ []byte) error {
	return errUnsupported
}

func (u *unfolderReflPtr) OnChildObjectDone(ctx *unfoldCtx) error {

	u.process(ctx)

	return nil
}
