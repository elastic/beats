// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersionVersion(t *testing.T) {
	tests := []struct {
		given    string
		expected string
	}{{
		given:    ">2.1.1",
		expected: "2.1.1",
	},
		{
			given:    "^0.0.1-alpha.preview+123.github",
			expected: "0.0.1",
		},
		{
			given:    "<=0.0.1-alpha.12",
			expected: "0.0.1",
		},
		{
			given:    "^1.0.3-beta",
			expected: "1.0.3",
		},
		{
			given:    "~^1.0.3",
			expected: "1.0.3",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("expected version %s does not match given %s", tt.expected, tt.given), func(t *testing.T) {
			parsed := parseVersion(tt.given)
			require.Equal(t, tt.expected, parsed)
		})
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		expected  string
		current   string
		shouldErr bool
	}{
		{
			expected:  "<2.0.0",
			current:   "^1.1.1",
			shouldErr: false,
		},
		{
			expected:  "<2.0.0",
			current:   "=2.1.1",
			shouldErr: true,
		},
		{
			expected:  "<2.0.0",
			current:   "2.0.0",
			shouldErr: true,
		},
		{
			expected:  "<1.0.0",
			current:   "0.0.1-alpha.11",
			shouldErr: false,
		},
		{
			expected:  "",
			current:   "file://blahblahblah",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("match expected %s with current %s version", tt.expected, tt.current), func(t *testing.T) {
			err := validateVersion(tt.expected, tt.current)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.Equal(t, nil, err)
			}
		})
	}
}
