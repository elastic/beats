// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	jsonByte := []byte("{'test':\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}")

	testCases := []struct {
		name     string
		opts     []string
		expected []byte
	}{
		{
			name:     "no options",
			opts:     []string{},
			expected: jsonByte,
		},
		{
			name:     "NEW_LINES option",
			opts:     []string{"NEW_LINES"},
			expected: []byte("{'test':\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:     "SINGLE_QUOTES option",
			opts:     []string{"SINGLE_QUOTES"},
			expected: []byte("{\"test\":\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:     "both options",
			opts:     []string{"NEW_LINES", "SINGLE_QUOTES"},
			expected: []byte("{\"test\":\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
	}

	// Run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := sanitize(jsonByte, tc.opts...)
			assert.Equal(t, tc.expected, res)
		})
	}
}
