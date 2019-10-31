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
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	structform "github.com/elastic/go-structform"
	stunsafe "github.com/elastic/go-structform/internal/unsafe"
)

// Expander supports the creation of an UnfoldState for handling the unfolding
// into the current value.
type Expander interface {
	Expand() UnfoldState
}

// UnfoldCtx provides access to the shared unfolding stack. It is used with
// UnfoldState, so to implement very custom parsing.
type UnfoldCtx interface {
	// Done signals the context that the current state is finished.
	// The current state will be removed from the stack and processing continues
	// with the current state.
	Done()

	// Cont replaces the current state with the new state. All unfolding
	// will continue with the new state
	Cont(st UnfoldState)

	// Push adds a new parsing state on top of the state stack. Unfolding
	// will continue with the new state.
	Push(st UnfoldState)
}

// UnfoldState defines a custom user defined unfolder.
// When unfolding a stack is used to keep state. The UnfoldCtx provides
// methods to manipluate the stack of active unfolders.
// The current UnfoldState will be used as long as it has not been remove or replaced
// using one of the UnfoldCtx control methods.
type UnfoldState interface {
	// primitives
	OnNil(ctx UnfoldCtx) error
	OnBool(ctx UnfoldCtx, b bool) error
	OnString(ctx UnfoldCtx, str string) error
	OnInt(ctx UnfoldCtx, i int64) error
	OnUint(ctx UnfoldCtx, u uint64) error
	OnFloat(ctx UnfoldCtx, f float64) error

	// array types
	OnArrayStart(ctx UnfoldCtx, length int, bt structform.BaseType) error
	OnArrayFinished(ctx UnfoldCtx) error

	// object types
	OnObjectStart(ctx UnfoldCtx, length int, bt structform.BaseType) error
	OnObjectFinished(ctx UnfoldCtx) error
	OnKey(ctx UnfoldCtx, key string) error
}

// BaseUnfoldState implements UnfoldState, but returns an error for every
// callback possible.
// One case embedd BaseUnfoldState in a custom struct, so to reduce the number
// of methods to implement.
type BaseUnfoldState struct{}

type stateUnfolder struct {
	unfolder UnfoldState
}

type unfoldUserStateInit struct {
	fn unfoldStateInitFn
}

type unfoldStateInitFn func(unsafe.Pointer) UnfoldState

type unfoldExpanderInit struct{}

func makeUserUnfolder(fn reflect.Value) (target reflect.Type, unfolder reflUnfolder, err error) {
	t := fn.Type()

	if fn.Kind() != reflect.Func {
		return nil, nil, errors.New("function type required")
	}

	switch {
	case t.NumIn() == 2 && t.NumOut() == 1:
		unfolder, err = makeUserPrimitiveUnfolder(fn)
	case t.NumIn() == 1 && t.NumOut() == 1:
		unfolder, err = makeUserStateUnfolder(fn)
	case t.NumIn() == 1 && t.NumOut() == 2:
		unfolder, err = makeUserProcessingUnfolder(fn)
	default:
		return nil, nil, fmt.Errorf("invalid number of arguments in unfolder: %v", fn)
	}

	return t.In(0), unfolder, err
}

func makeUserPrimitiveUnfolder(fn reflect.Value) (reflUnfolder, error) {
	t := fn.Type()

	if fn.Kind() != reflect.Func {
		return nil, errors.New("function type required")
	}

	if t.NumIn() != 2 {
		return nil, fmt.Errorf("function '%v' must accept 2 arguments", fn)
	}
	if t.NumOut() != 1 || (t.NumOut() > 0 && t.Out(0) != tError) {
		return nil, fmt.Errorf("function '%v' does not return errors", fn)
	}

	ta0 := t.In(0)
	if ta0.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("first argument in function '%v' must be a pointer", t.Name())
	}

	constr := lookupUserPrimitiveConstructor(t.In(1))
	if constr == nil {
		return nil, fmt.Errorf("%v is no supported primitive type", t.In(1))
	}

	unfolder := constr(fn)
	return liftGoUnfolder(unfolder), nil
}

func makeUserProcessingUnfolder(fn reflect.Value) (reflUnfolder, error) {
	if err := checkProcessingUnfolder(fn); err != nil {
		return nil, err
	}

	return liftGoUnfolder(&unfolderUserProcessingInit{
		fnInit: *((*userProcessingInitFn)(stunsafe.UnsafeFnPtr(fn))),
	}), nil
}

