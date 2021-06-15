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

// OptUint is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type OptUint struct {
	exists bool
	value  uint64
}

// NewNone returns a new OptUint wrapper
func NewNone() OptUint {
	return OptUint{
		exists: false,
		value:  0,
	}
}

// OptUintWith returns a new OptUint wrapper with a given int
func OptUintWith(i uint64) OptUint {
	return OptUint{
		exists: true,
		value:  i,
	}
}

// IsZero returns true if the underlying value nil
func (opt OptUint) IsZero() bool {
	return !opt.exists
}

// ValueOr returns the stored value, or a given int
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt OptUint) ValueOr(i uint64) uint64 {
	if opt.exists {
		return opt.value
	}
	return i
}

// SumOptUint sums a list of OptUint values
func SumOptUint(opts ...OptUint) uint64 {
	var sum uint64
	for _, opt := range opts {
		sum += opt.ValueOr(0)
	}
	return sum
}
