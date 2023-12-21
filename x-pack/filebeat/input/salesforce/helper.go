// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import "time"

// timeNow wraps time.Now to mock time for tests
var timeNow = time.Now

func mockTimeNow(t time.Time) {
	timeNow = func() time.Time {
		return t
	}
}

func resetTimeNow() {
	timeNow = time.Now
}

func pointer[T any](d T) *T {
	return &d
}
