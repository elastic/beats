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

// This file has been generated from 'unfold_user_primitive.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"

	stunsafe "github.com/elastic/go-structform/internal/unsafe"
)

type (
	userUnfolderBool struct {
		unfolderErrUnknown
		fn userUnfolderBoolCB
	}

	userUnfolderBoolCB func(unsafe.Pointer, bool) error
)

func newUserUnfolderBool(fn reflect.Value) ptrUnfolder {
	return &userUnfolderBool{
		fn: *((*userUnfolderBoolCB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderBool) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderBool) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderBool) process(ctx *unfoldCtx, v bool) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderString struct {
		unfolderErrUnknown
		fn userUnfolderStringCB
	}

	userUnfolderStringCB func(unsafe.Pointer, string) error
)

func newUserUnfolderString(fn reflect.Value) ptrUnfolder {
	return &userUnfolderString{
		fn: *((*userUnfolderStringCB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderString) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderString) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderString) process(ctx *unfoldCtx, v string) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderUint struct {
		unfolderErrUnknown
		fn userUnfolderUintCB
	}

	userUnfolderUintCB func(unsafe.Pointer, uint) error
)

func newUserUnfolderUint(fn reflect.Value) ptrUnfolder {
	return &userUnfolderUint{
		fn: *((*userUnfolderUintCB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderUint) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderUint) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderUint) process(ctx *unfoldCtx, v uint) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderUint8 struct {
		unfolderErrUnknown
		fn userUnfolderUint8CB
	}

	userUnfolderUint8CB func(unsafe.Pointer, uint8) error
)

