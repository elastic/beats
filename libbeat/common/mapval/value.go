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

// ValueResult represents the result of checking a leaf value.
type ValueResult struct {
	Valid   bool
	Message string // Reason this is invalid
}

// A ValueValidator is used to validate a value in a Map.
type ValueValidator func(v interface{}) ValueResult

// An IsDef defines the type of check to do.
// Generally only name and checker are set. optional and checkKeyMissing are
// needed for weird checks like key presence.
type IsDef struct {
	name            string
	checker         ValueValidator
	optional        bool
	checkKeyMissing bool
}

func (id IsDef) check(v interface{}, keyExists bool) ValueResult {
	if id.checkKeyMissing {
		if !keyExists {
			return ValidVR
		}

		return ValueResult{false, "key should not exist!"}
	}

	if !id.optional && !keyExists {
		return KeyMissingVR
	}

	if id.checker != nil {
		return id.checker(v)
	}

	return ValidVR
}

// ValidVR is a convenience value for Valid results.
var ValidVR = ValueResult{true, ""}

// KeyMissingVR is emitted when a key was expected, but was not present.
var KeyMissingVR = ValueResult{
	false,
	"expected to see a key here",
}

// StrictFailureVR is emitted when Strict() is used, and an unexpected field is found.
var StrictFailureVR = ValueResult{false, "unexpected field encountered during strict validation"}
