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
	"reflect"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// KeyPresent checks that the given key is in the map, even if it has a nil value.
var KeyPresent = IsDef{name: "check key present"}

// KeyMissing checks that the given key is not present defined.
var KeyMissing = IsDef{name: "check key not present", checkKeyMissing: true}

func init() {
	MustRegisterEqual(IsEqualToTime)
}

// InvalidEqualFnError is the error type returned by RegisterEqual when
// there is an issue with the given function.
type InvalidEqualFnError struct{ msg string }

func (e InvalidEqualFnError) Error() string {
	return fmt.Sprintf("Function is not a valid equal function: %s", e.msg)
}

// MustRegisterEqual is the panic-ing equivalent of RegisterEqual.
func MustRegisterEqual(fn interface{}) {
	if err := RegisterEqual(fn); err != nil {
		panic(fmt.Sprintf("Could not register fn as equal! %v", err))
	}
}

var equalChecks = map[reflect.Type]reflect.Value{}

// RegisterEqual takes a function of the form fn(v someType) IsDef
// and registers it to check equality for that type.
func RegisterEqual(fn interface{}) error {
	fnV := reflect.ValueOf(fn)
	fnT := fnV.Type()

	if fnT.Kind() != reflect.Func {
		return InvalidEqualFnError{"Provided value is not a function"}
	}
	if fnT.NumIn() != 1 {
		return InvalidEqualFnError{"Equal FN should take one argument"}
	}
	if fnT.NumOut() != 1 {
		return InvalidEqualFnError{"Equal FN should return one value"}
	}
	if fnT.Out(0) != reflect.TypeOf(IsDef{}) {
		return InvalidEqualFnError{"Equal FN should return an IsDef"}
	}

	inT := fnT.In(0)
	if _, ok := equalChecks[inT]; ok {
		return InvalidEqualFnError{fmt.Sprintf("Duplicate Equal FN for type %v encountered!", inT)}
	}

	equalChecks[inT] = fnV

	return nil
}

// IsEqual tests that the given object is equal to the actual object.
func IsEqual(to interface{}) IsDef {
	toV := reflect.ValueOf(to)
	isDefFactory, ok := equalChecks[toV.Type()]

	// If there are no handlers declared explicitly for this type we perform a deep equality check
	if !ok {
		return IsDeepEqual(to)
	}

	// We know this is an isdef due to the Register check previously
	checker := isDefFactory.Call([]reflect.Value{toV})[0].Interface().(IsDef).checker

	return Is("equals", func(path Path, v interface{}) *Results {
		return checker(path, v)
	})
}

// IsEqualToTime ensures that the actual value is the given time, regardless of zone.
func IsEqualToTime(to time.Time) IsDef {
	return Is("equal to time", func(path Path, v interface{}) *Results {
		actualTime, ok := v.(time.Time)
		if !ok {
			return SimpleResult(path, false, "Value %t was not a time.Time", v)
		}

		if actualTime.Equal(to) {
			return ValidResult(path)
		}

		return SimpleResult(path, false, "actual(%v) != expected(%v)", actualTime, to)
	})
}

// IsDeepEqual checks equality using reflect.DeepEqual.
func IsDeepEqual(to interface{}) IsDef {
	return Is("equals", func(path Path, v interface{}) *Results {
		if reflect.DeepEqual(v, to) {
			return ValidResult(path)
		}
		return SimpleResult(
			path,
			false,
			fmt.Sprintf("objects not equal: actual(%v) != expected(%v)", v, to),
		)
	})
}

// IsArrayOf validates that the array at the given key is an array of objects all validatable
// via the given Validator.
func IsArrayOf(validator Validator) IsDef {
	return Is("array of maps", func(path Path, v interface{}) *Results {
		vArr, isArr := v.([]common.MapStr)
		if !isArr {
			return SimpleResult(path, false, "Expected array at given path")
		}

		results := NewResults()

		for idx, curMap := range vArr {
			var validatorRes *Results
			validatorRes = validator(curMap)
			results.mergeUnderPrefix(path.ExtendSlice(idx), validatorRes)
		}

		return results
	})
}

// IsAny takes a variable number of IsDef's and combines them with a logical OR. If any single definition
// matches the key will be marked as valid.
func IsAny(of ...IsDef) IsDef {
	names := make([]string, len(of))
	for i, def := range of {
		names[i] = def.name
	}
	isName := fmt.Sprintf("either %#v", names)

	return Is(isName, func(path Path, v interface{}) *Results {
		for _, def := range of {
			vr := def.check(path, v, true)
			if vr.Valid {
				return vr
			}
		}

		return SimpleResult(
			path,
			false,
			fmt.Sprintf("Value was none of %#v, actual value was %#v", names, v),
		)
	})
}

// IsStringContaining validates that the the actual value contains the specified substring.
func IsStringContaining(needle string) IsDef {
	return Is("is string containing", func(path Path, v interface{}) *Results {
		strV, ok := v.(string)

		if !ok {
			return SimpleResult(
				path,
				false,
				fmt.Sprintf("Unable to convert '%v' to string", v),
			)
		}

		if !strings.Contains(strV, needle) {
			return SimpleResult(
				path,
				false,
				fmt.Sprintf("String '%s' did not contain substring '%s'", strV, needle),
			)
		}

		return ValidResult(path)
	})
}

// IsDuration tests that the given value is a duration.
var IsDuration = Is("is a duration", func(path Path, v interface{}) *Results {
	if _, ok := v.(time.Duration); ok {
		return ValidResult(path)
	}
	return SimpleResult(
		path,
		false,
		fmt.Sprintf("Expected a time.duration, got '%v' which is a %T", v, v),
	)
})

// IsNil tests that a value is nil.
var IsNil = Is("is nil", func(path Path, v interface{}) *Results {
	if v == nil {
		return ValidResult(path)
	}
	return SimpleResult(
		path,
		false,
		fmt.Sprintf("Value %v is not nil", v),
	)
})

func intGtChecker(than int) ValueValidator {
	return func(path Path, v interface{}) *Results {
		n, ok := v.(int)
		if !ok {
			msg := fmt.Sprintf("%v is a %T, but was expecting an int!", v, v)
			return SimpleResult(path, false, msg)
		}

		if n > than {
			return ValidResult(path)
		}

		return SimpleResult(
			path,
			false,
			fmt.Sprintf("%v is not greater than %v", n, than),
		)
	}
}

// IsIntGt tests that a value is an int greater than.
func IsIntGt(than int) IsDef {
	return Is("greater than", intGtChecker(than))
}
