// This file has been generated from 'unfold_map.yml', do not edit
package gotype

import (
	"unsafe"

	"github.com/elastic/go-structform"
)

var (
	unfolderReflMapIfc = liftGoUnfolder(newUnfolderMapIfc())

	unfolderReflMapBool = liftGoUnfolder(newUnfolderMapBool())

	unfolderReflMapString = liftGoUnfolder(newUnfolderMapString())

	unfolderReflMapUint = liftGoUnfolder(newUnfolderMapUint())

	unfolderReflMapUint8 = liftGoUnfolder(newUnfolderMapUint8())

	unfolderReflMapUint16 = liftGoUnfolder(newUnfolderMapUint16())

	unfolderReflMapUint32 = liftGoUnfolder(newUnfolderMapUint32())

	unfolderReflMapUint64 = liftGoUnfolder(newUnfolderMapUint64())

	unfolderReflMapInt = liftGoUnfolder(newUnfolderMapInt())

	unfolderReflMapInt8 = liftGoUnfolder(newUnfolderMapInt8())

	unfolderReflMapInt16 = liftGoUnfolder(newUnfolderMapInt16())

	unfolderReflMapInt32 = liftGoUnfolder(newUnfolderMapInt32())

	unfolderReflMapInt64 = liftGoUnfolder(newUnfolderMapInt64())

	unfolderReflMapFloat32 = liftGoUnfolder(newUnfolderMapFloat32())

	unfolderReflMapFloat64 = liftGoUnfolder(newUnfolderMapFloat64())
)

type unfolderMapIfc struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapIfc = &unfolderMapIfc{}

func newUnfolderMapIfc() *unfolderMapIfc {
	return _singletonUnfolderMapIfc
}

type unfoldMapStartIfc struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartIfc = &unfoldMapStartIfc{}

func newUnfoldMapStartIfc() *unfoldMapStartIfc {
	return _singletonUnfoldMapStartIfc
}

type unfoldMapKeyIfc struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyIfc = &unfoldMapKeyIfc{}

func newUnfoldMapKeyIfc() *unfoldMapKeyIfc {
	return _singletonUnfoldMapKeyIfc
}

func (u *unfolderMapIfc) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyIfc())
	ctx.unfolder.push(newUnfoldMapStartIfc())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyIfc) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartIfc) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapIfc) ptr(ctx *unfoldCtx) *map[string]interface{} {
	return (*map[string]interface{})(ctx.ptr.current)
}

func (u *unfoldMapStartIfc) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyIfc) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyIfc) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapIfc()
	return nil
}

func (u *unfoldMapKeyIfc) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapIfc) put(ctx *unfoldCtx, v interface{}) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]interface{}{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyIfc()
	return nil
}

type unfolderMapBool struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapBool = &unfolderMapBool{}

func newUnfolderMapBool() *unfolderMapBool {
	return _singletonUnfolderMapBool
}

type unfoldMapStartBool struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartBool = &unfoldMapStartBool{}

func newUnfoldMapStartBool() *unfoldMapStartBool {
	return _singletonUnfoldMapStartBool
}

type unfoldMapKeyBool struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyBool = &unfoldMapKeyBool{}

func newUnfoldMapKeyBool() *unfoldMapKeyBool {
	return _singletonUnfoldMapKeyBool
}

func (u *unfolderMapBool) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyBool())
	ctx.unfolder.push(newUnfoldMapStartBool())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyBool) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartBool) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapBool) ptr(ctx *unfoldCtx) *map[string]bool {
	return (*map[string]bool)(ctx.ptr.current)
}

func (u *unfoldMapStartBool) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyBool) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyBool) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapBool()
	return nil
}

func (u *unfoldMapKeyBool) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapBool) put(ctx *unfoldCtx, v bool) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]bool{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyBool()
	return nil
}

type unfolderMapString struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapString = &unfolderMapString{}

func newUnfolderMapString() *unfolderMapString {
	return _singletonUnfolderMapString
}

type unfoldMapStartString struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartString = &unfoldMapStartString{}

