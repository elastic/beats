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

// NewUint returns a new OptUint wrapper
func NewUint() OptUint {
	return OptUint{
		exists: false,
		value:  0,
	}
}

// NewUintFrom returns a new OptUint wrapper with a given int
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