func checkProcessingUnfolder(fn reflect.Value) error {
	if fn.Kind() != reflect.Func {
		return fmt.Errorf("processing unfolder '%v' is no function", fn)
	}

	t := fn.Type()

	// check input
	if t.NumIn() != 1 {
		return fmt.Errorf("processing unfolder '%v' must accept one target argument", fn)
	}
	in := t.In(0)
	if in.Kind() != reflect.Ptr {
		return fmt.Errorf("processing unfolder '%v' target argument must be a pointer", fn)
	}

	// check returns
	if t.NumOut() != 2 {
		return fmt.Errorf("processing unfolder '%v' must return 2 values", fn)
	}
	if t.Out(0) != tInterface {
		return fmt.Errorf("processing unfolder '%v' must return interface{} as first value", fn)
	}
	proc := t.Out(1)
	if proc.Kind() != reflect.Func {
		return fmt.Errorf("processing unfolder '%v' second return must be a function", fn)
	}

	// check processing function input
	if proc.NumIn() != 2 {
		return fmt.Errorf("processing function of '%v' must accept 2 arguments", fn)
	}
	if proc.In(0) != in {
		return fmt.Errorf("processing function of '%v' must accept the target type '%v'", fn, in)
	}
	if proc.In(1) != tInterface {
		return fmt.Errorf("processing function of '%v' must accept interface{} as second argument", fn)
	}

	// check processing function output
	if proc.NumOut() != 1 {
		return fmt.Errorf("processing function of '%v' must return exactly one value", fn)
	}
	if proc.Out(0) != tError {
		return fmt.Errorf("processing function of '%v' must return an error value", fn)
	}

	return nil
}

func makeUserStateUnfolder(fn reflect.Value) (reflUnfolder, error) {
	if err := checkUserStateUnfolder(fn); err != nil {
		return nil, err
	}

	return liftGoUnfolder(&unfoldUserStateInit{
		fn: *((*unfoldStateInitFn)(stunsafe.UnsafeFnPtr(fn))),
	}), nil

}

func checkUserStateUnfolder(fn reflect.Value) error {
	if fn.Kind() != reflect.Func {
		return fmt.Errorf("state unfolder '%v' is no function", fn)
	}

	t := fn.Type()

	// check input
	if t.NumIn() != 1 {
		return fmt.Errorf("state unfolder '%v' must accept one target argument", fn)
	}
	in := t.In(0)
	if in.Kind() != reflect.Ptr {
		return fmt.Errorf("state unfolder '%v' target argument must be a pointer", fn)
	}

	if t.NumOut() != 1 || (t.NumOut() > 0 && t.Out(0) != tUnfoldState) {
		return fmt.Errorf("function '%v' does not return UnfoldState type", fn)
	}

	return nil
}

var _unfoldExpanderInit = &unfoldExpanderInit{}

func newExpanderInit() reflUnfolder {
	return _unfoldExpanderInit
}

func (*unfoldExpanderInit) initState(ctx *unfoldCtx, val reflect.Value) {
	st := val.Interface().(Expander).Expand()
	ctx.Push(st)
}

func (ctx *unfoldCtx) Done() {
	ctx.unfolder.pop()
}

func (ctx *unfoldCtx) Cont(st UnfoldState) {
	ctx.unfolder.pop()
	ctx.unfolder.push(&stateUnfolder{st})
}

func (ctx *unfoldCtx) Push(st UnfoldState) {
	ctx.unfolder.push(&stateUnfolder{st})
}

func (u *unfoldUserStateInit) initState(ctx *unfoldCtx, ptr unsafe.Pointer) {
	st := u.fn(ptr)
	ctx.Push(st)
}

