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

import "fmt"

// Results the results of executing a schema.
// They are a flattened map (using dotted paths) of all the values []ValueResult representing the results
// of the IsDefs.
type Results struct {
	Source      map[string]interface{}
	Validations map[string][]ValueResult
}

func MakeResults(source map[string]interface{}) Results {
	return Results{Source: source, Validations: make(map[string][]ValueResult)}
}

func (r Results) record(path string, result ValueResult) {
	if r.Validations[path] == nil {
		r.Validations[path] = []ValueResult{result}
	} else {
		r.Validations[path] = append(r.Validations[path], result)
	}
}

// EachResult executes the given callback once per Value result.
// The provided callback can return true to keep iterating, or false
// to stop.
func (r Results) EachResult(f func(string, ValueResult) bool) {
	for path, pathResults := range r.Validations {
		for _, result := range pathResults {
			if !f(path, result) {
				return
			}
		}
	}
}

// DetailedErrors returns a new Results object consisting only of error data.
func (r Results) DetailedErrors() Results {
	errors := MakeResults(r.Source)
	r.EachResult(func(path string, vr ValueResult) bool {
		if !vr.Valid {
			errors.record(path, vr)
		}

		return true
	})
	return errors
}

// ValueResultError is used to represent an error validating an individual value.
type ValueResultError struct {
	path        string
	valueResult ValueResult
}

// Error returns the error that occurred during validation with its context included.
func (vre ValueResultError) Error() string {
	return fmt.Sprintf("@path '%s': %s", vre.path, vre.valueResult.Message)
}

// Errors returns a list of error objects, one per failed value validation.
func (r Results) Errors() []error {
	var errors []error

	r.EachResult(func(path string, vr ValueResult) bool {
		if !vr.Valid {
			errors = append(errors, ValueResultError{path, vr})
		}
		return true
	})

	return errors
}

// Valid returns true if there are no errors.
func (r Results) Valid() bool {
	r.EachResult(func(_ string, vr ValueResult) bool {
		return vr.Valid
	})
	// TODO: this is a pretty slow way to do this.
	return len(r.Errors()) == 0
}
