// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
)

var uptime = "179118.54 1008710.15"

func TestReadUpTimeFS(t *testing.T) {
	testCases := []struct {
		fstest fstest.MapFS
		assert func(int64, error)
	}{
		{
			fstest.MapFS{
				"proc/uptime": {
					Data: []byte(uptime),
				},
			},
			func(result int64, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result, 179118)
			}},
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
			},
			func(result int64, err error) {
				assert.Error(t, err)
			}},
	}

	for _, testCase := range testCases {
		result, err := ReadUptimeFS(testCase.fstest)
		testCase.assert(result, err)
	}
}
