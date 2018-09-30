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

package gotype

import (
	"reflect"
	"sync"
	"unsafe"

	structform "github.com/elastic/go-structform"
)

type Unfolder struct {
	unfoldCtx
}

type unfoldCtx struct {
	opts options

	// buf buffer

	unfolder unfolderStack
	value    reflectValueStack
	baseType structformTypeStack
	ptr      ptrStack
	key      keyStack
	idx      idxStack

	keyCache symbolCache

	valueBuffer unfoldBuf
}

type unfoldBuf struct {
	arrays       [][]byte
	mapPrimitive []map[string]byte
	mapAny       []map[string]interface{}
}

type ptrUnfolder interface {
	initState(*unfoldCtx, unsafe.Pointer)
}

type reflUnfolder interface {
	initState(*unfoldCtx, reflect.Value)
}

type unfolder interface {
	// primitives
	OnNil(*unfoldCtx) error
	OnBool(*unfoldCtx, bool) error
	OnString(*unfoldCtx, string) error
	OnStringRef(*unfoldCtx, []byte) error
	OnInt8(*unfoldCtx, int8) error
	OnInt16(*unfoldCtx, int16) error
	OnInt32(*unfoldCtx, int32) error
	OnInt64(*unfoldCtx, int64) error
	OnInt(*unfoldCtx, int) error
	OnByte(*unfoldCtx, byte) error
	OnUint8(*unfoldCtx, uint8) error
	OnUint16(*unfoldCtx, uint16) error
	OnUint32(*unfoldCtx, uint32) error
	OnUint64(*unfoldCtx, uint64) error
	OnUint(*unfoldCtx, uint) error
	OnFloat32(*unfoldCtx, float32) error
	OnFloat64(*unfoldCtx, float64) error

	// array types
	OnArrayStart(*unfoldCtx, int, structform.BaseType) error
	OnArrayFinished(*unfoldCtx) error
	OnChildArrayDone(*unfoldCtx) error

	// object types
	OnObjectStart(*unfoldCtx, int, structform.BaseType) error
	OnObjectFinished(*unfoldCtx) error
	OnKey(*unfoldCtx, string) error
	OnKeyRef(*unfoldCtx, []byte) error
	OnChildObjectDone(*unfoldCtx) error
}

type typeUnfoldRegistry struct {
	mu sync.RWMutex
	m  map[reflect.Type]reflUnfolder
}

var unfoldRegistry = newTypeUnfoldRegistry()

func NewUnfolder(to interface{}) (*Unfolder, error) {
	u := &Unfolder{}
	u.opts = options{tag: "struct"}

	u.unfolder.init(&unfolderNoTarget{})
	u.value.init(reflect.Value{})
	u.ptr.init()
	u.key.init()
	u.idx.init()
	u.baseType.init()
	u.valueBuffer.init()

	// TODO: make allocation buffer size configurable
	// u.buf.init(1024)

	if to != nil {
		err := u.SetTarget(to)
		if err != nil {
			return nil, err
		}
	}

	return u, nil
}

func (u *Unfolder) EnableKeyCache(max int) {
	u.keyCache.init(max)
}

// Reset reinitializes the unfolder and removes all references to the target
// object. Use Reset if the unfolder is re-used and the target changed.
// References to the target can prevent the garbage collector from collecting
// the target after processing. Use Reset to set the target to `nil`.
// SetTarget must be called after Reset and before another Unfold operation.
func (u *Unfolder) Reset() {
	u.SetTarget(nil)
}

func (u *Unfolder) SetTarget(to interface{}) error {
	ctx := &u.unfoldCtx

	if to == nil {
		// reset internal states on nil
		u.unfolder.init(&unfolderNoTarget{})
		u.value.init(reflect.Value{})
		u.ptr.init()
		u.key.init()
		u.idx.init()
		u.baseType.init()
		u.valueBuffer.reset()

		return nil
	}

	if ptr, u := lookupGoTypeUnfolder(to); u != nil {
		u.initState(ctx, ptr)
		return nil
	}

	t := reflect.TypeOf(to)
	if t.Kind() != reflect.Ptr {
		return errRequiresPointer
	}

	ru, err := lookupReflUnfolder(&u.unfoldCtx, t)
	if err != nil {
		return err
	}
	if ru != nil {
		ru.initState(ctx, reflect.ValueOf(to))
		return nil
	}

	return errUnsupported
}

