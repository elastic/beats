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

package conditions

// And is a compound condition that combines multiple conditions with logical AND.
type And []Condition

// NewAndCondition builds this condition from a slice of Condition objects.
func NewAndCondition(conditions []Condition) And {
	return And(conditions)
}

// Check determines whether the given event matches this condition
func (c And) Check(event ValuesMap) bool {
	for _, cond := range c {
		if !cond.Check(event) {
			return false
		}
	}
	return true
}

func (c And) String() (s string) {
	for _, cond := range c {
		s = s + cond.String() + " and "
	}
	s = s[:len(s)-len(" and ")] //delete the last and
	return s
}
