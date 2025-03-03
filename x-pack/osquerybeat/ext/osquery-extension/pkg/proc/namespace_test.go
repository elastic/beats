// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package proc

import (
	"os"
	"syscall"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

var nsIno = uint64(26041992)

func TestNamespaceFS(t *testing.T) {
	pid := "1"
	testCases := []struct {
		fstest fstest.MapFS
		assert func(NamespaceInfo, error)
	}{
		{
			fstest.MapFS{
				"proc/1/ns/pid": {
					Sys: dummyStat(t, nsIno),
				},
			},
			func(result NamespaceInfo, err error) {
				assert.Nil(t, err)
				assert.Equal(t, result.Ino, nsIno)
			}},
		{
			fstest.MapFS{
				"proc/2/ns/pid": {
					Sys: dummyStat(t, nsIno),
				},
			},
			func(result NamespaceInfo, err error) {
				assert.Error(t, err)
			}},
	}

	for _, testCase := range testCases {
		result, err := ReadNamespaceFS(testCase.fstest, pid)
		testCase.assert(result, err)
	}
}

// Used in order to get a mocked syscall.Stat_t structure with assigned ino
func dummyStat(t *testing.T, ino uint64) *syscall.Stat_t {
	name := t.TempDir()

	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}

	mockDsStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal(err)
	}

	mockDsStat.Ino = ino

	return mockDsStat
}
