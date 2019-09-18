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

package lookslike

import (
	"reflect"
	"sort"
	"strings"

	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
	"github.com/elastic/go-lookslike/validator"
)

// Compose combines multiple SchemaValidators into a single one.
func Compose(validators ...validator.Validator) validator.Validator {
	return func(actual interface{}) *llresult.Results {
		res := make([]*llresult.Results, len(validators))
		for idx, validator := range validators {
			res[idx] = validator(actual)
		}

		combined := llresult.NewResults()
		for _, r := range res {
			r.EachResult(func(path llpath.Path, vr llresult.ValueResult) bool {
				combined.Record(path, vr)
				return true
			})
		}
		return combined
	}
}

// Strict is used when you want any unspecified keys that are encountered to be considered errors.
func Strict(laxValidator validator.Validator) validator.Validator {
	return func(actual interface{}) *llresult.Results {
		res := laxValidator(actual)

		// When validating nil objects the lax validator is by definition sufficient
		if actual == nil {
			return res
		}

		// The inner workings of this are a little weird
		// We use a hash of dotted paths to track the res
		// We can Check if a key had a test associated with it by looking up the laxValidator
		// result data
		// What's trickier is intermediate maps, maps don't usually have explicit tests, they usually just have
		// their properties tested.
		// This method counts an intermediate map as tested if a subkey is tested.
		// Since the datastructure we have to search is a flattened hashmap of the original map we take that hashmap
		// and turn it into a sorted string array, then do a binary prefix search to determine if a subkey was tested.
		// It's a little weird, but is fairly efficient. We could stop using the flattened map as a datastructure, but
		// that would add complexity elsewhere. Probably a good refactor at some point, but not worth it now.
		validatedPaths := []string{}
		for k := range res.Fields {
			validatedPaths = append(validatedPaths, k)
		}
		sort.Strings(validatedPaths)

		walk(reflect.ValueOf(actual), false, func(woi walkObserverInfo) error {
			_, validatedExactly := res.Fields[woi.path.String()]
			if validatedExactly {
				return nil // This key was tested, passes strict test
			}

			// Search returns the point just before an actual match (since we ruled out an exact match with the cheaper
			// hash Check above. We have to validate the actual match with a prefix Check as well
			matchIdx := sort.SearchStrings(validatedPaths, woi.path.String())
			if matchIdx < len(validatedPaths) && strings.HasPrefix(validatedPaths[matchIdx], woi.path.String()) {
				return nil
			}

			res.Merge(llresult.StrictFailureResult(woi.path))

			return nil
		})

		return res
	}
}

func compile(in interface{}) (validator.Validator, error) {
	switch in.(type) {
	case isdef.IsDef:
		return compileIsDef(in.(isdef.IsDef))
	case nil:
		// nil can't be handled by the default case of IsEqual
		return compileIsDef(isdef.IsNil)
	default:
		inVal := reflect.ValueOf(in)
		switch inVal.Kind() {
		case reflect.Map:
			return compileMap(inVal)
		case reflect.Slice, reflect.Array:
			return compileSlice(inVal)
		default:
			return compileIsDef(isdef.IsEqual(in))
		}
	}
}

func compileMap(inVal reflect.Value) (validator validator.Validator, err error) {
	wo, compiled := setupWalkObserver()
	err = walkMap(inVal, true, wo)

	return func(actual interface{}) *llresult.Results {
		return compiled.Check(actual)
	}, err
}

func compileSlice(inVal reflect.Value) (validator validator.Validator, err error) {
	wo, compiled := setupWalkObserver()
	err = walkSlice(inVal, true, wo)

	// Slices are always strict in validation because
	// it would be surprising to only validate the first specified values
	return Strict(func(actual interface{}) *llresult.Results {
		return compiled.Check(actual)
	}), err
}

func compileIsDef(def isdef.IsDef) (validator validator.Validator, err error) {
	return func(actual interface{}) *llresult.Results {
		return def.Check(llpath.Path{}, actual, true)
	}, nil
}

func setupWalkObserver() (walkObserver, *CompiledSchema) {
	compiled := make(CompiledSchema, 0)
	return func(current walkObserverInfo) error {
		kind := current.value.Kind()
		isCollection := kind == reflect.Map || kind == reflect.Slice
		isEmptyCollection := isCollection && current.value.Len() == 0

		// We do comparisons on all leaf nodes. If the leaf is an empty collection
		// we do a comparison to let us test empty structures.
		if !isCollection || isEmptyCollection {
			isDef, isIsDef := current.value.Interface().(isdef.IsDef)
			if !isIsDef {
				isDef = isdef.IsEqual(current.value.Interface())
			}

			compiled = append(compiled, flatValidator{current.path, isDef})
		}
		return nil
	}, &compiled
}

// MustCompile compiles the given validation, panic-ing if that map is invalid.
func MustCompile(in interface{}) validator.Validator {
	compiled, err := compile(in)
	if err != nil {
		panic(err)
	}
	return compiled
}
