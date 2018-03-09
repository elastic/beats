// This file has been generated from 'unfold_ignore.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"

	structform "github.com/elastic/go-structform"
)

type unfoldIgnoreValue struct{}
type unfoldIgnorePtr struct{}
type unfolderIgnore struct {
	unfolderErrUnknown
}

var (
	_singletonUnfoldIgnoreValue = &unfoldIgnoreValue{}
	_singletonUnfoldIgnorePtr   = &unfoldIgnorePtr{}
	_singletonunfolderIgnore    = &unfolderIgnore{}
)

func (*unfoldIgnoreValue) initState(ctx *unfoldCtx, _ reflect.Value) {
	ctx.unfolder.push(_singletonunfolderIgnore)
}

func (*unfoldIgnorePtr) initState(ctx *unfoldCtx, _ unsafe.Pointer) {
	ctx.unfolder.push(_singletonunfolderIgnore)
}

func (u *unfolderIgnore) OnNil(ctx *unfoldCtx) error                { return u.onValue(ctx) }
func (u *unfolderIgnore) OnBool(ctx *unfoldCtx, _ bool) error       { return u.onValue(ctx) }
func (u *unfolderIgnore) OnString(ctx *unfoldCtx, _ string) error   { return u.onValue(ctx) }
func (u *unfolderIgnore) OnInt8(ctx *unfoldCtx, _ int8) error       { return u.onValue(ctx) }
func (u *unfolderIgnore) OnInt16(ctx *unfoldCtx, _ int16) error     { return u.onValue(ctx) }
func (u *unfolderIgnore) OnInt32(ctx *unfoldCtx, _ int32) error     { return u.onValue(ctx) }
func (u *unfolderIgnore) OnInt64(ctx *unfoldCtx, _ int64) error     { return u.onValue(ctx) }
func (u *unfolderIgnore) OnInt(ctx *unfoldCtx, _ int) error         { return u.onValue(ctx) }
func (u *unfolderIgnore) OnByte(ctx *unfoldCtx, _ byte) error       { return u.onValue(ctx) }
func (u *unfolderIgnore) OnUint8(ctx *unfoldCtx, _ uint8) error     { return u.onValue(ctx) }
func (u *unfolderIgnore) OnUint16(ctx *unfoldCtx, _ uint16) error   { return u.onValue(ctx) }
func (u *unfolderIgnore) OnUint32(ctx *unfoldCtx, _ uint32) error   { return u.onValue(ctx) }
func (u *unfolderIgnore) OnUint64(ctx *unfoldCtx, _ uint64) error   { return u.onValue(ctx) }
func (u *unfolderIgnore) OnUint(ctx *unfoldCtx, _ uint) error       { return u.onValue(ctx) }
func (u *unfolderIgnore) OnFloat32(ctx *unfoldCtx, _ float32) error { return u.onValue(ctx) }
func (u *unfolderIgnore) OnFloat64(ctx *unfoldCtx, _ float64) error { return u.onValue(ctx) }

func (u *unfolderIgnore) OnArrayStart(ctx *unfoldCtx, _ int, _ structform.BaseType) error {
	_singletonUnfoldIgnoreArrPtr.initState(ctx, nil)
	return nil
}

func (u *unfolderIgnore) OnChildArrayDone(ctx *unfoldCtx) error {
	return u.onValue(ctx)
}

func (u *unfolderIgnore) OnObjectStart(ctx *unfoldCtx, _ int, _ structform.BaseType) error {
	_singletonUnfoldIgnoreObjPtr.initState(ctx, nil)
	return nil
}

func (u *unfolderIgnore) OnChildObjectDone(ctx *unfoldCtx) error {
	return u.onValue(ctx)
}

func (*unfolderIgnore) onValue(ctx *unfoldCtx) error {
	ctx.unfolder.pop()
	return nil
}

type unfoldIgnoreArrValue struct{}
type unfoldIgnoreArrPtr struct{}
type unfolderIgnoreArr struct {
	unfolderErrUnknown
}

var (
	_singletonUnfoldIgnoreArrValue = &unfoldIgnoreArrValue{}
	_singletonUnfoldIgnoreArrPtr   = &unfoldIgnoreArrPtr{}
	_singletonunfolderIgnoreArr    = &unfolderIgnoreArr{}
)

func (*unfoldIgnoreArrValue) initState(ctx *unfoldCtx, _ reflect.Value) {
	ctx.unfolder.push(_singletonunfolderIgnoreArr)
}

func (*unfoldIgnoreArrPtr) initState(ctx *unfoldCtx, _ unsafe.Pointer) {
	ctx.unfolder.push(_singletonunfolderIgnoreArr)
}

