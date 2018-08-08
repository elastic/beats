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

// Package invariant provides helpers for checking and panicing on faulty invariants.
package invariant

import "fmt"

// Check will raise an error with the provided message in case b is false.
func Check(b bool, msg string) {
	if b {
		return
	}

	if msg == "" {
		panic("failing invariant")
	}
	panic(msg)
}

// Checkf will raise an error in case b is false. Checkf accept a fmt.Sprintf
// compatible format string with parameters.
func Checkf(b bool, msgAndArgs ...interface{}) {
	if b {
		return
	}

	switch len(msgAndArgs) {
	case 0:
		panic("failing invariant")
	case 1:
		panic(msgAndArgs[0].(string))
	default:
		panic(fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...))
	}
}

// CheckNot will raise an error with the provided message in case b is true.
func CheckNot(b bool, msg string) {
	Check(!b, msg)
}

// CheckNotf will raise an error with the provided message in case b is true.
// CheckNotf accept a fmt.Sprintf compatible format string with parameters.
func CheckNotf(b bool, msgAndArgs ...interface{}) {
	Checkf(!b, msgAndArgs...)
}

// Unreachable marks some code sequence that must never be executed.
func Unreachable(msg string) {
	panic(msg)
}

// Unreachablef marks some code sequence that must never be executed.
func Unreachablef(f string, vs ...interface{}) {
	panic(fmt.Sprintf(f, vs...))
}