func newUnfoldMapStartString() *unfoldMapStartString {
	return _singletonUnfoldMapStartString
}

type unfoldMapKeyString struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyString = &unfoldMapKeyString{}

func newUnfoldMapKeyString() *unfoldMapKeyString {
	return _singletonUnfoldMapKeyString
}

func (u *unfolderMapString) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyString())
	ctx.unfolder.push(newUnfoldMapStartString())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyString) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartString) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapString) ptr(ctx *unfoldCtx) *map[string]string {
	return (*map[string]string)(ctx.ptr.current)
}

func (u *unfoldMapStartString) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyString) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyString) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapString()
	return nil
}

func (u *unfoldMapKeyString) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapString) put(ctx *unfoldCtx, v string) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]string{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyString()
	return nil
}

type unfolderMapUint struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapUint = &unfolderMapUint{}

func newUnfolderMapUint() *unfolderMapUint {
	return _singletonUnfolderMapUint
}

type unfoldMapStartUint struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartUint = &unfoldMapStartUint{}

func newUnfoldMapStartUint() *unfoldMapStartUint {
	return _singletonUnfoldMapStartUint
}

type unfoldMapKeyUint struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyUint = &unfoldMapKeyUint{}

func newUnfoldMapKeyUint() *unfoldMapKeyUint {
	return _singletonUnfoldMapKeyUint
}

func (u *unfolderMapUint) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyUint())
	ctx.unfolder.push(newUnfoldMapStartUint())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyUint) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartUint) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapUint) ptr(ctx *unfoldCtx) *map[string]uint {
	return (*map[string]uint)(ctx.ptr.current)
}

func (u *unfoldMapStartUint) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyUint) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyUint) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapUint()
	return nil
}

func (u *unfoldMapKeyUint) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapUint) put(ctx *unfoldCtx, v uint) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]uint{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyUint()
	return nil
}

type unfolderMapUint8 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapUint8 = &unfolderMapUint8{}

func newUnfolderMapUint8() *unfolderMapUint8 {
	return _singletonUnfolderMapUint8
}

type unfoldMapStartUint8 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartUint8 = &unfoldMapStartUint8{}

func newUnfoldMapStartUint8() *unfoldMapStartUint8 {
	return _singletonUnfoldMapStartUint8
}

type unfoldMapKeyUint8 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyUint8 = &unfoldMapKeyUint8{}

func newUnfoldMapKeyUint8() *unfoldMapKeyUint8 {
	return _singletonUnfoldMapKeyUint8
}

func (u *unfolderMapUint8) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyUint8())
	ctx.unfolder.push(newUnfoldMapStartUint8())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyUint8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartUint8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapUint8) ptr(ctx *unfoldCtx) *map[string]uint8 {
	return (*map[string]uint8)(ctx.ptr.current)
}

func (u *unfoldMapStartUint8) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyUint8) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyUint8) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapUint8()
	return nil
}

func (u *unfoldMapKeyUint8) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapUint8) put(ctx *unfoldCtx, v uint8) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]uint8{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyUint8()
	return nil
}

type unfolderMapUint16 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapUint16 = &unfolderMapUint16{}

func newUnfolderMapUint16() *unfolderMapUint16 {
	return _singletonUnfolderMapUint16
}

type unfoldMapStartUint16 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartUint16 = &unfoldMapStartUint16{}

func newUnfoldMapStartUint16() *unfoldMapStartUint16 {
	return _singletonUnfoldMapStartUint16
}

type unfoldMapKeyUint16 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyUint16 = &unfoldMapKeyUint16{}

func newUnfoldMapKeyUint16() *unfoldMapKeyUint16 {
	return _singletonUnfoldMapKeyUint16
}

func (u *unfolderMapUint16) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyUint16())
	ctx.unfolder.push(newUnfoldMapStartUint16())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyUint16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartUint16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapUint16) ptr(ctx *unfoldCtx) *map[string]uint16 {
	return (*map[string]uint16)(ctx.ptr.current)
}