func (u *stateUnfolder) OnNil(ctx *unfoldCtx) error              { return u.unfolder.OnNil(ctx) }
func (u *stateUnfolder) OnBool(ctx *unfoldCtx, v bool) error     { return u.unfolder.OnBool(ctx, v) }
func (u *stateUnfolder) OnString(ctx *unfoldCtx, v string) error { return u.unfolder.OnString(ctx, v) }
func (u *stateUnfolder) OnStringRef(ctx *unfoldCtx, v []byte) error {
	return u.unfolder.OnString(ctx, string(v))
}
func (u *stateUnfolder) OnInt8(ctx *unfoldCtx, v int8) error   { return u.unfolder.OnInt(ctx, int64(v)) }
func (u *stateUnfolder) OnInt16(ctx *unfoldCtx, v int16) error { return u.unfolder.OnInt(ctx, int64(v)) }
func (u *stateUnfolder) OnInt32(ctx *unfoldCtx, v int32) error { return u.unfolder.OnInt(ctx, int64(v)) }
func (u *stateUnfolder) OnInt64(ctx *unfoldCtx, v int64) error { return u.unfolder.OnInt(ctx, int64(v)) }
func (u *stateUnfolder) OnInt(ctx *unfoldCtx, v int) error     { return u.unfolder.OnUint(ctx, uint64(v)) }
func (u *stateUnfolder) OnByte(ctx *unfoldCtx, v byte) error   { return u.unfolder.OnUint(ctx, uint64(v)) }
func (u *stateUnfolder) OnUint8(ctx *unfoldCtx, v uint8) error {
	return u.unfolder.OnUint(ctx, uint64(v))
}
func (u *stateUnfolder) OnUint16(ctx *unfoldCtx, v uint16) error {
	return u.unfolder.OnUint(ctx, uint64(v))
}
func (u *stateUnfolder) OnUint32(ctx *unfoldCtx, v uint32) error {
	return u.unfolder.OnUint(ctx, uint64(v))
}
func (u *stateUnfolder) OnUint64(ctx *unfoldCtx, v uint64) error {
	return u.unfolder.OnUint(ctx, uint64(v))
}
func (u *stateUnfolder) OnUint(ctx *unfoldCtx, v uint) error { return u.unfolder.OnUint(ctx, uint64(v)) }
func (u *stateUnfolder) OnFloat32(ctx *unfoldCtx, v float32) error {
	return u.unfolder.OnFloat(ctx, float64(v))
}
func (u *stateUnfolder) OnFloat64(ctx *unfoldCtx, v float64) error {
	return u.unfolder.OnFloat(ctx, float64(v))
}
func (u *stateUnfolder) OnArrayStart(ctx *unfoldCtx, N int, bt structform.BaseType) error {
	return u.unfolder.OnArrayStart(ctx, N, bt)
}
func (u *stateUnfolder) OnArrayFinished(ctx *unfoldCtx) error  { return u.unfolder.OnArrayFinished(ctx) }
func (u *stateUnfolder) OnChildArrayDone(ctx *unfoldCtx) error { return nil }
func (u *stateUnfolder) OnObjectStart(ctx *unfoldCtx, N int, bt structform.BaseType) error {
	return u.unfolder.OnObjectStart(ctx, N, bt)
}
func (u *stateUnfolder) OnObjectFinished(ctx *unfoldCtx) error {
	return u.unfolder.OnObjectFinished(ctx)
}
func (u *stateUnfolder) OnKey(ctx *unfoldCtx, v string) error { return u.unfolder.OnKey(ctx, v) }
func (u *stateUnfolder) OnKeyRef(ctx *unfoldCtx, v []byte) error {
	return u.unfolder.OnKey(ctx, string(v))
}
func (u *stateUnfolder) OnChildObjectDone(ctx *unfoldCtx) error { return nil }

func (*BaseUnfoldState) OnNil(ctx UnfoldCtx) error                { return errUnexpectedNil }
func (*BaseUnfoldState) OnBool(ctx UnfoldCtx, b bool) error       { return errUnexpectedBool }
func (*BaseUnfoldState) OnString(ctx UnfoldCtx, str string) error { return errUnexpectedString }
func (*BaseUnfoldState) OnInt(ctx UnfoldCtx, i int64) error       { return errUnexpectedNum }
func (*BaseUnfoldState) OnUint(ctx UnfoldCtx, u uint64) error     { return errUnexpectedNum }
func (*BaseUnfoldState) OnFloat(ctx UnfoldCtx, f float64) error   { return errUnexpectedNum }
func (*BaseUnfoldState) OnArrayStart(ctx UnfoldCtx, length int, bt structform.BaseType) error {
	return errUnexpectedArrayStart
}
func (*BaseUnfoldState) OnArrayFinished(ctx UnfoldCtx) error { return errUnexpectedArrayEnd }
func (*BaseUnfoldState) OnObjectStart(ctx UnfoldCtx, length int, bt structform.BaseType) error {
	return errUnexpectedObjectStart
}
func (*BaseUnfoldState) OnObjectFinished(ctx UnfoldCtx) error  { return errUnexpectedObjectEnd }
func (*BaseUnfoldState) OnKey(ctx UnfoldCtx, key string) error { return errUnexpectedString }
