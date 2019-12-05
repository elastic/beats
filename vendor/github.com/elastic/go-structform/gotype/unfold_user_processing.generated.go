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

// This file has been generated from 'unfold_user_processing.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"

	structform "github.com/elastic/go-structform"
)

type unfolderUserProcessingInit struct {
	fnInit userProcessingInitFn
}

type unfolderUserFailing struct {
	err error
}

type unfolderUserProcessing struct {
	// XXX: move user processing into unfoldCtx as stacks? -> no more allocations
	startSz int
	fn      userProcessingFn
}

type userProcessingInitFn func(unsafe.Pointer) (interface{}, userProcessingFn)

type userProcessingFn func(unsafe.Pointer, interface{}) error

func (u *unfolderUserProcessingInit) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	cell, cont := u.fnInit(ptr)

	unfolder, err := lookupReflUnfolder(ctx, reflect.TypeOf(cell), false)
	if err != nil {
		// Use unfolderUserFailing. If there is a chance that the no value is
		// unfolded into the current target, then we can continue processing
		// without reporting said error.
		ctx.unfolder.push(&unfolderUserFailing{err})
		return
	}

	startSz := len(ctx.unfolder.stack)

	v := reflect.ValueOf(cell)
	ctx.ptr.push(ptr)
	ctx.value.push(v)

	unfolder.initState(ctx, v)
	ctx.unfolder.push(&unfolderUserProcessing{
		startSz: startSz,
		fn:      cont,
	})
}

func (u *unfolderUserFailing) OnNil(*unfoldCtx) error                                   { return u.err }
func (u *unfolderUserFailing) OnBool(*unfoldCtx, bool) error                            { return u.err }
func (u *unfolderUserFailing) OnByte(*unfoldCtx, byte) error                            { return u.err }
func (u *unfolderUserFailing) OnString(*unfoldCtx, string) error                        { return u.err }
func (u *unfolderUserFailing) OnStringRef(*unfoldCtx, []byte) error                     { return u.err }
func (u *unfolderUserFailing) OnInt8(*unfoldCtx, int8) error                            { return u.err }
func (u *unfolderUserFailing) OnInt16(*unfoldCtx, int16) error                          { return u.err }
func (u *unfolderUserFailing) OnInt32(*unfoldCtx, int32) error                          { return u.err }
func (u *unfolderUserFailing) OnInt64(*unfoldCtx, int64) error                          { return u.err }
func (u *unfolderUserFailing) OnInt(*unfoldCtx, int) error                              { return u.err }
func (u *unfolderUserFailing) OnUint8(*unfoldCtx, uint8) error                          { return u.err }
func (u *unfolderUserFailing) OnUint16(*unfoldCtx, uint16) error                        { return u.err }
func (u *unfolderUserFailing) OnUint32(*unfoldCtx, uint32) error                        { return u.err }
func (u *unfolderUserFailing) OnUint64(*unfoldCtx, uint64) error                        { return u.err }
func (u *unfolderUserFailing) OnUint(*unfoldCtx, uint) error                            { return u.err }
func (u *unfolderUserFailing) OnFloat32(*unfoldCtx, float32) error                      { return u.err }
func (u *unfolderUserFailing) OnFloat64(*unfoldCtx, float64) error                      { return u.err }
func (u *unfolderUserFailing) OnArrayStart(*unfoldCtx, int, structform.BaseType) error  { return u.err }
func (u *unfolderUserFailing) OnArrayFinished(*unfoldCtx) error                         { return u.err }
func (u *unfolderUserFailing) OnChildArrayDone(*unfoldCtx) error                        { return u.err }
func (u *unfolderUserFailing) OnObjectStart(*unfoldCtx, int, structform.BaseType) error { return u.err }
func (u *unfolderUserFailing) OnObjectFinished(*unfoldCtx) error                        { return u.err }
func (u *unfolderUserFailing) OnKey(*unfoldCtx, string) error                           { return u.err }
func (u *unfolderUserFailing) OnKeyRef(*unfoldCtx, []byte) error                        { return u.err }
func (u *unfolderUserFailing) OnChildObjectDone(*unfoldCtx) error                       { return u.err }

func (u *unfolderUserProcessing) beforeCall(ctx *unfoldCtx) {
	ctx.unfolder.pop() // temporarily remove unfolder from top of stack
}

