// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package source

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSourceValidation(t *testing.T) {
	type testCase struct {
		name    string
		source  Source
		wantErr error
	}
	testCases := []testCase{
		{
			"no error",
			Source{Inline: &InlineSource{}},
			nil,
		},
		{
			"no concrete source",
			Source{},
			ErrInvalidSource,
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
