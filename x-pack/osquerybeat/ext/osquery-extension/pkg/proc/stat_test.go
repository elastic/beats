// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
)

var stat = "6462 (bash) S 6402 6462 6462 34817 37849 4194304 14126 901131 0 191 15 9 3401 725 20 0 1 0 134150 20156416 1369 18446744073709551615 94186936238080 94186936960773 140723699470016 0 0 0 65536 3670020 1266777851 1 0 0 17 7 0 0 0 0 0 94186937191664 94186937239044 94186967023616 140723699476902 140723699476912 140723699476912 140723699478510 0"
var status = `Name:   myProcess`

func TestReadStatFS(t *testing.T) {
	pid := "1"
	testCases := []struct {
		fstest fstest.MapFS
		assert func(ProcStat, error)
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
					Data: []byte(stat),
				},
				"proc/1/status": {
					Data: []byte(status),
				},
				"proc/2/stat": {
					Data: []byte("some data"),
				},
			},
			func(result ProcStat, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result.Name, "myProcess")
			}},
		{
			fstest.MapFS{
				"hello.txt": {
					Data: []byte("hello, world"),
				},
			},
			func(result ProcStat, err error) {
				assert.Error(t, err)
			}},
		{
			fstest.MapFS{
				"proc/1/stat": {
					Data: []byte(stat),
				},
			},
			func(results ProcStat, err error) {
				assert.Error(t, err)
			}},
	}

	for _, testCase := range testCases {
		result, err := ReadStatFS(testCase.fstest, pid)
		testCase.assert(result, err)
	}
}
