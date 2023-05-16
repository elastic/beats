// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin

package source

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInlineSourceValidation(t *testing.T) {
	type testCase struct {
		name    string
		source  *InlineSource
		wantErr error
	}
	testCases := []testCase{
		{
			"no error",
			&InlineSource{Script: "a script"},
			nil,
		},
		{
			"no script",
			&InlineSource{},
			ErrNoInlineScript,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.source.Validate()
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