func (u *unfoldMapStartUint16) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyUint16) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyUint16) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapUint16()
	return nil
}

func (u *unfoldMapKeyUint16) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapUint16) put(ctx *unfoldCtx, v uint16) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]uint16{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyUint16()
	return nil
}

type unfolderMapUint32 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapUint32 = &unfolderMapUint32{}

func newUnfolderMapUint32() *unfolderMapUint32 {
	return _singletonUnfolderMapUint32
}

type unfoldMapStartUint32 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartUint32 = &unfoldMapStartUint32{}

func newUnfoldMapStartUint32() *unfoldMapStartUint32 {
	return _singletonUnfoldMapStartUint32
}

type unfoldMapKeyUint32 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyUint32 = &unfoldMapKeyUint32{}

func newUnfoldMapKeyUint32() *unfoldMapKeyUint32 {
	return _singletonUnfoldMapKeyUint32
}

func (u *unfolderMapUint32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyUint32())
	ctx.unfolder.push(newUnfoldMapStartUint32())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyUint32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartUint32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapUint32) ptr(ctx *unfoldCtx) *map[string]uint32 {
	return (*map[string]uint32)(ctx.ptr.current)
}

func (u *unfoldMapStartUint32) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyUint32) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyUint32) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapUint32()
	return nil
}

func (u *unfoldMapKeyUint32) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapUint32) put(ctx *unfoldCtx, v uint32) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]uint32{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyUint32()
	return nil
}

type unfolderMapUint64 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapUint64 = &unfolderMapUint64{}

func newUnfolderMapUint64() *unfolderMapUint64 {
	return _singletonUnfolderMapUint64
}

type unfoldMapStartUint64 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartUint64 = &unfoldMapStartUint64{}

func newUnfoldMapStartUint64() *unfoldMapStartUint64 {
	return _singletonUnfoldMapStartUint64
}

type unfoldMapKeyUint64 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyUint64 = &unfoldMapKeyUint64{}

func newUnfoldMapKeyUint64() *unfoldMapKeyUint64 {
	return _singletonUnfoldMapKeyUint64
}

func (u *unfolderMapUint64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyUint64())
	ctx.unfolder.push(newUnfoldMapStartUint64())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyUint64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartUint64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapUint64) ptr(ctx *unfoldCtx) *map[string]uint64 {
	return (*map[string]uint64)(ctx.ptr.current)
}

func (u *unfoldMapStartUint64) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyUint64) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyUint64) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapUint64()
	return nil
}

func (u *unfoldMapKeyUint64) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapUint64) put(ctx *unfoldCtx, v uint64) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]uint64{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyUint64()
	return nil
}

type unfolderMapInt struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapInt = &unfolderMapInt{}

func newUnfolderMapInt() *unfolderMapInt {
	return _singletonUnfolderMapInt
}

type unfoldMapStartInt struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartInt = &unfoldMapStartInt{}

func newUnfoldMapStartInt() *unfoldMapStartInt {
	return _singletonUnfoldMapStartInt
}

type unfoldMapKeyInt struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyInt = &unfoldMapKeyInt{}

func newUnfoldMapKeyInt() *unfoldMapKeyInt {
	return _singletonUnfoldMapKeyInt
}

func (u *unfolderMapInt) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyInt())
	ctx.unfolder.push(newUnfoldMapStartInt())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyInt) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartInt) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapInt) ptr(ctx *unfoldCtx) *map[string]int {
	return (*map[string]int)(ctx.ptr.current)
}

func (u *unfoldMapStartInt) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyInt) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyInt) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapInt()
	return nil
}

func (u *unfoldMapKeyInt) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapInt) put(ctx *unfoldCtx, v int) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]int{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyInt()
	return nil
}

type unfolderMapInt8 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapInt8 = &unfolderMapInt8{}

func newUnfolderMapInt8() *unfolderMapInt8 {
	return _singletonUnfolderMapInt8
}

type unfoldMapStartInt8 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartInt8 = &unfoldMapStartInt8{}

