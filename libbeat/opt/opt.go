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

package opt

import "github.com/elastic/go-structform"

// ZeroInterface is a type interface for cases where we need to cast from a void pointer
type ZeroInterface interface {
	IsZero() bool
}

// Int

// Int is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type Int struct {
	exists bool
	value  int
}

// NewIntNone returns a new OptUint wrapper
func NewIntNone() Int {
	return Int{
		exists: false,
		value:  0,
	}
}

// IntWith returns a new OptUint wrapper with a given int
func IntWith(i int) Int {
	return Int{
		exists: true,
		value:  i,
	}
}

// IsZero returns true if the underlying value nil
func (opt Int) IsZero() bool {
	return !opt.exists
}

// Exists returns true if the underlying value exists
func (opt Int) Exists() bool {
	return opt.exists
}

// ValueOr returns the stored value, or a given int
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt Int) ValueOr(i int) int {
	if opt.exists {
		return opt.value
	}
	return i
}

// Fold implements the folder interface for OptUint
func (opt *Int) Fold(v structform.ExtVisitor) error {
	if opt.exists {
		value := opt.value
		err := v.OnInt(value)
		if err != nil {
			return err
		}
	} else {
		err := v.OnNil()
		if err != nil {
			return err
		}
	}
	return nil
}

// Uint

// Uint is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type Uint struct {
	exists bool
	value  uint64
}

// NewUintNone returns a new OptUint wrapper
func NewUintNone() Uint {
	return Uint{
		exists: false,
		value:  0,
	}
}

// UintWith returns a new OptUint wrapper with a given int
func UintWith(i uint64) Uint {
	return Uint{
		exists: true,
		value:  i,
	}
}

// IsZero returns true if the underlying value nil
func (opt Uint) IsZero() bool {
	return !opt.exists
}

// Exists returns true if the underlying value exists
func (opt Uint) Exists() bool {
	return opt.exists
}

// ValueOr returns the stored value, or a given int
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt Uint) ValueOr(i uint64) uint64 {
	if opt.exists {
		return opt.value
	}
	return i
}

// MultUint64OrNone or will multiply the existing Uint value by a supplied uint64, and return None if either the Uint is none, or the supplied uint64 is zero.
func (opt Uint) MultUint64OrNone(i uint64) Uint {
	if !opt.exists {
		return opt
	}
	if i == 0 {
		return Uint{exists: false}
	}
	return Uint{exists: true, value: opt.value * i}
}

// SubtractOrNone will subtract the existing uint with the supplied uint64 value. If this would result in a value invalid for a uint (ie, a negative number), return None
func (opt Uint) SubtractOrNone(i Uint) Uint {
	if !opt.exists || !i.Exists() {
		return opt
	}

	if i.ValueOr(0) > opt.value {
		return Uint{exists: false}
	}

	return Uint{exists: true, value: opt.value - i.ValueOr(0)}
}

// SumOptUint sums a list of OptUint values
func SumOptUint(opts ...Uint) uint64 {
	var sum uint64
	for _, opt := range opts {
		sum += opt.ValueOr(0)
	}
	return sum
}

// Fold implements the folder interface for OptUint
func (opt *Uint) Fold(v structform.ExtVisitor) error {
	if opt.exists {
		value := opt.value
		err := v.OnUint64(value)
		if err != nil {
			return err
		}
	} else {
		err := v.OnNil()
		if err != nil {
			return err
		}
	}
	return nil
}

// Float

// Float is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type Float struct {
	exists bool
	value  float64
}

// NewFloatNone returns a new uint wrapper
func NewFloatNone() Float {
	return Float{
		exists: false,
		value:  0,
	}
}

// FloatWith returns a new uint wrapper for the specified value
func FloatWith(f float64) Float {
	return Float{
		exists: true,
		value:  f,
	}
}

// IsZero returns true if the underlying value nil
func (opt Float) IsZero() bool {
	return !opt.exists
}

// Exists returns true if the underlying value exists
func (opt Float) Exists() bool {
	return opt.exists
}

// ValueOr returns the stored value, or zero
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt Float) ValueOr(f float64) float64 {
	if opt.exists {
		return opt.value
	}
	return f
}

// Fold implements the folder interface for OptUint
func (opt *Float) Fold(v structform.ExtVisitor) error {
	if opt.exists {
		value := opt.value
		err := v.OnFloat64(value)
		if err != nil {
			return err
		}
	} else {
		err := v.OnNil()
		if err != nil {
			return err
		}
	}
	return nil
}
