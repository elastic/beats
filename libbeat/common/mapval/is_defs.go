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

package mapval

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"
)

// KeyPresent checks that the given key is in the map, even if it has a nil value.
var KeyPresent = IsDef{name: "check key present"}

// KeyMissing checks that the given key is not present defined.
var KeyMissing = IsDef{name: "check key not present", checkKeyMissing: true}

// IsDuration tests that the given value is a duration.
var IsDuration = Is("is a duration", func(v interface{}) ValueResult {
	if _, ok := v.(time.Duration); ok {
		return ValidVR
	}
	return ValueResult{
		false,
		fmt.Sprintf("Expected a time.duration, got '%v' which is a %T", v, v),
	}
})

// IsEqual tests that the given object is equal to the actual object.
func IsEqual(to interface{}) IsDef {
	return Is("equals", func(v interface{}) ValueResult {
		if assert.ObjectsAreEqual(v, to) {
			return ValidVR
		}
		return ValueResult{
			false,
			fmt.Sprintf("objects not equal: %v != %v", v, to),
		}
	})
}

// IsEqualToValue tests that the given value is equal to the actual value.
func IsEqualToValue(to interface{}) IsDef {
	return Is("equals", func(v interface{}) ValueResult {
		if assert.ObjectsAreEqualValues(v, to) {
			return ValidVR
		}
		return ValueResult{
			false,
			fmt.Sprintf("values not equal: %v != %v", v, to),
		}
	})
}

// IsNil tests that a value is nil.
var IsNil = Is("is nil", func(v interface{}) ValueResult {
	if v == nil {
		return ValidVR
	}
	return ValueResult{
		false,
		fmt.Sprintf("Value %v is not nil", v),
	}
})

func intGtChecker(than int) ValueValidator {
	return func(v interface{}) ValueResult {
		n, ok := v.(int)
		if !ok {
			msg := fmt.Sprintf("%v is a %T, but was expecting an int!", v, v)
			return ValueResult{false, msg}
		}

		if n > than {
			return ValidVR
		}

		return ValueResult{
			false,
			fmt.Sprintf("%v is not greater than %v", n, than),
		}
	}
}

// IsIntGt tests that a value is an int greater than.
func IsIntGt(than int) IsDef {
	return Is("greater than", intGtChecker(than))
}
