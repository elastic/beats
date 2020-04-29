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

package llresult

import (
	"fmt"

	"github.com/elastic/go-lookslike/llpath"
)

// Results the results of executing a schema.
// They are a flattened map (using dotted paths) of all the values ValueResult representing the results
// of the IsDefs.
type Results struct {
	Fields map[string][]ValueResult
	Valid  bool
}

//ValueResult represents the result of checking a leaf value.
type ValueResult struct {
	Valid   bool
	Message string // Reason this is invalid
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
func SimpleResult(path llpath.Path, valid bool, msg string, args ...interface{}) *Results {
	vr := ValueResult{valid, fmt.Sprintf(msg, args...)}
	return SingleResult(path, vr)
}

// SingleResult returns a *Results object with a single validated value at the given Path
// using the providedValueResult as its sole validation.
func SingleResult(path llpath.Path, result ValueResult) *Results {
	r := NewResults()
	r.Record(path, result)
	return r
}

// Merge combines multiple *Results sets together.
func (r *Results) Merge(other *Results) {
	for otherPath, valueResults := range other.Fields {
		for _, valueResult := range valueResults {
			r.Record(llpath.MustParsePath(otherPath), valueResult)
		}
	}
}

// MergeUnderPrefix merges the given results at the path specified by the given prefix.
func (r *Results) MergeUnderPrefix(prefix llpath.Path, other *Results) {
	if len(prefix) == 0 {
		// If the prefix is empty, just use standard Merge
		// No need to add the dots
		r.Merge(other)
		return
	}

	for otherPath, valueResults := range other.Fields {
		for _, valueResult := range valueResults {
			parsed := llpath.MustParsePath(otherPath)
			r.Record(prefix.Concat(parsed), valueResult)
		}
	}
}

// Record records a single path result to this instance.
func (r *Results) Record(p llpath.Path, result ValueResult) {
	if r.Fields[p.String()] == nil {
		r.Fields[p.String()] = []ValueResult{result}
	} else {
		r.Fields[p.String()] = append(r.Fields[p.String()], result)
	}

	if !result.Valid {
		r.Valid = false
	}
}

// EachResult executes the given callback once per Value result.
// The provided callback can return true to keep iterating, or false
// to stop.
func (r Results) EachResult(f func(llpath.Path, ValueResult) bool) {
	for p, pathResults := range r.Fields {
		for _, result := range pathResults {
			// We can ignore path parse errors here, those are from scalars and other
			// types that have an invalid string path
			// TODO: Find a cleaner way to do this
			parsed, _ := llpath.ParsePath(p)
			if !f(parsed, result) {
				return
			}
		}
	}
}

// DetailedErrors returns a new Results object consisting only of error data.
func (r *Results) DetailedErrors() *Results {
	errors := NewResults()
	r.EachResult(func(p llpath.Path, vr ValueResult) bool {
		if !vr.Valid {
			errors.Record(p, vr)
		}

		return true
	})
	return errors
}

//ValueResultError is used to represent an error validating an individual value.
type ValueResultError struct {
	path        llpath.Path
	valueResult ValueResult
}

// Error returns the error that occurred during validation with its context included.
func (vre ValueResultError) Error() string {
	return fmt.Sprintf("@Path '%s': %s", vre.path, vre.valueResult.Message)
}

// Errors returns a list of error objects, one per failed value validation.
func (r Results) Errors() []error {
	errors := make([]error, 0)

	r.EachResult(func(path llpath.Path, vr ValueResult) bool {
		if !vr.Valid {
			errors = append(errors, ValueResultError{path, vr})
		}
		return true
	})

	return errors
}
