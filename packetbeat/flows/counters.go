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

package flows

import "sync"

type Var interface{}

type flagsInfo struct {
	i    int
	mask uint8
}

type Int struct {
	i int
	f flagsInfo
}

type Uint struct {
	i int
	f flagsInfo
}

type Float struct {
	i int
	f flagsInfo
}

type counterReg struct {
	mutex sync.Mutex

	ints   counterTypeReg
	uints  counterTypeReg
	floats counterTypeReg
}

type counterTypeReg struct {
	names []string
}

type flowStats struct {
	intFlags   []uint8
	uintFlags  []uint8
	floatFlags []uint8

	ints   []int64
	uints  []uint64
	floats []float64
}

func (c *Int) Add(f *Flow, delta int64) {
	ints := f.stats.ints
	if c.i < len(ints) {
		ints[c.i] += delta
		c.f.apply(f.stats.intFlags)
	}
}

func (c *Int) Set(f *Flow, value int64) {
	ints := f.stats.ints
	if c.i < len(ints) {
		ints[c.i] = value
		c.f.apply(f.stats.intFlags)
	}
}

func (c *Uint) Add(f *Flow, delta uint64) {
	uints := f.stats.uints
	if c.i < len(uints) {
		uints[c.i] += delta
		c.f.apply(f.stats.uintFlags)
	}
}

func (c *Uint) Set(f *Flow, value uint64) {
	uints := f.stats.uints
	if c.i < len(uints) {
		uints[c.i] = value
		c.f.apply(f.stats.uintFlags)
	}
}

func (c *Float) Add(f *Flow, delta float64) {
	floats := f.stats.floats
	if c.i < len(floats) {
		floats[c.i] += delta
		c.f.apply(f.stats.floatFlags)
	}
}

func (c *Float) Set(f *Flow, value float64) {
	floats := f.stats.floats
	if c.i < len(floats) {
		floats[c.i] = value
		c.f.apply(f.stats.floatFlags)
	}
}

func (c *counterReg) newInt(name string) (*Int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	i, err := c.ints.reg(name)
	if err != nil {
		return nil, err
	}
	return &Int{i, makeFlagsInfo(i)}, nil
}

func (c *counterReg) newUint(name string) (*Uint, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	i, err := c.uints.reg(name)
	if err != nil {
		return nil, err
	}
	return &Uint{i, makeFlagsInfo(i)}, nil
}

func (c *counterReg) newFloat(name string) (*Float, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	i, err := c.floats.reg(name)
	if err != nil {
		return nil, err
	}
	return &Float{i, makeFlagsInfo(i)}, nil
}

// XXX:
//  - error on index > int max
//  - error if already in use
func (reg *counterTypeReg) reg(name string) (int, error) {
	debugf("register flow counter: %v", name)

	i := len(reg.names)
	reg.names = append(reg.names, name)
	return i, nil
}

func (reg *counterTypeReg) getNames() []string {
	return reg.names
}

func newFlowStats(reg *counterReg) *flowStats {
	s := &flowStats{}
	s.init(reg)
	return s
}

func (s *flowStats) init(reg *counterReg) {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	nInts := len(reg.ints.names)
	nUints := len(reg.uints.names)
	nFloats := len(reg.floats.names)

	s.ints = make([]int64, nInts)
	s.uints = make([]uint64, nUints)
	s.floats = make([]float64, nFloats)

	s.intFlags = make([]uint8, (nInts+7)/8)
	s.uintFlags = make([]uint8, (nUints+7)/8)
	s.floatFlags = make([]uint8, (nFloats+7)/8)
}

func makeFlagsInfo(i int) flagsInfo {
	return flagsInfo{
		i:    i / 8,
		mask: 1 << uint(i%8),
	}
}

func (f *flagsInfo) apply(flags []uint8) {
	flags[f.i] |= f.mask
}