func (u *unfolderIgnoreArr) OnNil(ctx *unfoldCtx) error                { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnBool(ctx *unfoldCtx, _ bool) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnString(ctx *unfoldCtx, _ string) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnInt8(ctx *unfoldCtx, _ int8) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnInt16(ctx *unfoldCtx, _ int16) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnInt32(ctx *unfoldCtx, _ int32) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnInt64(ctx *unfoldCtx, _ int64) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnInt(ctx *unfoldCtx, _ int) error         { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnByte(ctx *unfoldCtx, _ byte) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnUint8(ctx *unfoldCtx, _ uint8) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnUint16(ctx *unfoldCtx, _ uint16) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnUint32(ctx *unfoldCtx, _ uint32) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnUint64(ctx *unfoldCtx, _ uint64) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnUint(ctx *unfoldCtx, _ uint) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnFloat32(ctx *unfoldCtx, _ float32) error { return u.onValue(ctx) }
func (u *unfolderIgnoreArr) OnFloat64(ctx *unfoldCtx, _ float64) error { return u.onValue(ctx) }

func (u *unfolderIgnoreArr) OnArrayStart(ctx *unfoldCtx, _ int, _ structform.BaseType) error {
	_singletonUnfoldIgnoreArrPtr.initState(ctx, nil)
	return nil
}

func (u *unfolderIgnoreArr) OnChildArrayDone(ctx *unfoldCtx) error {
	return u.onValue(ctx)
}

func (u *unfolderIgnoreArr) OnObjectStart(ctx *unfoldCtx, _ int, _ structform.BaseType) error {
	_singletonUnfoldIgnoreObjPtr.initState(ctx, nil)
	return nil
}

func (u *unfolderIgnoreArr) OnChildObjectDone(ctx *unfoldCtx) error {
	return u.onValue(ctx)
}

func (*unfolderIgnoreArr) onValue(ctx *unfoldCtx) error {
	return nil
}

func (*unfolderIgnoreArr) OnArrayFinished(ctx *unfoldCtx) error {
	ctx.unfolder.pop()
	return nil
}

type unfoldIgnoreObjValue struct{}
type unfoldIgnoreObjPtr struct{}
type unfolderIgnoreObj struct {
	unfolderErrUnknown
}

var (
	_singletonUnfoldIgnoreObjValue = &unfoldIgnoreObjValue{}
	_singletonUnfoldIgnoreObjPtr   = &unfoldIgnoreObjPtr{}
	_singletonunfolderIgnoreObj    = &unfolderIgnoreObj{}
)

func (*unfoldIgnoreObjValue) initState(ctx *unfoldCtx, _ reflect.Value) {
	ctx.unfolder.push(_singletonunfolderIgnoreObj)
}

func (*unfoldIgnoreObjPtr) initState(ctx *unfoldCtx, _ unsafe.Pointer) {
	ctx.unfolder.push(_singletonunfolderIgnoreObj)
}

func (u *unfolderIgnoreObj) OnNil(ctx *unfoldCtx) error                { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnBool(ctx *unfoldCtx, _ bool) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnString(ctx *unfoldCtx, _ string) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnInt8(ctx *unfoldCtx, _ int8) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnInt16(ctx *unfoldCtx, _ int16) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnInt32(ctx *unfoldCtx, _ int32) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnInt64(ctx *unfoldCtx, _ int64) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnInt(ctx *unfoldCtx, _ int) error         { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnByte(ctx *unfoldCtx, _ byte) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnUint8(ctx *unfoldCtx, _ uint8) error     { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnUint16(ctx *unfoldCtx, _ uint16) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnUint32(ctx *unfoldCtx, _ uint32) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnUint64(ctx *unfoldCtx, _ uint64) error   { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnUint(ctx *unfoldCtx, _ uint) error       { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnFloat32(ctx *unfoldCtx, _ float32) error { return u.onValue(ctx) }
func (u *unfolderIgnoreObj) OnFloat64(ctx *unfoldCtx, _ float64) error { return u.onValue(ctx) }

func (u *unfolderIgnoreObj) OnArrayStart(ctx *unfoldCtx, _ int, _ structform.BaseType) error {
	_singletonUnfoldIgnoreArrPtr.initState(ctx, nil)
	return nil
}

func (u *unfolderIgnoreObj) OnChildArrayDone(ctx *unfoldCtx) error {
	return u.onValue(ctx)
}

func (u *unfolderIgnoreObj) OnObjectStart(ctx *unfoldCtx, _ int, _ structform.BaseType) error {
	_singletonUnfoldIgnoreObjPtr.initState(ctx, nil)
	return nil
}

func (u *unfolderIgnoreObj) OnChildObjectDone(ctx *unfoldCtx) error {
	return u.onValue(ctx)
}

func (*unfolderIgnoreObj) onValue(ctx *unfoldCtx) error {
	return nil
}

func (*unfolderIgnoreObj) OnObjectFinished(ctx *unfoldCtx) error {
	ctx.unfolder.pop()
	return nil
}
