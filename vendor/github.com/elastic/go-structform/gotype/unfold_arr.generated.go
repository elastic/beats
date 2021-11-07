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

// This file has been generated from 'unfold_arr.yml', do not edit
package gotype

import (
	"unsafe"

	structform "github.com/elastic/go-structform"
)

var (
	unfolderReflArrIfc = liftGoUnfolder(newUnfolderArrIfc())

	unfolderReflArrBool = liftGoUnfolder(newUnfolderArrBool())

	unfolderReflArrString = liftGoUnfolder(newUnfolderArrString())

	unfolderReflArrUint = liftGoUnfolder(newUnfolderArrUint())

	unfolderReflArrUint8 = liftGoUnfolder(newUnfolderArrUint8())

	unfolderReflArrUint16 = liftGoUnfolder(newUnfolderArrUint16())

	unfolderReflArrUint32 = liftGoUnfolder(newUnfolderArrUint32())

	unfolderReflArrUint64 = liftGoUnfolder(newUnfolderArrUint64())

	unfolderReflArrInt = liftGoUnfolder(newUnfolderArrInt())

	unfolderReflArrInt8 = liftGoUnfolder(newUnfolderArrInt8())

	unfolderReflArrInt16 = liftGoUnfolder(newUnfolderArrInt16())

	unfolderReflArrInt32 = liftGoUnfolder(newUnfolderArrInt32())

	unfolderReflArrInt64 = liftGoUnfolder(newUnfolderArrInt64())

	unfolderReflArrFloat32 = liftGoUnfolder(newUnfolderArrFloat32())

	unfolderReflArrFloat64 = liftGoUnfolder(newUnfolderArrFloat64())
)

type unfolderArrIfc struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrIfc = &unfolderArrIfc{}

func newUnfolderArrIfc() *unfolderArrIfc {
	return _singletonUnfolderArrIfc
}

type unfoldArrStartIfc struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartIfc = &unfoldArrStartIfc{}

func newUnfoldArrStartIfc() *unfoldArrStartIfc {
	return _singletonUnfoldArrStartIfc
}

func (u *unfolderArrIfc) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartIfc())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrIfc) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartIfc) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartIfc) ptr(ctx *unfoldCtx) *[]interface{} {
	return (*[]interface{})(ctx.ptr.current)
}

func (u *unfolderArrIfc) ptr(ctx *unfoldCtx) *[]interface{} {
	return (*[]interface{})(ctx.ptr.current)
}

func (u *unfoldArrStartIfc) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]interface{}, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrIfc) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrIfc) append(ctx *unfoldCtx, v interface{}) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrBool struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrBool = &unfolderArrBool{}

func newUnfolderArrBool() *unfolderArrBool {
	return _singletonUnfolderArrBool
}

type unfoldArrStartBool struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartBool = &unfoldArrStartBool{}

func newUnfoldArrStartBool() *unfoldArrStartBool {
	return _singletonUnfoldArrStartBool
}

func (u *unfolderArrBool) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartBool())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrBool) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartBool) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartBool) ptr(ctx *unfoldCtx) *[]bool {
	return (*[]bool)(ctx.ptr.current)
}

func (u *unfolderArrBool) ptr(ctx *unfoldCtx) *[]bool {
	return (*[]bool)(ctx.ptr.current)
}

func (u *unfoldArrStartBool) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]bool, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrBool) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrBool) append(ctx *unfoldCtx, v bool) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrString struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrString = &unfolderArrString{}

func newUnfolderArrString() *unfolderArrString {
	return _singletonUnfolderArrString
}

type unfoldArrStartString struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartString = &unfoldArrStartString{}

func newUnfoldArrStartString() *unfoldArrStartString {
	return _singletonUnfoldArrStartString
}

func (u *unfolderArrString) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartString())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrString) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartString) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartString) ptr(ctx *unfoldCtx) *[]string {
	return (*[]string)(ctx.ptr.current)
}

