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

// Package atomic provides common primitive types with atomic accessors.
package atomic

import a "sync/atomic"

// Bool provides an atomic boolean type.
type Bool struct{ u Uint32 }

// Int32 provides an atomic int32 type.
type Int32 struct{ value int32 }

// Int64 provides an atomic int64 type.
type Int64 struct{ value int64 }

// Uint32 provides an atomic uint32 type.
type Uint32 struct{ value uint32 }

// Uint64 provides an atomic uint64 type.
type Uint64 struct{ value uint64 }

func MakeBool(v bool) Bool             { return Bool{MakeUint32(encBool(v))} }
func NewBool(v bool) *Bool             { return &Bool{MakeUint32(encBool(v))} }
func (b *Bool) Load() bool             { return b.u.Load() == 1 }
func (b *Bool) Store(v bool)           { b.u.Store(encBool(v)) }
func (b *Bool) Swap(new bool) bool     { return b.u.Swap(encBool(new)) == 1 }
func (b *Bool) CAS(old, new bool) bool { return b.u.CAS(encBool(old), encBool(new)) }

func MakeInt32(v int32) Int32            { return Int32{v} }
func NewInt32(v int32) *Int32            { return &Int32{v} }
func (i *Int32) Load() int32             { return a.LoadInt32(&i.value) }
func (i *Int32) Store(v int32)           { a.StoreInt32(&i.value, v) }
func (i *Int32) Swap(new int32) int32    { return a.SwapInt32(&i.value, new) }
func (i *Int32) Add(delta int32) int32   { return a.AddInt32(&i.value, delta) }
func (i *Int32) Sub(delta int32) int32   { return a.AddInt32(&i.value, -delta) }
func (i *Int32) Inc() int32              { return i.Add(1) }
func (i *Int32) Dec() int32              { return i.Add(-1) }
func (i *Int32) CAS(old, new int32) bool { return a.CompareAndSwapInt32(&i.value, old, new) }

func MakeInt64(v int64) Int64            { return Int64{v} }
func NewInt64(v int64) *Int64            { return &Int64{v} }
func (i *Int64) Load() int64             { return a.LoadInt64(&i.value) }
func (i *Int64) Store(v int64)           { a.StoreInt64(&i.value, v) }
func (i *Int64) Swap(new int64) int64    { return a.SwapInt64(&i.value, new) }
func (i *Int64) Add(delta int64) int64   { return a.AddInt64(&i.value, delta) }
func (i *Int64) Sub(delta int64) int64   { return a.AddInt64(&i.value, -delta) }
func (i *Int64) Inc() int64              { return i.Add(1) }
func (i *Int64) Dec() int64              { return i.Add(-1) }
func (i *Int64) CAS(old, new int64) bool { return a.CompareAndSwapInt64(&i.value, old, new) }

func MakeUint32(v uint32) Uint32           { return Uint32{v} }
func NewUint32(v uint32) *Uint32           { return &Uint32{v} }
func (u *Uint32) Load() uint32             { return a.LoadUint32(&u.value) }
func (u *Uint32) Store(v uint32)           { a.StoreUint32(&u.value, v) }
func (u *Uint32) Swap(new uint32) uint32   { return a.SwapUint32(&u.value, new) }
func (u *Uint32) Add(delta uint32) uint32  { return a.AddUint32(&u.value, delta) }
func (u *Uint32) Sub(delta uint32) uint32  { return a.AddUint32(&u.value, ^uint32(delta-1)) }
func (u *Uint32) Inc() uint32              { return u.Add(1) }
func (u *Uint32) Dec() uint32              { return u.Add(^uint32(0)) }
func (u *Uint32) CAS(old, new uint32) bool { return a.CompareAndSwapUint32(&u.value, old, new) }

func MakeUint64(v uint64) Uint64           { return Uint64{v} }
func NewUint64(v uint64) *Uint64           { return &Uint64{v} }
func (u *Uint64) Load() uint64             { return a.LoadUint64(&u.value) }
func (u *Uint64) Store(v uint64)           { a.StoreUint64(&u.value, v) }
func (u *Uint64) Swap(new uint64) uint64   { return a.SwapUint64(&u.value, new) }
func (u *Uint64) Add(delta uint64) uint64  { return a.AddUint64(&u.value, delta) }
func (u *Uint64) Sub(delta uint64) uint64  { return a.AddUint64(&u.value, ^uint64(delta-1)) }
func (u *Uint64) Inc() uint64              { return u.Add(1) }
func (u *Uint64) Dec() uint64              { return u.Add(^uint64(0)) }
func (u *Uint64) CAS(old, new uint64) bool { return a.CompareAndSwapUint64(&u.value, old, new) }

func encBool(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}
