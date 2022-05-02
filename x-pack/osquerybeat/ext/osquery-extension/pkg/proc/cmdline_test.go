// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package proc

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

var cmdData = `/usr/bin/processName --config=/etc/conf/config.conf`

func TestReadCmdLineFS(t *testing.T) {
	pid := "1"
	testCases := []struct {
		fstest fstest.MapFS
		assert func(string, error)
	}{
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
				"proc/1/cmdline": {
					Data: []byte(cmdData),
				},
				"proc/1/stat": {
					Data: []byte("some data"),
				},
			},
			func(result string, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result, cmdData)
			}},
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
			},
			func(result string, err error) {
				assert.Error(t, err)
				assert.Equal(t, result, "")
			}},
	}

	for _, testCase := range testCases {
		result, err := ReadCmdLineFS(testCase.fstest, pid)
		testCase.assert(result, err)
	}
}