func newUserUnfolderUint8(fn reflect.Value) ptrUnfolder {
	return &userUnfolderUint8{
		fn: *((*userUnfolderUint8CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderUint8) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderUint8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderUint8) process(ctx *unfoldCtx, v uint8) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderUint16 struct {
		unfolderErrUnknown
		fn userUnfolderUint16CB
	}

	userUnfolderUint16CB func(unsafe.Pointer, uint16) error
)

func newUserUnfolderUint16(fn reflect.Value) ptrUnfolder {
	return &userUnfolderUint16{
		fn: *((*userUnfolderUint16CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderUint16) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderUint16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderUint16) process(ctx *unfoldCtx, v uint16) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderUint32 struct {
		unfolderErrUnknown
		fn userUnfolderUint32CB
	}

	userUnfolderUint32CB func(unsafe.Pointer, uint32) error
)

func newUserUnfolderUint32(fn reflect.Value) ptrUnfolder {
	return &userUnfolderUint32{
		fn: *((*userUnfolderUint32CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderUint32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderUint32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderUint32) process(ctx *unfoldCtx, v uint32) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderUint64 struct {
		unfolderErrUnknown
		fn userUnfolderUint64CB
	}

	userUnfolderUint64CB func(unsafe.Pointer, uint64) error
)

func newUserUnfolderUint64(fn reflect.Value) ptrUnfolder {
	return &userUnfolderUint64{
		fn: *((*userUnfolderUint64CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderUint64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderUint64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderUint64) process(ctx *unfoldCtx, v uint64) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderInt struct {
		unfolderErrUnknown
		fn userUnfolderIntCB
	}

	userUnfolderIntCB func(unsafe.Pointer, int) error
)

func newUserUnfolderInt(fn reflect.Value) ptrUnfolder {
	return &userUnfolderInt{
		fn: *((*userUnfolderIntCB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderInt) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderInt) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderInt) process(ctx *unfoldCtx, v int) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderInt8 struct {
		unfolderErrUnknown
		fn userUnfolderInt8CB
	}

	userUnfolderInt8CB func(unsafe.Pointer, int8) error
)

func newUserUnfolderInt8(fn reflect.Value) ptrUnfolder {
	return &userUnfolderInt8{
		fn: *((*userUnfolderInt8CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderInt8) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderInt8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderInt8) process(ctx *unfoldCtx, v int8) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderInt16 struct {
		unfolderErrUnknown
		fn userUnfolderInt16CB
	}

	userUnfolderInt16CB func(unsafe.Pointer, int16) error
)

func newUserUnfolderInt16(fn reflect.Value) ptrUnfolder {
	return &userUnfolderInt16{
		fn: *((*userUnfolderInt16CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderInt16) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderInt16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderInt16) process(ctx *unfoldCtx, v int16) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderInt32 struct {
		unfolderErrUnknown
		fn userUnfolderInt32CB
	}

	userUnfolderInt32CB func(unsafe.Pointer, int32) error
)

func newUserUnfolderInt32(fn reflect.Value) ptrUnfolder {
	return &userUnfolderInt32{
		fn: *((*userUnfolderInt32CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderInt32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderInt32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderInt32) process(ctx *unfoldCtx, v int32) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderInt64 struct {
		unfolderErrUnknown
		fn userUnfolderInt64CB
	}

	userUnfolderInt64CB func(unsafe.Pointer, int64) error
)

func newUserUnfolderInt64(fn reflect.Value) ptrUnfolder {
	return &userUnfolderInt64{
		fn: *((*userUnfolderInt64CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderInt64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderInt64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderInt64) process(ctx *unfoldCtx, v int64) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderFloat32 struct {
		unfolderErrUnknown
		fn userUnfolderFloat32CB
	}

	userUnfolderFloat32CB func(unsafe.Pointer, float32) error
)

func newUserUnfolderFloat32(fn reflect.Value) ptrUnfolder {
	return &userUnfolderFloat32{
		fn: *((*userUnfolderFloat32CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderFloat32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderFloat32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderFloat32) process(ctx *unfoldCtx, v float32) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

type (
	userUnfolderFloat64 struct {
		unfolderErrUnknown
		fn userUnfolderFloat64CB
	}

	userUnfolderFloat64CB func(unsafe.Pointer, float64) error
)

func newUserUnfolderFloat64(fn reflect.Value) ptrUnfolder {
	return &userUnfolderFloat64{
		fn: *((*userUnfolderFloat64CB)(stunsafe.UnsafeFnPtr(fn))),
	}
}

func (u *userUnfolderFloat64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.ptr.push(ptr)
}

func (u *userUnfolderFloat64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.ptr.pop()
}

func (u *userUnfolderFloat64) process(ctx *unfoldCtx, v float64) error {
	err := u.fn(ctx.ptr.current, v)
	u.cleanup(ctx)
	return err
}

func (u *userUnfolderBool) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, false)
}

func (u *userUnfolderBool) OnBool(ctx *unfoldCtx, v bool) error { return u.process(ctx, v) }

func (u *userUnfolderString) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, "")
}

func (u *userUnfolderString) OnString(ctx *unfoldCtx, v string) error { return u.process(ctx, v) }
func (u *userUnfolderString) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *userUnfolderUint) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderUint) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, uint(v))
}

func (u *userUnfolderUint8) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderUint8) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint8) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, uint8(v))
}

func (u *userUnfolderUint16) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderUint16) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint16) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, uint16(v))
}

func (u *userUnfolderUint32) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderUint32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, uint32(v))
}

func (u *userUnfolderUint64) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderUint64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderUint64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, uint64(v))
}

func (u *userUnfolderInt) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderInt) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, int(v))
}

func (u *userUnfolderInt8) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderInt8) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt8) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, int8(v))
}

func (u *userUnfolderInt16) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderInt16) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt16) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, int16(v))
}

func (u *userUnfolderInt32) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderInt32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, int32(v))
}

func (u *userUnfolderInt64) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderInt64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderInt64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, int64(v))
}

func (u *userUnfolderFloat32) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderFloat32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, float32(v))
}

func (u *userUnfolderFloat64) OnNil(ctx *unfoldCtx) error {
	return u.process(ctx, 0)
}

func (u *userUnfolderFloat64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnInt(ctx *unfoldCtx, v int) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.process(ctx, float64(v))
}

func (u *userUnfolderFloat64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.process(ctx, float64(v))
}
