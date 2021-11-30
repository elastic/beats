// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTargetInfo(t *testing.T) {
	cases := []struct {
		name        string
		param       string
		expected    targetInfo
		expectedErr string
	}{
		{
			name:     "valid url.value",
			param:    "url.value",
			expected: targetInfo{Type: "url.value"},
		},
		{
			name:        "invalid url.value",
			param:       "url.value.something",
			expectedErr: "invalid target: url.value.something",
		},
		{
			name:     "valid url.params",
			param:    "url.params.foo",
			expected: targetInfo{Type: "url.params", Name: "foo"},
		},
		{
			name:        "invalid url.params",
			param:       "url.params",
			expectedErr: "invalid target: url.params",
		},
		{
			name:     "valid header",
			param:    "header.foo",
			expected: targetInfo{Type: "header", Name: "foo"},
		},
		{
			name:     "valid body",
			param:    "body.foo.bar",
			expected: targetInfo{Type: "body", Name: "foo.bar"},
		},
		{
			name:        "invalid target: missing part",
			param:       "header",
			expectedErr: "invalid target: header",
		},
		{
			name:        "invalid target: unknown",
			param:       "unknown.foo",
			expectedErr: "invalid target: unknown.foo",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, gotErr := getTargetInfo(tc.param)
			if tc.expectedErr == "" {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.expected, got)
			} else {
				assert.EqualError(t, gotErr, tc.expectedErr)
			}
		})
	}
}