func (u *unfolderArrString) ptr(ctx *unfoldCtx) *[]string {
	return (*[]string)(ctx.ptr.current)
}

func (u *unfoldArrStartString) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]string, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrString) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrString) append(ctx *unfoldCtx, v string) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrUint struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrUint = &unfolderArrUint{}

func newUnfolderArrUint() *unfolderArrUint {
	return _singletonUnfolderArrUint
}

type unfoldArrStartUint struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartUint = &unfoldArrStartUint{}

func newUnfoldArrStartUint() *unfoldArrStartUint {
	return _singletonUnfoldArrStartUint
}

func (u *unfolderArrUint) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartUint())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrUint) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartUint) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartUint) ptr(ctx *unfoldCtx) *[]uint {
	return (*[]uint)(ctx.ptr.current)
}

func (u *unfolderArrUint) ptr(ctx *unfoldCtx) *[]uint {
	return (*[]uint)(ctx.ptr.current)
}

func (u *unfoldArrStartUint) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]uint, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint) append(ctx *unfoldCtx, v uint) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrUint8 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrUint8 = &unfolderArrUint8{}

func newUnfolderArrUint8() *unfolderArrUint8 {
	return _singletonUnfolderArrUint8
}

type unfoldArrStartUint8 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartUint8 = &unfoldArrStartUint8{}

func newUnfoldArrStartUint8() *unfoldArrStartUint8 {
	return _singletonUnfoldArrStartUint8
}

func (u *unfolderArrUint8) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartUint8())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrUint8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartUint8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartUint8) ptr(ctx *unfoldCtx) *[]uint8 {
	return (*[]uint8)(ctx.ptr.current)
}

func (u *unfolderArrUint8) ptr(ctx *unfoldCtx) *[]uint8 {
	return (*[]uint8)(ctx.ptr.current)
}

func (u *unfoldArrStartUint8) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]uint8, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint8) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint8) append(ctx *unfoldCtx, v uint8) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrUint16 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrUint16 = &unfolderArrUint16{}

func newUnfolderArrUint16() *unfolderArrUint16 {
	return _singletonUnfolderArrUint16
}

type unfoldArrStartUint16 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartUint16 = &unfoldArrStartUint16{}

func newUnfoldArrStartUint16() *unfoldArrStartUint16 {
	return _singletonUnfoldArrStartUint16
}

func (u *unfolderArrUint16) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartUint16())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrUint16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartUint16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartUint16) ptr(ctx *unfoldCtx) *[]uint16 {
	return (*[]uint16)(ctx.ptr.current)
}

func (u *unfolderArrUint16) ptr(ctx *unfoldCtx) *[]uint16 {
	return (*[]uint16)(ctx.ptr.current)
}

func (u *unfoldArrStartUint16) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]uint16, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint16) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint16) append(ctx *unfoldCtx, v uint16) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrUint32 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrUint32 = &unfolderArrUint32{}

func newUnfolderArrUint32() *unfolderArrUint32 {
	return _singletonUnfolderArrUint32
}

type unfoldArrStartUint32 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartUint32 = &unfoldArrStartUint32{}

func newUnfoldArrStartUint32() *unfoldArrStartUint32 {
	return _singletonUnfoldArrStartUint32
}

func (u *unfolderArrUint32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartUint32())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrUint32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartUint32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartUint32) ptr(ctx *unfoldCtx) *[]uint32 {
	return (*[]uint32)(ctx.ptr.current)
}

func (u *unfolderArrUint32) ptr(ctx *unfoldCtx) *[]uint32 {
	return (*[]uint32)(ctx.ptr.current)
}

func (u *unfoldArrStartUint32) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]uint32, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint32) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint32) append(ctx *unfoldCtx, v uint32) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrUint64 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrUint64 = &unfolderArrUint64{}

func newUnfolderArrUint64() *unfolderArrUint64 {
	return _singletonUnfolderArrUint64
}