func newUnfoldMapStartInt8() *unfoldMapStartInt8 {
	return _singletonUnfoldMapStartInt8
}

type unfoldMapKeyInt8 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyInt8 = &unfoldMapKeyInt8{}

func newUnfoldMapKeyInt8() *unfoldMapKeyInt8 {
	return _singletonUnfoldMapKeyInt8
}

func (u *unfolderMapInt8) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyInt8())
	ctx.unfolder.push(newUnfoldMapStartInt8())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyInt8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartInt8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapInt8) ptr(ctx *unfoldCtx) *map[string]int8 {
	return (*map[string]int8)(ctx.ptr.current)
}

func (u *unfoldMapStartInt8) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyInt8) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyInt8) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapInt8()
	return nil
}

func (u *unfoldMapKeyInt8) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapInt8) put(ctx *unfoldCtx, v int8) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]int8{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyInt8()
	return nil
}

type unfolderMapInt16 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapInt16 = &unfolderMapInt16{}

func newUnfolderMapInt16() *unfolderMapInt16 {
	return _singletonUnfolderMapInt16
}

type unfoldMapStartInt16 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartInt16 = &unfoldMapStartInt16{}

func newUnfoldMapStartInt16() *unfoldMapStartInt16 {
	return _singletonUnfoldMapStartInt16
}

type unfoldMapKeyInt16 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyInt16 = &unfoldMapKeyInt16{}

func newUnfoldMapKeyInt16() *unfoldMapKeyInt16 {
	return _singletonUnfoldMapKeyInt16
}

func (u *unfolderMapInt16) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyInt16())
	ctx.unfolder.push(newUnfoldMapStartInt16())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyInt16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartInt16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapInt16) ptr(ctx *unfoldCtx) *map[string]int16 {
	return (*map[string]int16)(ctx.ptr.current)
}

func (u *unfoldMapStartInt16) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyInt16) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyInt16) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapInt16()
	return nil
}

func (u *unfoldMapKeyInt16) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapInt16) put(ctx *unfoldCtx, v int16) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]int16{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyInt16()
	return nil
}

type unfolderMapInt32 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapInt32 = &unfolderMapInt32{}

func newUnfolderMapInt32() *unfolderMapInt32 {
	return _singletonUnfolderMapInt32
}

type unfoldMapStartInt32 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartInt32 = &unfoldMapStartInt32{}

func newUnfoldMapStartInt32() *unfoldMapStartInt32 {
	return _singletonUnfoldMapStartInt32
}

type unfoldMapKeyInt32 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyInt32 = &unfoldMapKeyInt32{}

func newUnfoldMapKeyInt32() *unfoldMapKeyInt32 {
	return _singletonUnfoldMapKeyInt32
}

func (u *unfolderMapInt32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyInt32())
	ctx.unfolder.push(newUnfoldMapStartInt32())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyInt32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartInt32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapInt32) ptr(ctx *unfoldCtx) *map[string]int32 {
	return (*map[string]int32)(ctx.ptr.current)
}

func (u *unfoldMapStartInt32) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyInt32) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyInt32) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapInt32()
	return nil
}

func (u *unfoldMapKeyInt32) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapInt32) put(ctx *unfoldCtx, v int32) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]int32{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyInt32()
	return nil
}

type unfolderMapInt64 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapInt64 = &unfolderMapInt64{}

func newUnfolderMapInt64() *unfolderMapInt64 {
	return _singletonUnfolderMapInt64
}

type unfoldMapStartInt64 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartInt64 = &unfoldMapStartInt64{}

func newUnfoldMapStartInt64() *unfoldMapStartInt64 {
	return _singletonUnfoldMapStartInt64
}

type unfoldMapKeyInt64 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyInt64 = &unfoldMapKeyInt64{}

func newUnfoldMapKeyInt64() *unfoldMapKeyInt64 {
	return _singletonUnfoldMapKeyInt64
}

