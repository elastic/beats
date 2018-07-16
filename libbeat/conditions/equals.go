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

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
)

type equalsValue struct {
	Int  uint64
	Str  string
	Bool bool
}

// Equals is a Condition for testing string equality.
type Equals map[string]equalsValue

// NewEqualsCondition builds a new Equals using the given configuration of string equality checks.
func NewEqualsCondition(fields map[string]interface{}) (c Equals, err error) {
	c = Equals{}

	for field, value := range fields {
		uintValue, err := ExtractInt(value)
		if err == nil {
			c[field] = equalsValue{Int: uintValue}
			continue
		}

		sValue, err := ExtractString(value)
		if err == nil {
			c[field] = equalsValue{Str: sValue}
			continue
		}

		bValue, err := ExtractBool(value)
		if err == nil {
			c[field] = equalsValue{Bool: bValue}
			continue
		}

		return nil, fmt.Errorf("condition attempted to set '%v' -> '%v' and encountered unexpected type '%T', only strings ints, and bools are allowed", field, value, value)
	}

	return c, nil
}

// Check determines whether the given event matches this condition.
func (c Equals) Check(event ValuesMap) bool {
	for field, equalValue := range c {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		intValue, err := ExtractInt(value)
		if err == nil {
			if intValue != equalValue.Int {
				return false
			}

			continue
		}

		sValue, err := ExtractString(value)
		if err == nil {
			if sValue != equalValue.Str {
				return false
			}

			continue
		}

		bValue, err := ExtractBool(value)
		if err == nil {
			if bValue != equalValue.Bool {
				return false
			}

			continue
		}

		logp.Err("unexpected type %T in equals condition as it accepts only integers, strings or bools. ", value)
		return false
	}

	return true
}

func (c Equals) String() string {
	return fmt.Sprintf("equals: %v", map[string]equalsValue(c))
}