type unfoldArrStartUint64 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartUint64 = &unfoldArrStartUint64{}

func newUnfoldArrStartUint64() *unfoldArrStartUint64 {
	return _singletonUnfoldArrStartUint64
}

func (u *unfolderArrUint64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartUint64())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrUint64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartUint64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartUint64) ptr(ctx *unfoldCtx) *[]uint64 {
	return (*[]uint64)(ctx.ptr.current)
}

func (u *unfolderArrUint64) ptr(ctx *unfoldCtx) *[]uint64 {
	return (*[]uint64)(ctx.ptr.current)
}

func (u *unfoldArrStartUint64) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]uint64, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint64) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrUint64) append(ctx *unfoldCtx, v uint64) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrInt struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrInt = &unfolderArrInt{}

func newUnfolderArrInt() *unfolderArrInt {
	return _singletonUnfolderArrInt
}

type unfoldArrStartInt struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartInt = &unfoldArrStartInt{}

func newUnfoldArrStartInt() *unfoldArrStartInt {
	return _singletonUnfoldArrStartInt
}

func (u *unfolderArrInt) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartInt())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrInt) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartInt) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartInt) ptr(ctx *unfoldCtx) *[]int {
	return (*[]int)(ctx.ptr.current)
}

func (u *unfolderArrInt) ptr(ctx *unfoldCtx) *[]int {
	return (*[]int)(ctx.ptr.current)
}

func (u *unfoldArrStartInt) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]int, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt) append(ctx *unfoldCtx, v int) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrInt8 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrInt8 = &unfolderArrInt8{}

func newUnfolderArrInt8() *unfolderArrInt8 {
	return _singletonUnfolderArrInt8
}

type unfoldArrStartInt8 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartInt8 = &unfoldArrStartInt8{}

func newUnfoldArrStartInt8() *unfoldArrStartInt8 {
	return _singletonUnfoldArrStartInt8
}

func (u *unfolderArrInt8) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartInt8())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrInt8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartInt8) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartInt8) ptr(ctx *unfoldCtx) *[]int8 {
	return (*[]int8)(ctx.ptr.current)
}

func (u *unfolderArrInt8) ptr(ctx *unfoldCtx) *[]int8 {
	return (*[]int8)(ctx.ptr.current)
}

func (u *unfoldArrStartInt8) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]int8, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt8) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt8) append(ctx *unfoldCtx, v int8) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrInt16 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrInt16 = &unfolderArrInt16{}

func newUnfolderArrInt16() *unfolderArrInt16 {
	return _singletonUnfolderArrInt16
}

type unfoldArrStartInt16 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartInt16 = &unfoldArrStartInt16{}

func newUnfoldArrStartInt16() *unfoldArrStartInt16 {
	return _singletonUnfoldArrStartInt16
}

func (u *unfolderArrInt16) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartInt16())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrInt16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartInt16) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartInt16) ptr(ctx *unfoldCtx) *[]int16 {
	return (*[]int16)(ctx.ptr.current)
}

func (u *unfolderArrInt16) ptr(ctx *unfoldCtx) *[]int16 {
	return (*[]int16)(ctx.ptr.current)
}

func (u *unfoldArrStartInt16) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]int16, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt16) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt16) append(ctx *unfoldCtx, v int16) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrInt32 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrInt32 = &unfolderArrInt32{}

func newUnfolderArrInt32() *unfolderArrInt32 {
	return _singletonUnfolderArrInt32
}

type unfoldArrStartInt32 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartInt32 = &unfoldArrStartInt32{}

func newUnfoldArrStartInt32() *unfoldArrStartInt32 {
	return _singletonUnfoldArrStartInt32
}

func (u *unfolderArrInt32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartInt32())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrInt32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartInt32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartInt32) ptr(ctx *unfoldCtx) *[]int32 {
	return (*[]int32)(ctx.ptr.current)
}

func (u *unfolderArrInt32) ptr(ctx *unfoldCtx) *[]int32 {
	return (*[]int32)(ctx.ptr.current)
}

