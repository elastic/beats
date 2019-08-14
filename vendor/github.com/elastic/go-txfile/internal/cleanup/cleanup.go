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

// Package cleanup provides common helpers for common cleanup patterns on defer
//
// Use the helpers with `defer`. For example use IfNot with `defer`, such that
// cleanup functions will be executed if `check` is false, no matter if an
// error has been returned or an panic has occured.
//
//     initOK := false
//     defer cleanup.IfNot(&initOK, func() {
//       cleanup
//     })
//
//     ... // init structures...
//
//     initOK = true // notify handler cleanup code must not be executed
//
package cleanup

// If will run the cleanup function if the bool value is true.
func If(check *bool, cleanup func()) {
	if *check {
		cleanup()
	}
}

// IfNot will run the cleanup function if the bool value is false.
func IfNot(check *bool, cleanup func()) {
	if !(*check) {
		cleanup()
	}
}

// IfPred will run the cleanup function if pred returns true.
func IfPred(pred func() bool, cleanup func()) {
	if pred() {
		cleanup()
	}
}

// IfNotPred will run the cleanup function if pred returns false.
func IfNotPred(pred func() bool, cleanup func()) {
	if !pred() {
		cleanup()
	}
}

// WithError returns a cleanup function calling a custom handler if an error occured.
func WithError(fn func(error), cleanup func() error) func() {
	return func() {
		if err := cleanup(); err != nil {
			fn(err)
		}
	}
}

// IgnoreError silently ignores errors in the cleanup function.
func IgnoreError(cleanup func() error) func() {
	return func() { _ = cleanup() }
}
