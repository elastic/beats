// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSubpath(t *testing.T) {
	testCases := []struct {
		root       string
		path       string
		resultPath string
		isSubpath  bool
	}{
		{"/", "a", "/a", true},
		{"/a", "b", "/a/b", true},
		{"/a", "b/c", "/a/b/c", true},
		{"/a/b", "/a/c", "/a/c", false},
		{"/a/b", "/a/b/../c", "/a/c", false},
		{"/a/b", "../c", "/a/c", false},
	}

	for _, test := range testCases {
		t.Run(fmt.Sprintf("'%s'-'%s'", test.root, test.path), func(t *testing.T) {
			newPath, result := joinPaths(test.root, test.path)
			assert.Equal(t, test.resultPath, newPath)
			assert.Equal(t, test.isSubpath, result)
		})
	}

}