func (u *unfoldArrStartInt32) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]int32, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt32) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt32) append(ctx *unfoldCtx, v int32) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrInt64 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrInt64 = &unfolderArrInt64{}

func newUnfolderArrInt64() *unfolderArrInt64 {
	return _singletonUnfolderArrInt64
}

type unfoldArrStartInt64 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartInt64 = &unfoldArrStartInt64{}

func newUnfoldArrStartInt64() *unfoldArrStartInt64 {
	return _singletonUnfoldArrStartInt64
}

func (u *unfolderArrInt64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartInt64())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrInt64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartInt64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartInt64) ptr(ctx *unfoldCtx) *[]int64 {
	return (*[]int64)(ctx.ptr.current)
}

func (u *unfolderArrInt64) ptr(ctx *unfoldCtx) *[]int64 {
	return (*[]int64)(ctx.ptr.current)
}

func (u *unfoldArrStartInt64) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]int64, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt64) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrInt64) append(ctx *unfoldCtx, v int64) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrFloat32 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrFloat32 = &unfolderArrFloat32{}

func newUnfolderArrFloat32() *unfolderArrFloat32 {
	return _singletonUnfolderArrFloat32
}

type unfoldArrStartFloat32 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartFloat32 = &unfoldArrStartFloat32{}

func newUnfoldArrStartFloat32() *unfoldArrStartFloat32 {
	return _singletonUnfoldArrStartFloat32
}

func (u *unfolderArrFloat32) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartFloat32())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrFloat32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartFloat32) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartFloat32) ptr(ctx *unfoldCtx) *[]float32 {
	return (*[]float32)(ctx.ptr.current)
}

func (u *unfolderArrFloat32) ptr(ctx *unfoldCtx) *[]float32 {
	return (*[]float32)(ctx.ptr.current)
}

func (u *unfoldArrStartFloat32) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]float32, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrFloat32) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrFloat32) append(ctx *unfoldCtx, v float32) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

type unfolderArrFloat64 struct {
	unfolderErrUnknown
}

var _singletonUnfolderArrFloat64 = &unfolderArrFloat64{}

func newUnfolderArrFloat64() *unfolderArrFloat64 {
	return _singletonUnfolderArrFloat64
}

type unfoldArrStartFloat64 struct {
	unfolderErrArrayStart
}

var _singletonUnfoldArrStartFloat64 = &unfoldArrStartFloat64{}

func newUnfoldArrStartFloat64() *unfoldArrStartFloat64 {
	return _singletonUnfoldArrStartFloat64
}

func (u *unfolderArrFloat64) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	ctx.unfolder.push(u)
	ctx.unfolder.push(newUnfoldArrStartFloat64())
	ctx.idx.push(0)
	ctx.ptr.push(ptr)
}

func (u *unfolderArrFloat64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
	ctx.idx.pop()
	ctx.ptr.pop()
}

func (u *unfoldArrStartFloat64) cleanup(ctx *unfoldCtx) {
	ctx.unfolder.pop()
}

func (u *unfoldArrStartFloat64) ptr(ctx *unfoldCtx) *[]float64 {
	return (*[]float64)(ctx.ptr.current)
}

func (u *unfolderArrFloat64) ptr(ctx *unfoldCtx) *[]float64 {
	return (*[]float64)(ctx.ptr.current)
}

func (u *unfoldArrStartFloat64) OnArrayStart(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	to := u.ptr(ctx)
	if l < 0 {
		l = 0
	}

	if *to == nil && l > 0 {
		*to = make([]float64, l)
	} else if l < len(*to) {
		*to = (*to)[:l]
	}

	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrFloat64) OnArrayFinished(ctx *unfoldCtx) error {
	u.cleanup(ctx)
	return nil
}

