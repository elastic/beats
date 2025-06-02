// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package utils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddInt64OrNull(t *testing.T) {
	const unchangedBase int64 = 123
	const changedBase int64 = 456

	unchanged := int64(unchangedBase)
	changed := int64(changedBase)

	require.Nil(t, AddInt64OrNull(nil, nil))
	require.Same(t, &unchanged, AddInt64OrNull(&unchanged, nil))
	require.NotSame(t, &unchanged, AddInt64OrNull(nil, &unchanged))
	require.Equal(t, unchanged, *AddInt64OrNull(nil, &unchanged))
	require.Equal(t, unchangedBase, unchanged)
	require.Same(t, &changed, AddInt64OrNull(&changed, &unchanged))
	require.Equal(t, changedBase+unchanged*2, *AddInt64OrNull(&changed, &unchanged))
}

func TestParseArrayOfStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Standard comma-separated input",
			input:    "[a,b,c]",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Input with spaces",
			input:    "[ a , b , c ]",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Single element",
			input:    "[value]",
			expected: []string{"value"},
		},
		{
			name:     "Single element without brackets",
			input:    " value",
			expected: []string{"value"},
		},
		{
			name:     "Empty brackets",
			input:    "[]",
			expected: []string{},
		},
		{
			name:     "Brackets with only spaces",
			input:    "[   ]",
			expected: []string{},
		},
		{
			name:     "Input without brackets (edge case)",
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Leading comma",
			input:    "[,a,b]",
			expected: []string{"a", "b"},
		},
		{
			name:     "Trailing comma",
			input:    "[a,b,c,]",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Only commas",
			input:    "[,,]",
			expected: []string{},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseArrayOfStrings(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseArrayOfStrings(%q) = %#v; expected %#v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnyMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		patterns []string
		expected bool
	}{
		{
			name:     "One value matches pattern",
			values:   []string{".monitoring-alerts-7", "metrics"},
			patterns: []string{".monitoring-a*"},
			expected: true,
		},
		{
			name:     "Multiple values, one matches prefix pattern",
			values:   []string{"apm-server", "metrics"},
			patterns: []string{"apm*"},
			expected: true,
		},
		{
			name:     "No values match patterns",
			values:   []string{"security", "kibana"},
			patterns: []string{"logs*", "traces*"},
			expected: false,
		},
		{
			name:     "Empty values slice",
			values:   []string{},
			patterns: []string{"*"},
			expected: false,
		},
		{
			name:     "Empty patterns slice",
			values:   []string{"any"},
			patterns: []string{},
			expected: false,
		},
		{
			name:     "Both slices empty",
			values:   []string{},
			patterns: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnyMatchesAnyPattern(tt.values, tt.patterns)
			if result != tt.expected {
				t.Errorf("AnyMatchesAnyPattern(%v, %v) = %v; expected %v", tt.values, tt.patterns, result, tt.expected)
			}
		})
	}
}
