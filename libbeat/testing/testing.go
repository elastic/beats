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

package testing

// Driver for testing, manages test flow and controls output
type Driver interface {
	// Run tests under a given namespace
	Run(name string, f func(Driver))

	// Info reports some value retrieved while testing to the user
	Info(field, value string)

	// Warn shows a warning to the user
	Warn(field string, reason string)

	// Error will report an error on the given field if err != nil, will report OK if not
	Error(field string, err error)

	// Fatal behaves like error but stops current goroutine on error
	Fatal(field string, err error)

	// Shows given result to the user
	Result(data string)
}

// Testable is optionally implemented by clients that support self testing.
// Test method will test current settings work for this output.
type Testable interface {
	Test(Driver)
}
