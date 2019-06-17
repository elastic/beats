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

import "github.com/elastic/go-lookslike/llpath"

// ValidResult is a convenience value for Valid results.
func ValidResult(p llpath.Path) *Results {
	return SimpleResult(p, true, "is valid")
}

// ValidVR is a convenience value for Valid results.
var ValidVR = ValueResult{true, "is valid"}

// KeyMissingResult is emitted when a key was expected, but was not present.
func KeyMissingResult(path llpath.Path) *Results {
	return SingleResult(path, KeyMissingVR)
}

// KeyMissingVR is emitted when a key was expected, but was not present.
var KeyMissingVR = ValueResult{
	false,
	"expected this key to be present",
}

// StrictFailureResult is emitted when Strict() is used, and an unexpected field is found.
func StrictFailureResult(path llpath.Path) *Results {
	return SingleResult(path, StrictFailureVR)
}

// StrictFailureVR is emitted when Strict() is used, and an unexpected field is found.
var StrictFailureVR = ValueResult{
	false,
	"unexpected field encountered during strict validation",
}