func (u *unfoldCtx) OnObjectStart(len int, baseType structform.BaseType) error {
	return u.unfolder.current.OnObjectStart(u, len, baseType)
}

func (u *unfoldCtx) OnObjectFinished() error {
	lBefore := len(u.unfolder.stack) + 1

	if err := u.unfolder.current.OnObjectFinished(u); err != nil {
		return err
	}

	lAfter := len(u.unfolder.stack) + 1
	if old := u.unfolder.current; lAfter > 1 && lBefore != lAfter {
		return old.OnChildObjectDone(u)
	}

	return nil
}

func (u *unfoldCtx) OnKey(s string) error {
	return u.unfolder.current.OnKey(u, s)
}

func (u *unfoldCtx) OnKeyRef(s []byte) error {
	return u.unfolder.current.OnKeyRef(u, s)
}

func (u *unfoldCtx) OnArrayStart(len int, baseType structform.BaseType) error {
	return u.unfolder.current.OnArrayStart(u, len, baseType)
}

func (u *unfoldCtx) OnArrayFinished() error {
	lBefore := len(u.unfolder.stack) + 1

	if err := u.unfolder.current.OnArrayFinished(u); err != nil {
		return err
	}

	lAfter := len(u.unfolder.stack) + 1
	if old := u.unfolder.current; lAfter > 1 && lBefore != lAfter {
		return old.OnChildArrayDone(u)
	}

	return nil
}

func (u *unfoldCtx) OnNil() error {
	return u.unfolder.current.OnNil(u)
}

func (u *unfoldCtx) OnBool(b bool) error {
	return u.unfolder.current.OnBool(u, b)
}

func (u *unfoldCtx) OnString(s string) error {
	return u.unfolder.current.OnString(u, s)
}

func (u *unfoldCtx) OnStringRef(s []byte) error {
	return u.unfolder.current.OnStringRef(u, s)
}

func (u *unfoldCtx) OnInt8(i int8) error {
	return u.unfolder.current.OnInt8(u, i)
}

func (u *unfoldCtx) OnInt16(i int16) error {
	return u.unfolder.current.OnInt16(u, i)
}

func (u *unfoldCtx) OnInt32(i int32) error {
	return u.unfolder.current.OnInt32(u, i)
}

func (u *unfoldCtx) OnInt64(i int64) error {
	return u.unfolder.current.OnInt64(u, i)
}

func (u *unfoldCtx) OnInt(i int) error {
	return u.unfolder.current.OnInt(u, i)
}

func (u *unfoldCtx) OnByte(b byte) error {
	return u.unfolder.current.OnByte(u, b)
}

func (u *unfoldCtx) OnUint8(v uint8) error {
	return u.unfolder.current.OnUint8(u, v)
}

func (u *unfoldCtx) OnUint16(v uint16) error {
	return u.unfolder.current.OnUint16(u, v)
}

func (u *unfoldCtx) OnUint32(v uint32) error {
	return u.unfolder.current.OnUint32(u, v)
}

func (u *unfoldCtx) OnUint64(v uint64) error {
	return u.unfolder.current.OnUint64(u, v)
}

func (u *unfoldCtx) OnUint(v uint) error {
	return u.unfolder.current.OnUint(u, v)
}

func (u *unfoldCtx) OnFloat32(f float32) error {
	return u.unfolder.current.OnFloat32(u, f)
}

func (u *unfoldCtx) OnFloat64(f float64) error {
	return u.unfolder.current.OnFloat64(u, f)
}

func newTypeUnfoldRegistry() *typeUnfoldRegistry {
	return &typeUnfoldRegistry{m: map[reflect.Type]reflUnfolder{}}
}

func (r *typeUnfoldRegistry) find(t reflect.Type) reflUnfolder {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.m[t]
}

func (r *typeUnfoldRegistry) set(t reflect.Type, f reflUnfolder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[t] = f
}

func makeUnfoldBuf() unfoldBuf {
	return unfoldBuf{
		arrays:       make([][]byte, 0, 4),
		mapPrimitive: make([]map[string]byte, 0, 1),
		mapAny:       make([]map[string]interface{}, 0, 4),
	}
}

func (u *unfoldBuf) init() {
	*u = makeUnfoldBuf()
}

func (u *unfoldBuf) reset() {
	u.arrays = u.arrays[:0]
	u.mapPrimitive = u.mapPrimitive[:0]
	u.mapAny = u.mapAny[:0]
}
