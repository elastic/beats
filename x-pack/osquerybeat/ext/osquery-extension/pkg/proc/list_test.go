// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
)

func TestList(t *testing.T) {
	testCases := []struct {
		fstest fstest.MapFS
		assert func([]string, error)
	}{
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
				"proc/1/cmdline": {
					Data: []byte("some data"),
				},
				"proc/1/stat": {
					Data: []byte("some data"),
				},
				"proc/2/stat": {
					Data: []byte("some data"),
				},
			},
			func(results []string, err error) {
				assert.Nil(t, err)
				assert.Len(t, results, 2)
				assert.Contains(t, results, "1")
				assert.Contains(t, results, "2")
			}},
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
			},
			func(results []string, err error) {
				assert.Error(t, err)
				assert.Nil(t, results)
			}},
		{
			fstest.MapFS{
				"proc/uptime": {
					Data: []byte("hello, world"),
				},
			},
			func(results []string, err error) {
				assert.Nil(t, err)
				assert.Nil(t, results)
			}},
	}

	for _, testCase := range testCases {
		result, err := ListFS(testCase.fstest)
		testCase.assert(result, err)
	}
}