func (u *unfolderMapInt64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyInt64())
	ctx.unfolder.push(newUnfoldMapStartInt64())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyInt64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartInt64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapInt64) ptr(ctx *unfoldCtx) *map[string]int64 {
	return (*map[string]int64)(ctx.ptr.current)
}

func (u *unfoldMapStartInt64) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyInt64) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyInt64) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapInt64()
	return nil
}

func (u *unfoldMapKeyInt64) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapInt64) put(ctx *unfoldCtx, v int64) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]int64{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyInt64()
	return nil
}

type unfolderMapFloat32 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapFloat32 = &unfolderMapFloat32{}

func newUnfolderMapFloat32() *unfolderMapFloat32 {
	return _singletonUnfolderMapFloat32
}

type unfoldMapStartFloat32 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartFloat32 = &unfoldMapStartFloat32{}

func newUnfoldMapStartFloat32() *unfoldMapStartFloat32 {
	return _singletonUnfoldMapStartFloat32
}

type unfoldMapKeyFloat32 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyFloat32 = &unfoldMapKeyFloat32{}

func newUnfoldMapKeyFloat32() *unfoldMapKeyFloat32 {
	return _singletonUnfoldMapKeyFloat32
}

func (u *unfolderMapFloat32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyFloat32())
	ctx.unfolder.push(newUnfoldMapStartFloat32())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyFloat32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartFloat32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapFloat32) ptr(ctx *unfoldCtx) *map[string]float32 {
	return (*map[string]float32)(ctx.ptr.current)
}

func (u *unfoldMapStartFloat32) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyFloat32) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyFloat32) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapFloat32()
	return nil
}

func (u *unfoldMapKeyFloat32) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapFloat32) put(ctx *unfoldCtx, v float32) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]float32{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyFloat32()
	return nil
}

type unfolderMapFloat64 struct {
	unfolderErrUnknown
}

var _singletonUnfolderMapFloat64 = &unfolderMapFloat64{}

func newUnfolderMapFloat64() *unfolderMapFloat64 {
	return _singletonUnfolderMapFloat64
}

type unfoldMapStartFloat64 struct {
	unfolderErrObjectStart
}

var _singletonUnfoldMapStartFloat64 = &unfoldMapStartFloat64{}

func newUnfoldMapStartFloat64() *unfoldMapStartFloat64 {
	return _singletonUnfoldMapStartFloat64
}

type unfoldMapKeyFloat64 struct {
	unfolderErrExpectKey
}

var _singletonUnfoldMapKeyFloat64 = &unfoldMapKeyFloat64{}

func newUnfoldMapKeyFloat64() *unfoldMapKeyFloat64 {
	return _singletonUnfoldMapKeyFloat64
}

func (u *unfolderMapFloat64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(newUnfoldMapKeyFloat64())
	ctx.unfolder.push(newUnfoldMapStartFloat64())
	ctx.ptr.push(ptr)
}

func (u *unfoldMapKeyFloat64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *unfoldMapStartFloat64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfolderMapFloat64) ptr(ctx *unfoldCtx) *map[string]float64 {
	return (*map[string]float64)(ctx.ptr.current)
}

func (u *unfoldMapStartFloat64) OnObjectStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	// TODO: validate baseType

	u.cleanup(ctx)
	return nil
}

func (u *unfoldMapKeyFloat64) OnKeyRef(ctx *unfoldCtx, key []byte) error {
	return u.OnKey(ctx, ctx.keyCache.get(key))
}

func (u *unfoldMapKeyFloat64) OnKey(ctx *unfoldCtx, key string) error {
	ctx.key.push(key)
	ctx.unfolder.current = newUnfolderMapFloat64()
	return nil
}

func (u *unfoldMapKeyFloat64) OnObjectFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderMapFloat64) put(ctx *unfoldCtx, v float64) error {
	to := u.ptr(ctx)
	if *to == nil {
		*to = map[string]float64{}
	}
	(*to)[ctx.key.pop()] = v

	ctx.unfolder.current = newUnfoldMapKeyFloat64()
	return nil
}

func (u *unfolderMapIfc) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, nil)
}

