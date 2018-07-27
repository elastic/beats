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

// Map is the type used to define schema definitions for Schema.
type Map map[string]interface{}

// Slice is a convenience []interface{} used to declare schema defs.
type Slice []interface{}

// Validator is the result of Schema and is run against the map you'd like to test.
type Validator func(common.MapStr) (*Results, error)

// Compose combines multiple SchemaValidators into a single one.
func Compose(validators ...Validator) Validator {
	return func(actual common.MapStr) (r *Results, err error) {
		results := make([]*Results, len(validators))
		for idx, validator := range validators {
			results[idx], err = validator(actual)
			if err != nil {
				return nil, err
			}
		}

		combined := NewResults()
		for _, r := range results {
			r.EachResult(func(path Path, vr ValueResult) bool {
				combined.record(path, vr)
				return true
			})
		}
		return combined, err
	}
}

// Strict is used when you want any unspecified keys that are encountered to be considered errors.
func Strict(laxValidator Validator) Validator {
	return func(actual common.MapStr) (*Results, error) {
		results, err := laxValidator(actual)
		if err != nil {
			return results, err
		}

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

		err = walk(actual, func(woi walkObserverInfo) {
			_, validatedExactly := results.Fields[woi.path.String()]
			if validatedExactly {
				return // This key was tested, passes strict test
			}

			// Search returns the point just before an actual match (since we ruled out an exact match with the cheaper
			// hash check above. We have to validate the actual match with a prefix check as well
			matchIdx := sort.SearchStrings(validatedPaths, woi.path.String())
			if matchIdx < len(validatedPaths) && strings.HasPrefix(validatedPaths[matchIdx], woi.path.String()) {
				return
			}

			results.merge(StrictFailureResult(woi.path))
		})

		return results, err
	}
}

// Schema takes a Map and returns an executable Validator function.
func Schema(expected Map) Validator {
	return func(actual common.MapStr) (*Results, error) {
		return walkValidate(expected, actual)
	}
}

func walkValidate(expected Map, actual common.MapStr) (results *Results, err error) {
	results = NewResults()
	err = walk(
		common.MapStr(expected),
		func(expInfo walkObserverInfo) {
			actualKeyExists, actualV := expInfo.path.GetFrom(actual)

			// If this is a definition use it, if not, check exact equality
			isDef, isIsDef := expInfo.value.(IsDef)
			if !isIsDef {

				if !interfaceIsCollection(expInfo.value) {
					isDef = IsEqual(expInfo.value)
				} else if interfaceIsCollection(actualV) {
					// We don't check collections for equality, we check their properties
					// individual via our own traversal, so bail early unless the collection
					// is empty. The one exception
					if reflect.ValueOf(actualV).Len() > 0 {
						return
					}

					isDef = IsEqual(expInfo.value)
				}
			}

			if !isDef.optional || isDef.optional && actualKeyExists {
				var checkRes *Results
				checkRes, err = isDef.check(expInfo.path, actualV, actualKeyExists)
				results.merge(checkRes)
			}
		})

	return results, err
}

func interfaceIsCollection(o interface{}) bool {
	kind := reflect.ValueOf(o).Kind()
	return kind == reflect.Map || kind == reflect.Slice
}
