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

	"github.com/elastic/beats/v7/libbeat/logp"
)

type equalsValue struct {
	Int  uint64
	Str  string
	Bool bool

	t interface{}
}

// Equals is a Condition for testing string equality.
type Equals map[string]equalsValue

// NewEqualsCondition builds a new Equals using the given configuration of string equality checks.
func NewEqualsCondition(fields map[string]interface{}) (c Equals, err error) {
	c = Equals{}

	for field, value := range fields {
		uintValue, err := ExtractInt(value)
		if err == nil {
			c[field] = equalsValue{Int: uintValue, t: uint64(0)}
			continue
		}

		sValue, err := ExtractString(value)
		if err == nil {
			c[field] = equalsValue{Str: sValue, t: ""}
			continue
		}

		bValue, err := ExtractBool(value)
		if err == nil {
			c[field] = equalsValue{Bool: bValue, t: false}
			continue
		}

		return nil, fmt.Errorf("condition attempted to set '%v' -> '%v' and encountered unexpected type '%T', only strings, ints, and booleans are allowed", field, value, value)
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

		switch equalValue.t.(type) {
		case uint64:
			intValue, err := ExtractInt(value)
			if err == nil {
				if intValue != equalValue.Int {
					return false
				}

			} else {
				logp.L().Named(logName).Warnf("expected int but got type %T in equals condition.", value)
			}
			continue

		case string:
			sValue, err := ExtractString(value)
			if err == nil {
				if sValue != equalValue.Str {
					return false
				}
			} else {
				logp.L().Named(logName).Warnf("expected string but got type %T in equals condition.", value)
			}
			continue

		case bool:
			bValue, err := ExtractBool(value)
			if err == nil {
				if bValue != equalValue.Bool {
					return false
				}
			} else {
				logp.L().Named(logName).Warnf("expected bool but got type %T in equals condition.", value)
			}
			continue
		}

		logp.L().Named(logName).Warnf("unexpected type %T in equals condition as it accepts only integers, strings, or booleans.", value)
		return false
	}

	return true
}

func (c Equals) String() string {
	return fmt.Sprintf("equals: %v", map[string]equalsValue(c))
}

// Equals2 is a Condition for testing string equality.
type Equals2 map[string]equalsValueType

// equalsValueType checks its defined value equals the given value
type equalsValueType interface {
	Check(interface{}) bool
}

type equalsIntValue uint64

func (e equalsIntValue) Check(value interface{}) bool {
	if intValue, err := ExtractInt(value); err == nil {
		return intValue == uint64(e)
	}
	logp.L().Named(logName).Warnf("expected int but got type %T in equals condition.", value)
	return false
}

type equalsStringValue string

func (e equalsStringValue) Check(value interface{}) bool {
	if sValue, err := ExtractString(value); err == nil {
		return sValue == string(e)
	}
	logp.L().Named(logName).Warnf("expected string but got type %T in equals condition.", value)
	return false
}

type equalsBoolValue bool

func (e equalsBoolValue) Check(value interface{}) bool {
	if bValue, err := ExtractBool(value); err == nil {
		return bValue == bool(e)
	}
	logp.L().Named(logName).Warnf("expected bool but got type %T in equals condition.", value)
	return false
}

// NewEqualsCondition2 builds a new Equals using the given configuration of string equality checks.
func NewEqualsCondition2(fields map[string]interface{}) (c Equals2, err error) {
	c = Equals2{}

	for field, value := range fields {
		uintValue, err := ExtractInt(value)
		if err == nil {
			c[field] = equalsIntValue(uintValue)
			continue
		}

		sValue, err := ExtractString(value)
		if err == nil {
			c[field] = equalsStringValue(sValue)
			continue
		}

		bValue, err := ExtractBool(value)
		if err == nil {
			c[field] = equalsBoolValue(bValue)
			continue
		}

		return nil, fmt.Errorf("condition attempted to set '%v' -> '%v' and encountered unexpected type '%T', only strings, ints, and booleans are allowed", field, value, value)
	}

	return c, nil
}

// Check determines whether the given event matches this condition.
func (c Equals2) Check(event ValuesMap) bool {
	for field, equalValue := range c {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		if !equalValue.Check(value) {
			return false
		}
	}

	return true
}

func (c Equals2) String() string {
	return fmt.Sprintf("equals: %v", map[string]equalsValueType(c))
}

// Equals3 is a Condition for testing string equality.
type Equals3 map[string]equalsValueFunc

type equalsValueFunc func(interface{}) bool

func equalsIntValue3(i uint64) equalsValueFunc {
	return func(value interface{}) bool {
		if sValue, err := ExtractInt(value); err == nil {
			return sValue == i
		}
		logp.L().Named(logName).Warnf("expected int but got type %T in equals condition.", value)
		return false
	}
}

func equalsStringValue3(s string) equalsValueFunc {
	return func(value interface{}) bool {
		if sValue, err := ExtractString(value); err == nil {
			return sValue == s
		}
		logp.L().Named(logName).Warnf("expected string but got type %T in equals condition.", value)
		return false
	}
}

func equalsBoolValue3(b bool) equalsValueFunc {
	return func(value interface{}) bool {
		if sValue, err := ExtractBool(value); err == nil {
			return sValue == b
		}
		logp.L().Named(logName).Warnf("expected bool but got type %T in equals condition.", value)
		return false
	}
}

// NewEqualsCondition3 builds a new Equals using the given configuration of string equality checks.
func NewEqualsCondition3(fields map[string]interface{}) (c Equals3, err error) {
	c = Equals3{}

	for field, value := range fields {
		uintValue, err := ExtractInt(value)
		if err == nil {
			c[field] = equalsIntValue3(uintValue)
			continue
		}

		sValue, err := ExtractString(value)
		if err == nil {
			c[field] = equalsStringValue3(sValue)
			continue
		}

		bValue, err := ExtractBool(value)
		if err == nil {
			c[field] = equalsBoolValue3(bValue)
			continue
		}

		return nil, fmt.Errorf("condition attempted to set '%v' -> '%v' and encountered unexpected type '%T', only strings, ints, and booleans are allowed", field, value, value)
	}

	return c, nil
}

// Check determines whether the given event matches this condition.
func (c Equals3) Check(event ValuesMap) bool {
	for field, equalValue := range c {

		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		if !equalValue(value) {
			return false
		}
	}

	return true
}

func (c Equals3) String() string {
	return fmt.Sprintf("equals: %v", map[string]equalsValueFunc(c))
}
