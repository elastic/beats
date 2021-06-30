// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package capabilities

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestExpr(t *testing.T) {
	cases := []struct {
		Pattern     string
		Value       string
		ShouldMatch bool
	}{
		{"", "", true},
		{"*", "", true},
		{"*", "test", true},
		{"*", "system/test", true},
		{"system/*", "system/test", true},
		{"*/test", "system/test", true},
		{"*/*", "system/test", true},
		{"system/*", "agent/test", false},
		{"*/test", "test/system", false},
		{"*/test", "test", false},
		{"*/*", "test", false},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("testcase #%d", i), func(tt *testing.T) {
			match := matchesExpr(tc.Pattern, tc.Value)
			assert.Equal(t,
				tc.ShouldMatch,
				match,
				fmt.Sprintf("'%s' and '%s' and expecting should match: %v", tc.Pattern, tc.Value, tc.ShouldMatch),
			)
		})
	}
}
