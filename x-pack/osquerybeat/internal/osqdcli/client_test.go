// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveHitTypes(t *testing.T) {

	tests := []struct {
		name          string
		hit, colTypes map[string]string
		res           map[string]interface{}
	}{
		{
			name: "empty",
			res:  map[string]interface{}{},
		},
		{
			name: "resolvable",
			hit: map[string]string{
				"pid":      "5551",
				"pid_int":  "5552",
				"pid_uint": "5553",
				"pid_text": "5543",
				"foo":      "bar",
			},
			colTypes: map[string]string{
				"pid":      "BIGINT",
				"pid_int":  "INTEGER",
				"pid_uint": "UNSIGNED_BIGINT",
				"pid_text": "TEXT",
			},
			res: map[string]interface{}{
				"pid":      int64(5551),
				"pid_int":  int64(5552),
				"pid_uint": uint64(5553),
				"pid_text": "5543",
				"foo":      "bar",
			},
		},
		{
			// Should preserve the field if it can not be parsed into the type
			name: "wrong type",
			hit: map[string]string{
				"data": "0,22,137,138,29754,49154,49155",
				"foo":  "bar",
			},
			colTypes: map[string]string{"data": "BIGINT"},
			res: map[string]interface{}{
				"data": "0,22,137,138,29754,49154,49155",
				"foo":  "bar",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := resolveHitTypes(tc.hit, tc.colTypes)
			diff := cmp.Diff(tc.res, res)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
