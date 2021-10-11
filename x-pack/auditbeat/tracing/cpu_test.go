// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux
// +build linux

package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCPUSetFromExpression(t *testing.T) {
	for _, testCase := range []struct {
		content string
		result  CPUSet
		fail    bool
	}{
		{
			content: "0",
			result: CPUSet{
				mask:  []bool{true},
				count: 1,
			},
		},
		{
			content: "0-3",
			result: CPUSet{
				mask:  []bool{true, true, true, true},
				count: 4,
			},
		},
		{
			content: "5-0",
			fail:    true,
		},
		{
			content: "5-2147483648",
			fail:    true,
		},
		{
			content: "0,2-2",
			result: CPUSet{
				mask:  []bool{true, false, true},
				count: 2,
			},
		},
		{
			content: "7",
			result: CPUSet{
				mask:  []bool{false, false, false, false, false, false, false, true},
				count: 1,
			},
		},
		{
			content: "-1",
			fail:    true,
		},
		{
			content: "",
		},
		{
			content: ",",
		},
		{
			content: "-",
			fail:    true,
		},
		{
			content: "3,-",
			fail:    true,
		},
		{
			content: "3-4-5",
			fail:    true,
		},
		{
			content: "0-4,5,6-6,,,,15",
			result: CPUSet{
				mask: []bool{
					true, true, true, true, true, true, true, false,
					false, false, false, false, false, false, false, true,
				},
				count: 8,
			},
		},
	} {
		mask, err := NewCPUSetFromExpression(testCase.content)
		if !assert.Equal(t, testCase.fail, err != nil, testCase.content) {
			t.Fatal(err)
		}
		assert.Equal(t, testCase.result, mask, testCase.content)
	}
}

func TestCPUSet(t *testing.T) {
	for _, test := range []struct {
		expr  string
		num   int
		isSet func(int) bool
		list  []int
	}{
		{
			expr:  "0-2,5",
			num:   4,
			isSet: func(i int) bool { return i == 5 || (i >= 0 && i < 3) },
			list:  []int{0, 1, 2, 5},
		},
		{
			expr:  "0",
			num:   1,
			isSet: func(i int) bool { return i == 0 },
			list:  []int{0},
		},
		{
			expr:  "2",
			num:   1,
			isSet: func(i int) bool { return i == 2 },
			list:  []int{2},
		},
		{
			expr:  "0-7",
			num:   8,
			isSet: func(i int) bool { return i >= 0 && i < 8 },
			list:  []int{0, 1, 2, 3, 4, 5, 6, 7},
		},
		{
			expr:  "",
			num:   0,
			isSet: func(i int) bool { return false },
			list:  []int{},
		},
		{
			expr:  "1-2,0,2,0-0,0-1",
			num:   3,
			isSet: func(i int) bool { return i >= 0 && i < 3 },
			list:  []int{0, 1, 2},
		},
	} {
		set, err := NewCPUSetFromExpression(test.expr)
		if !assert.NoError(t, err, test.expr) {
			t.Fatal(err)
		}
		assert.Equal(t, test.num, set.NumCPU(), test.expr)
		for i := -1; i < 10; i++ {
			assert.Equal(t, test.isSet(i), set.Contains(i), test.expr)
		}
		assert.Equal(t, test.list, set.AsList(), test.expr)
	}
}
