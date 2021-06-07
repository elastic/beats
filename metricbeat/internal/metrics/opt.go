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

package metrics

import (
	"github.com/elastic/go-structform"
	"github.com/elastic/go-structform/gotype"
)

// OptUint is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type OptUint struct {
	exists bool
	value  uint64
}

// NewUint returns a new uint wrapper
func NewUint() OptUint {
	return OptUint{
		exists: false,
		value:  0,
	}
}

// NewUintFrom returns a OptUint wrapper
func NewUintFrom(i uint64) OptUint {
	return OptUint{
		exists: true,
		value:  i,
	}
}

// None marks the Uint as not having a value.
func (opt *OptUint) None() {
	opt.exists = false
}

// Exists returns true if the underlying value is valid
func (opt OptUint) Exists() bool {
	return opt.exists
}

// Some Sets a valid value inside the OptUint
func (opt *OptUint) Some(i uint64) {
	opt.value = i
	opt.exists = true
}

// ValueOrZero returns the stored value, or zero
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt OptUint) ValueOrZero() uint64 {
	if opt.exists {
		return opt.value
	}
	return 0
}

// structform implementation for the OptUint types

type OptUnfolder struct {
	gotype.BaseUnfoldState
	to *OptUint
}

// UnfoldOpt is the unfolder implementation for OptUint
func UnfoldOpt(to *OptUint) gotype.UnfoldState {
	return &OptUnfolder{to: to}
}

func FoldOpt(in *OptUint, v structform.ExtVisitor) error {
	var value uint64
	if in.exists == true {
		value = in.value
		v.OnUint64(value)
	} else {
		v.OnNil()
	}
	return nil
}

func (u *OptUnfolder) OnUint64(ctx gotype.UnfoldCtx, in uint64) error {
	defer ctx.Done()
	u.to = &OptUint{exists: true, value: in}

	return nil
}

func (u *OptUnfolder) OnNil(ctx gotype.UnfoldCtx) error {
	defer ctx.Done()
	u.to = &OptUint{exists: false, value: 0}

	return nil
}
