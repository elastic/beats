// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFlagsAreSame(t *testing.T) {
	tests := []struct {
		Name           string
		Flags1, Flags2 Flags
		Expected       bool
	}{
		{
			Name:     "both nils",
			Expected: true,
		},
		{
			Name:   "first nil, second non",
			Flags1: nil,
			Flags2: Flags{
				"foo": "bar",
			},
			Expected: false,
		},
		{
			Name: "same",
			Flags1: Flags{
				"foo": "bar",
			},
			Flags2: Flags{
				"foo": "bar",
			},
			Expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			res := FlagsAreSame(tc.Flags1, tc.Flags2)
			diff := cmp.Diff(tc.Expected, res)
			if diff != "" {
				t.Error(diff)
			}
		})
	}
}
