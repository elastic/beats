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

// +build 386 arm mips mipsle

package atomic

// atomic Uint/Int for 32bit systems

// Uint provides an architecture specific atomic uint.
type Uint struct{ a Uint32 }

// Int provides an architecture specific atomic uint.
type Int struct{ a Int32 }

func MakeUint(v uint) Uint             { return Uint{MakeUint32(uint32(v))} }
func NewUint(v uint) *Uint             { return &Uint{MakeUint32(uint32(v))} }
func (u *Uint) Load() uint             { return uint(u.a.Load()) }
func (u *Uint) Store(v uint)           { u.a.Store(uint32(v)) }
func (u *Uint) Swap(new uint) uint     { return uint(u.a.Swap(uint32(new))) }
func (u *Uint) Add(delta uint) uint    { return uint(u.a.Add(uint32(delta))) }
func (u *Uint) Sub(delta uint) uint    { return uint(u.a.Add(uint32(-delta))) }
func (u *Uint) Inc() uint              { return uint(u.a.Inc()) }
func (u *Uint) Dec() uint              { return uint(u.a.Dec()) }
func (u *Uint) CAS(old, new uint) bool { return u.a.CAS(uint32(old), uint32(new)) }

func MakeInt(v int) Int              { return Int{MakeInt32(int32(v))} }
func NewInt(v int) *Int              { return &Int{MakeInt32(int32(v))} }
func (i *Int) Load() int             { return int(i.a.Load()) }
func (i *Int) Store(v int)           { i.a.Store(int32(v)) }
func (i *Int) Swap(new int) int      { return int(i.a.Swap(int32(new))) }
func (i *Int) Add(delta int) int     { return int(i.a.Add(int32(delta))) }
func (i *Int) Sub(delta int) int     { return int(i.a.Add(int32(-delta))) }
func (i *Int) Inc() int              { return int(i.a.Inc()) }
func (i *Int) Dec() int              { return int(i.a.Dec()) }
func (i *Int) CAS(old, new int) bool { return i.a.CAS(int32(old), int32(new)) }
