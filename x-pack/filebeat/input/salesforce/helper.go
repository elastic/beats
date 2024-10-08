// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import "time"

// timeNow wraps time.Now to mock time for tests.
var timeNow = time.Now

// mockTimeNow mocks timeNow for tests.
func mockTimeNow(t time.Time) {
	timeNow = func() time.Time {
		return t
	}
}

// resetTimeNow resets timeNow to time.Now.
func resetTimeNow() {
	timeNow = time.Now
}

// pointer returns a pointer to the given value.
//
// For example: Assigning &true to value of type *bool is not possible but
// pointer(true) is assignable to the same value of type *bool as address operator
// can be applied to pointer(true) as the returned value is an addressable value.
//
// See: https://go.dev/ref/spec#Address_operators
func pointer[T any](d T) *T {
	return &d
}