func (u *unfolderMapIfc) OnBool(ctx *unfoldCtx, v bool) error { return u.put(ctx, v) }

func (u *unfolderMapIfc) OnString(ctx *unfoldCtx, v string) error { return u.put(ctx, v) }
func (u *unfolderMapIfc) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderMapIfc) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, (interface{})(v))
}

func (u *unfolderMapIfc) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, (interface{})(v))
}

func (*unfolderMapIfc) OnArrayStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	return unfoldIfcStartSubArray(ctx, l, bt)
}

func (u *unfolderMapIfc) OnChildArrayDone(ctx *unfoldCtx) error {
	v, err := unfoldIfcFinishSubArray(ctx)
	if err == nil {
		err = u.put(ctx, v)
	}
	return err
}

func (*unfolderMapIfc) OnObjectStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	return unfoldIfcStartSubMap(ctx, l, bt)
}

func (u *unfolderMapIfc) OnChildObjectDone(ctx *unfoldCtx) error {
	v, err := unfoldIfcFinishSubMap(ctx)
	if err == nil {
		err = u.put(ctx, v)
	}
	return err
}

func (u *unfolderMapBool) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, false)
}

func (u *unfolderMapBool) OnBool(ctx *unfoldCtx, v bool) error { return u.put(ctx, v) }

func (u *unfolderMapString) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, "")
}

func (u *unfolderMapString) OnString(ctx *unfoldCtx, v string) error { return u.put(ctx, v) }
func (u *unfolderMapString) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderMapUint) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapUint) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, uint(v))
}

func (u *unfolderMapUint8) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapUint8) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint8) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, uint8(v))
}

func (u *unfolderMapUint16) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapUint16) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint16) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, uint16(v))
}

func (u *unfolderMapUint32) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapUint32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, uint32(v))
}

func (u *unfolderMapUint64) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapUint64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapUint64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, uint64(v))
}

func (u *unfolderMapInt) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapInt) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, int(v))
}

func (u *unfolderMapInt8) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapInt8) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt8) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, int8(v))
}

func (u *unfolderMapInt16) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapInt16) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt16) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, int16(v))
}

func (u *unfolderMapInt32) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapInt32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, int32(v))
}

func (u *unfolderMapInt64) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapInt64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapInt64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, int64(v))
}

func (u *unfolderMapFloat32) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapFloat32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, float32(v))
}

func (u *unfolderMapFloat64) OnNil(ctx *unfoldCtx) error {
	return u.put(ctx, 0)
}

func (u *unfolderMapFloat64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnInt(ctx *unfoldCtx, v int) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.put(ctx, float64(v))
}

func (u *unfolderMapFloat64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.put(ctx, float64(v))
}

func unfoldIfcStartSubMap(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	_, ptr, unfolder := makeMapPtr(ctx, l, baseType)
	ctx.ptr.push(ptr)
	ctx.baseType.push(baseType)
	unfolder.initState(ctx, ptr)
	return ctx.unfolder.current.OnObjectStart(ctx, l, baseType)
}

func unfoldIfcFinishSubMap(ctx *unfoldCtx) (interface{}, error) {
	child := ctx.ptr.pop()
	bt := ctx.baseType.pop()
	switch bt {

	case structform.AnyType:
		value := *(*map[string]interface{})(child)
		last := len(ctx.valueBuffer.mapAny) - 1
		ctx.valueBuffer.mapAny = ctx.valueBuffer.mapAny[:last]
		return value, nil

	case structform.BoolType:
		value := *(*map[string]bool)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.ByteType:
		value := *(*map[string]uint8)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Float32Type:
		value := *(*map[string]float32)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Float64Type:
		value := *(*map[string]float64)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Int16Type:
		value := *(*map[string]int16)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Int32Type:
		value := *(*map[string]int32)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Int64Type:
		value := *(*map[string]int64)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Int8Type:
		value := *(*map[string]int8)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.IntType:
		value := *(*map[string]int)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.StringType:
		value := *(*map[string]string)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Uint16Type:
		value := *(*map[string]uint16)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Uint32Type:
		value := *(*map[string]uint32)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Uint64Type:
		value := *(*map[string]uint64)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.Uint8Type:
		value := *(*map[string]uint8)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.UintType:
		value := *(*map[string]uint)(child)
		last := len(ctx.valueBuffer.mapPrimitive) - 1
		ctx.valueBuffer.mapPrimitive = ctx.valueBuffer.mapPrimitive[:last]
		return value, nil

	case structform.ZeroType:
		value := *(*map[string]interface{})(child)
		last := len(ctx.valueBuffer.mapAny) - 1
		ctx.valueBuffer.mapAny = ctx.valueBuffer.mapAny[:last]
		return value, nil

	default:
		return nil, errTODO()
	}
}