func (u *unfolderArrFloat64) append(ctx *unfoldCtx, v float64) error {
	idx := &ctx.idx
	to := u.ptr(ctx)
	if len(*to) <= idx.current {
		*to = append(*to, v)
	} else {
		(*to)[idx.current] = v
	}

	idx.current++
	return nil
}

func (u *unfolderArrIfc) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, nil)
}

func (u *unfolderArrIfc) OnBool(ctx *unfoldCtx, v bool) error { return u.append(ctx, v) }

func (u *unfolderArrIfc) OnString(ctx *unfoldCtx, v string) error { return u.append(ctx, v) }
func (u *unfolderArrIfc) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderArrIfc) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, (interface{})(v))
}

func (u *unfolderArrIfc) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, (interface{})(v))
}

func (*unfolderArrIfc) OnArrayStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	return unfoldIfcStartSubArray(ctx, l, bt)
}

func (u *unfolderArrIfc) OnChildArrayDone(ctx *unfoldCtx) error {
	v, err := unfoldIfcFinishSubArray(ctx)
	if err == nil {
		err = u.append(ctx, v)
	}
	return err
}

func (*unfolderArrIfc) OnObjectStart(ctx *unfoldCtx, l int, bt structform.BaseType) error {
	return unfoldIfcStartSubMap(ctx, l, bt)
}

func (u *unfolderArrIfc) OnChildObjectDone(ctx *unfoldCtx) error {
	v, err := unfoldIfcFinishSubMap(ctx)
	if err == nil {
		err = u.append(ctx, v)
	}
	return err
}

func (u *unfolderArrBool) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, false)
}

func (u *unfolderArrBool) OnBool(ctx *unfoldCtx, v bool) error { return u.append(ctx, v) }

func (u *unfolderArrString) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, "")
}

func (u *unfolderArrString) OnString(ctx *unfoldCtx, v string) error { return u.append(ctx, v) }
func (u *unfolderArrString) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.OnString(ctx, string(v))
}

func (u *unfolderArrUint) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrUint) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, uint(v))
}

func (u *unfolderArrUint8) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrUint8) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint8) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, uint8(v))
}

func (u *unfolderArrUint16) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrUint16) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint16) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, uint16(v))
}

func (u *unfolderArrUint32) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrUint32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, uint32(v))
}

func (u *unfolderArrUint64) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrUint64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrUint64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, uint64(v))
}

func (u *unfolderArrInt) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrInt) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, int(v))
}

func (u *unfolderArrInt8) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrInt8) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt8) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, int8(v))
}

func (u *unfolderArrInt16) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrInt16) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt16) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, int16(v))
}

func (u *unfolderArrInt32) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrInt32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, int32(v))
}

func (u *unfolderArrInt64) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrInt64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrInt64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, int64(v))
}

func (u *unfolderArrFloat32) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrFloat32) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat32) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, float32(v))
}

func (u *unfolderArrFloat64) OnNil(ctx *unfoldCtx) error {
	return u.append(ctx, 0)
}

func (u *unfolderArrFloat64) OnByte(ctx *unfoldCtx, v byte) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnUint(ctx *unfoldCtx, v uint) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnInt(ctx *unfoldCtx, v int) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnInt8(ctx *unfoldCtx, v int8) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnInt16(ctx *unfoldCtx, v int16) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnInt32(ctx *unfoldCtx, v int32) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnInt64(ctx *unfoldCtx, v int64) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.append(ctx, float64(v))
}

func (u *unfolderArrFloat64) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.append(ctx, float64(v))
}

func unfoldIfcStartSubArray(ctx *unfoldCtx, l int, baseType structform.BaseType) error {
	_, ptr, unfolder := makeArrayPtr(ctx, l, baseType)
	ctx.ptr.push(ptr) // store pointer for use in 'Finish'
	ctx.baseType.push(baseType)
	unfolder.initState(ctx, ptr)
	return ctx.unfolder.current.OnArrayStart(ctx, l, baseType)
}

