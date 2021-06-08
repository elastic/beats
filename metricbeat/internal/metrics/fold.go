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

// OptUintUnfolder is the stateful contain for the unfolder
type OptUintUnfolder struct {
	gotype.BaseUnfoldState
	to *OptUint
}

// FoldOptUint is a helper for structform's Fold() function
// pass this to gotype.NewIterator
func FoldOptUint(in *OptUint, v structform.ExtVisitor) error {
	if in.exists == true {
		value := in.value
		v.OnUint64(value)
	} else {
		v.OnNil()
	}
	return nil
}

// UnfoldOptUint is a helper function for structform's Fold() function
// Pass this to gotype.NewIterator
func UnfoldOptUint(to *OptUint) gotype.UnfoldState {
	return &OptUintUnfolder{to: to}
}

// OnUint64 Folds Uint64 values
func (u *OptUintUnfolder) OnUint64(ctx gotype.UnfoldCtx, in uint64) error {
	defer ctx.Done()
	u.to = &OptUint{exists: true, value: in}

	return nil
}

// OnNil Folds nil values
func (u *OptUintUnfolder) OnNil(ctx gotype.UnfoldCtx) error {
	defer ctx.Done()
	u.to = &OptUint{exists: false, value: 0}

	return nil
}

// FoldOptFloat is a helper for structform's Fold() function
// pass this to gotype.NewIterator
func FoldOptFloat(in *OptFloat, v structform.ExtVisitor) error {
	if in.exists == true {
		value := in.value
		v.OnFloat64(value)
	} else {
		v.OnNil()
	}
	return nil
}

// OptFloatUnfolder is the stateful contain for the unfolder
type OptFloatUnfolder struct {
	gotype.BaseUnfoldState
	to *OptFloat
}

// UnfoldOptFloat is a helper function for structform's Fold() function
// Pass this to gotype.NewIterator
func UnfoldOptFloat(to *OptFloat) gotype.UnfoldState {
	return &OptFloatUnfolder{to: to}
}

// OnFloat64 Folds Uint64 values
func (u *OptFloatUnfolder) OnFloat64(ctx gotype.UnfoldCtx, in float64) error {
	defer ctx.Done()
	u.to = &OptFloat{exists: true, value: in}

	return nil
}

// OnNil Folds nil values
func (u *OptFloatUnfolder) OnNil(ctx gotype.UnfoldCtx) error {
	defer ctx.Done()
	u.to = &OptFloat{exists: false, value: 0}

	return nil
}