func makeMapPtr(ctx *unfoldCtx, l int, bt structform.BaseType) (interface{}, unsafe.Pointer, ptrUnfolder) {
	switch bt {

	case structform.AnyType:
		idx := len(ctx.valueBuffer.mapAny)
		ctx.valueBuffer.mapAny = append(ctx.valueBuffer.mapAny, nil)
		to := &ctx.valueBuffer.mapAny[idx]
		ptr := unsafe.Pointer(to)
		unfolder := newUnfolderMapIfc()
		return to, ptr, unfolder

	case structform.BoolType:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]bool)(ptr)
		unfolder := newUnfolderMapBool()
		return to, ptr, unfolder

	case structform.ByteType:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]uint8)(ptr)
		unfolder := newUnfolderMapUint8()
		return to, ptr, unfolder

	case structform.Float32Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]float32)(ptr)
		unfolder := newUnfolderMapFloat32()
		return to, ptr, unfolder

	case structform.Float64Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]float64)(ptr)
		unfolder := newUnfolderMapFloat64()
		return to, ptr, unfolder

	case structform.Int16Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]int16)(ptr)
		unfolder := newUnfolderMapInt16()
		return to, ptr, unfolder

	case structform.Int32Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]int32)(ptr)
		unfolder := newUnfolderMapInt32()
		return to, ptr, unfolder

	case structform.Int64Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]int64)(ptr)
		unfolder := newUnfolderMapInt64()
		return to, ptr, unfolder

	case structform.Int8Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]int8)(ptr)
		unfolder := newUnfolderMapInt8()
		return to, ptr, unfolder

	case structform.IntType:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]int)(ptr)
		unfolder := newUnfolderMapInt()
		return to, ptr, unfolder

	case structform.StringType:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]string)(ptr)
		unfolder := newUnfolderMapString()
		return to, ptr, unfolder

	case structform.Uint16Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]uint16)(ptr)
		unfolder := newUnfolderMapUint16()
		return to, ptr, unfolder

	case structform.Uint32Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]uint32)(ptr)
		unfolder := newUnfolderMapUint32()
		return to, ptr, unfolder

	case structform.Uint64Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]uint64)(ptr)
		unfolder := newUnfolderMapUint64()
		return to, ptr, unfolder

	case structform.Uint8Type:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]uint8)(ptr)
		unfolder := newUnfolderMapUint8()
		return to, ptr, unfolder

	case structform.UintType:
		idx := len(ctx.valueBuffer.mapPrimitive)
		ctx.valueBuffer.mapPrimitive = append(ctx.valueBuffer.mapPrimitive, nil)
		mapPtr := &ctx.valueBuffer.mapPrimitive[idx]
		ptr := unsafe.Pointer(mapPtr)
		to := (*map[string]uint)(ptr)
		unfolder := newUnfolderMapUint()
		return to, ptr, unfolder

	case structform.ZeroType:
		idx := len(ctx.valueBuffer.mapAny)
		ctx.valueBuffer.mapAny = append(ctx.valueBuffer.mapAny, nil)
		to := &ctx.valueBuffer.mapAny[idx]
		ptr := unsafe.Pointer(to)
		unfolder := newUnfolderMapIfc()
		return to, ptr, unfolder

	default:
		panic("invalid type code")
		return nil, nil, nil
	}
}
