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
	Fields map[string][]ValueResult
	Valid  bool
}

// NewResults creates a new Results object.
func NewResults() *Results {
	return &Results{
		Fields: make(map[string][]ValueResult),
		Valid:  true,
	}
}

// SimpleResult provides a convenient and simple method for creating a *Results object for a single validation.
// It's a very common way for validators to return a *Results object, and is generally simpler than
// using SingleResult.
func SimpleResult(path path, valid bool, msg string, args ...interface{}) *Results {
	vr := ValueResult{valid, fmt.Sprintf(msg, args...)}
	return SingleResult(path, vr)
}

// SingleResult returns a *Results object with a single validated value at the given path
// using the provided ValueResult as its sole validation.
func SingleResult(path path, result ValueResult) *Results {
	r := NewResults()
	r.record(path, result)
	return r
}

func (r *Results) merge(other *Results) {
	for path, valueResults := range other.Fields {
		for _, valueResult := range valueResults {
			r.record(mustParsePath(path), valueResult)
		}
	}
}

func (r *Results) mergeUnderPrefix(prefix path, other *Results) {
	if len(prefix) == 0 {
		// If the prefix is empty, just use standard merge
		// No need to add the dots
		r.merge(other)
		return
	}

	for path, valueResults := range other.Fields {
		for _, valueResult := range valueResults {
			parsed := mustParsePath(path)
			r.record(prefix.concat(parsed), valueResult)
		}
	}
}

func (r *Results) record(path path, result ValueResult) {
	if r.Fields[path.String()] == nil {
		r.Fields[path.String()] = []ValueResult{result}
	} else {
		r.Fields[path.String()] = append(r.Fields[path.String()], result)
	}

	if !result.Valid {
		r.Valid = false
	}
}

// EachResult executes the given callback once per Value result.
// The provided callback can return true to keep iterating, or false
// to stop.
func (r Results) EachResult(f func(path, ValueResult) bool) {
	for path, pathResults := range r.Fields {
		for _, result := range pathResults {
			if !f(mustParsePath(path), result) {
				return
			}
		}
	}
}

// DetailedErrors returns a new Results object consisting only of error data.
func (r *Results) DetailedErrors() *Results {
	errors := NewResults()
	r.EachResult(func(path path, vr ValueResult) bool {
		if !vr.Valid {
			errors.record(path, vr)
		}

		return true
	})
	return errors
}

// ValueResultError is used to represent an error validating an individual value.
type ValueResultError struct {
	path        path
	valueResult ValueResult
}

// Error returns the error that occurred during validation with its context included.
func (vre ValueResultError) Error() string {
	return fmt.Sprintf("@path '%s': %s", vre.path, vre.valueResult.Message)
}

// Errors returns a list of error objects, one per failed value validation.
func (r Results) Errors() []error {
	errors := make([]error, 0)

	r.EachResult(func(path path, vr ValueResult) bool {
		if !vr.Valid {
			errors = append(errors, ValueResultError{path, vr})
		}
		return true
	})

	return errors
}