func (u *unfolderUserProcessing) afterCall(ctx *unfoldCtx, err error) error {
	if err != nil {
		return err
	}

	if u.startSz >= len(ctx.unfolder.stack) {
		return u.finalize(ctx)
	}

	ctx.unfolder.push(u)
	return nil
}

func (u *unfolderUserProcessing) finalize(ctx *unfoldCtx) error {
	return u.fn(ctx.ptr.pop(), ctx.value.pop().Interface())
}

func (u *unfolderUserProcessing) OnNil(ctx *unfoldCtx) error {
	u.beforeCall(ctx)
	err := ctx.OnNil()
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnStringRef(ctx *unfoldCtx, v []byte) error {
	u.beforeCall(ctx)
	err := ctx.OnStringRef(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnBool(ctx *unfoldCtx, v bool) error {
	u.beforeCall(ctx)
	err := ctx.OnBool(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnString(ctx *unfoldCtx, v string) error {
	u.beforeCall(ctx)
	err := ctx.OnString(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnUint(ctx *unfoldCtx, v uint) error {
	u.beforeCall(ctx)
	err := ctx.OnUint(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnUint8(ctx *unfoldCtx, v uint8) error {
	u.beforeCall(ctx)
	err := ctx.OnUint8(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnUint16(ctx *unfoldCtx, v uint16) error {
	u.beforeCall(ctx)
	err := ctx.OnUint16(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnUint32(ctx *unfoldCtx, v uint32) error {
	u.beforeCall(ctx)
	err := ctx.OnUint32(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnUint64(ctx *unfoldCtx, v uint64) error {
	u.beforeCall(ctx)
	err := ctx.OnUint64(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnInt(ctx *unfoldCtx, v int) error {
	u.beforeCall(ctx)
	err := ctx.OnInt(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnInt8(ctx *unfoldCtx, v int8) error {
	u.beforeCall(ctx)
	err := ctx.OnInt8(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnInt16(ctx *unfoldCtx, v int16) error {
	u.beforeCall(ctx)
	err := ctx.OnInt16(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnInt32(ctx *unfoldCtx, v int32) error {
	u.beforeCall(ctx)
	err := ctx.OnInt32(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnInt64(ctx *unfoldCtx, v int64) error {
	u.beforeCall(ctx)
	err := ctx.OnInt64(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnFloat32(ctx *unfoldCtx, v float32) error {
	u.beforeCall(ctx)
	err := ctx.OnFloat32(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnFloat64(ctx *unfoldCtx, v float64) error {
	u.beforeCall(ctx)
	err := ctx.OnFloat64(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnByte(ctx *unfoldCtx, v byte) error {
	u.beforeCall(ctx)
	err := ctx.OnByte(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnArrayStart(ctx *unfoldCtx, v int, bt structform.BaseType) error {
	u.beforeCall(ctx)
	err := ctx.unfolder.current.OnArrayStart(ctx, v, bt)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnArrayFinished(ctx *unfoldCtx) error {
	u.beforeCall(ctx)
	err := ctx.unfolder.current.OnArrayFinished(ctx)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnChildArrayDone(ctx *unfoldCtx) error {
	u.beforeCall(ctx)
	err := ctx.unfolder.current.OnChildArrayDone(ctx)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnObjectStart(ctx *unfoldCtx, v int, bt structform.BaseType) error {
	u.beforeCall(ctx)
	err := ctx.unfolder.current.OnObjectStart(ctx, v, bt)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnObjectFinished(ctx *unfoldCtx) error {
	u.beforeCall(ctx)
	err := ctx.unfolder.current.OnObjectFinished(ctx)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnChildObjectDone(ctx *unfoldCtx) error {
	u.beforeCall(ctx)
	err := ctx.unfolder.current.OnChildObjectDone(ctx)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnKey(ctx *unfoldCtx, v string) error {
	u.beforeCall(ctx)
	err := ctx.OnKey(v)
	return u.afterCall(ctx, err)
}

func (u *unfolderUserProcessing) OnKeyRef(ctx *unfoldCtx, v []byte) error {
	u.beforeCall(ctx)
	err := ctx.OnKeyRef(v)
	return u.afterCall(ctx, err)
}
