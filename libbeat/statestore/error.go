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

package statestore

import (
	"errors"
	"fmt"
)

// ErrorAccess indicates that an error occurred when trying to open a Store.
type ErrorAccess struct {
	name  string
	cause error
}

// Store reports the name of the store that could not been accessed.
func (e *ErrorAccess) Store() string { return e.name }

// Unwrap returns the cause for the error or nil if the cause is unknown or has
// not been reported by the backend
func (e *ErrorAccess) Unwrap() error { return e.cause }

// Error creates a descriptive error string.
func (e *ErrorAccess) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("failed to open store '%v'", e.name)
	}
	return fmt.Sprintf("failed to open store '%v': %v", e.name, e.cause)
}

// ErrorClosed indicates that the operation failed because the store has already been closed.
type ErrorClosed struct {
	name      string
	operation string
}

// Store reports the name of the store that has been closed.
func (e *ErrorClosed) Store() string { return e.name }

// Operation returns a 'readable' name for the operation that failed to access the closed store.
func (e *ErrorClosed) Operation() string { return e.operation }

// Error creates a descriptive error string.
func (e *ErrorClosed) Error() string {
	return fmt.Sprintf("can not executed %v operation on closed store '%v'", e.operation, e.name)
}

// ErrorOperation is returned when a generic store operation failed.
type ErrorOperation struct {
	name      string
	operation string
	cause     error
}

// Store reports the name of the store.
func (e *ErrorOperation) Store() string { return e.name }

// Operation returns a 'readable' name for the operation that failed.
func (e *ErrorOperation) Operation() string { return e.operation }

// Unwrap returns the cause of the failure.
func (e *ErrorOperation) Unwrap() error { return e.cause }

// Error creates a descriptive error string.
func (e *ErrorOperation) Error() string {
	return fmt.Sprintf("failed in %v operation on store '%v': %v", e.operation, e.name, e.cause)
}

// IsClosed returns true if the cause for an Error is ErrorClosed.
func IsClosed(err error) bool {
	var tmp *ErrorClosed
	return errors.As(err, &tmp)
}
