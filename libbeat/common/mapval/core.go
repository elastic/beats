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
	"reflect"
	"sort"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

// Is creates a named IsDef with the given checker.
func Is(name string, checker ValueValidator) IsDef {
	return IsDef{name: name, checker: checker}
}

// Optional wraps an IsDef to mark the field's presence as optional.
func Optional(id IsDef) IsDef {
	id.name = "optional " + id.name
	id.optional = true
	return id
}

// Map is the type used to define schema definitions for Compile.
type Map map[string]interface{}

// Slice is a convenience []interface{} used to declare schema defs.
type Slice []interface{}

// Validator is the result of Compile and is run against the map you'd like to test.
type Validator func(common.MapStr) *Results

// Compose combines multiple SchemaValidators into a single one.
func Compose(validators ...Validator) Validator {
	return func(actual common.MapStr) *Results {
		results := make([]*Results, len(validators))
		for idx, validator := range validators {
			results[idx] = validator(actual)
		}

		combined := NewResults()
		for _, r := range results {
			r.EachResult(func(path Path, vr ValueResult) bool {
				combined.record(path, vr)
				return true
			})
		}
		return combined
	}
}

// Strict is used when you want any unspecified keys that are encountered to be considered errors.
func Strict(laxValidator Validator) Validator {
	return func(actual common.MapStr) *Results {
		results := laxValidator(actual)

		// The inner workings of this are a little weird
		// We use a hash of dotted paths to track the results
		// We can check if a key had a test associated with it by looking up the laxValidator
		// result data
		// What's trickier is intermediate maps, maps don't usually have explicit tests, they usually just have
		// their properties tested.
		// This method counts an intermediate map as tested if a subkey is tested.
		// Since the datastructure we have to search is a flattened hashmap of the original map we take that hashmap
		// and turn it into a sorted string array, then do a binary prefix search to determine if a subkey was tested.
		// It's a little weird, but is fairly efficient. We could stop using the flattened map as a datastructure, but
		// that would add complexity elsewhere. Probably a good refactor at some point, but not worth it now.
		validatedPaths := []string{}
		for k := range results.Fields {
			validatedPaths = append(validatedPaths, k)
		}
		sort.Strings(validatedPaths)

		walk(actual, false, func(woi walkObserverInfo) error {
			_, validatedExactly := results.Fields[woi.path.String()]
			if validatedExactly {
				return nil // This key was tested, passes strict test
			}

			// Search returns the point just before an actual match (since we ruled out an exact match with the cheaper
			// hash check above. We have to validate the actual match with a prefix check as well
			matchIdx := sort.SearchStrings(validatedPaths, woi.path.String())
			if matchIdx < len(validatedPaths) && strings.HasPrefix(validatedPaths[matchIdx], woi.path.String()) {
				return nil
			}

			results.merge(StrictFailureResult(woi.path))

			return nil
		})

		return results
	}
}

// Compile takes the given map, validates the paths within it, and returns
// a Validator that can check real data.
func Compile(in Map) (validator Validator, err error) {
	compiled := make(CompiledSchema, 0)
	err = walk(common.MapStr(in), true, func(current walkObserverInfo) error {

		// Determine whether we should test this value
		// We want to test all values except collections that contain a value
		// If a collection contains a value, we check those 'leaf' values instead
		rv := reflect.ValueOf(current.value)
		kind := rv.Kind()
		isCollection := kind == reflect.Map || kind == reflect.Slice
		isNonEmptyCollection := isCollection && rv.Len() > 0

		if !isNonEmptyCollection {
			isDef, isIsDef := current.value.(IsDef)
			if !isIsDef {
				isDef = IsEqual(current.value)
			}

			compiled = append(compiled, flatValidator{current.path, isDef})
		}
		return nil
	})

	return func(actual common.MapStr) *Results {
		return compiled.Check(actual)
	}, err
}

// MustCompile compiles the given map, panic-ing if that map is invalid.
func MustCompile(in Map) Validator {
	compiled, err := Compile(in)
	if err != nil {
		panic(err)
	}
	return compiled
}