func unfoldIfcFinishSubArray(ctx *unfoldCtx) (interface{}, error) {
	child := ctx.ptr.pop()
	bt := ctx.baseType.pop()
	switch bt {

	case structform.AnyType:
		value := *(*[]interface{})(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.BoolType:
		value := *(*[]bool)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.ByteType:
		value := *(*[]uint8)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Float32Type:
		value := *(*[]float32)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Float64Type:
		value := *(*[]float64)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Int16Type:
		value := *(*[]int16)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Int32Type:
		value := *(*[]int32)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Int64Type:
		value := *(*[]int64)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Int8Type:
		value := *(*[]int8)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.IntType:
		value := *(*[]int)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.StringType:
		value := *(*[]string)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Uint16Type:
		value := *(*[]uint16)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Uint32Type:
		value := *(*[]uint32)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Uint64Type:
		value := *(*[]uint64)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.Uint8Type:
		value := *(*[]uint8)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.UintType:
		value := *(*[]uint)(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	case structform.ZeroType:
		value := *(*[]interface{})(child)
		last := len(ctx.valueBuffer.arrays) - 1
		ctx.valueBuffer.arrays = ctx.valueBuffer.arrays[:last]
		return value, nil

	default:
		return nil, errTODO()
	}
}

func makeArrayPtr(ctx *unfoldCtx, l int, bt structform.BaseType) (interface{}, unsafe.Pointer, ptrUnfolder) {
	switch bt {

	case structform.AnyType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]interface{})(ptr)
		unfolder := newUnfolderArrIfc()

		return to, ptr, unfolder

	case structform.BoolType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]bool)(ptr)
		unfolder := newUnfolderArrBool()

		return to, ptr, unfolder

	case structform.ByteType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]uint8)(ptr)
		unfolder := newUnfolderArrUint8()

		return to, ptr, unfolder

	case structform.Float32Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]float32)(ptr)
		unfolder := newUnfolderArrFloat32()

		return to, ptr, unfolder

	case structform.Float64Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]float64)(ptr)
		unfolder := newUnfolderArrFloat64()

		return to, ptr, unfolder

	case structform.Int16Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]int16)(ptr)
		unfolder := newUnfolderArrInt16()

		return to, ptr, unfolder

	case structform.Int32Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]int32)(ptr)
		unfolder := newUnfolderArrInt32()

		return to, ptr, unfolder

	case structform.Int64Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]int64)(ptr)
		unfolder := newUnfolderArrInt64()

		return to, ptr, unfolder

	case structform.Int8Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]int8)(ptr)
		unfolder := newUnfolderArrInt8()

		return to, ptr, unfolder

	case structform.IntType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]int)(ptr)
		unfolder := newUnfolderArrInt()

		return to, ptr, unfolder

	case structform.StringType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]string)(ptr)
		unfolder := newUnfolderArrString()

		return to, ptr, unfolder

	case structform.Uint16Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]uint16)(ptr)
		unfolder := newUnfolderArrUint16()

		return to, ptr, unfolder

	case structform.Uint32Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]uint32)(ptr)
		unfolder := newUnfolderArrUint32()

		return to, ptr, unfolder

	case structform.Uint64Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]uint64)(ptr)
		unfolder := newUnfolderArrUint64()

		return to, ptr, unfolder

	case structform.Uint8Type:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]uint8)(ptr)
		unfolder := newUnfolderArrUint8()

		return to, ptr, unfolder

	case structform.UintType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]uint)(ptr)
		unfolder := newUnfolderArrUint()

		return to, ptr, unfolder

	case structform.ZeroType:
		idx := len(ctx.valueBuffer.arrays)
		ctx.valueBuffer.arrays = append(ctx.valueBuffer.arrays, nil)
		arrPtr := &ctx.valueBuffer.arrays[idx]
		ptr := unsafe.Pointer(arrPtr)
		to := (*[]interface{})(ptr)
		unfolder := newUnfolderArrIfc()

		return to, ptr, unfolder

	default:
		panic("invalid type code")
	}
}
