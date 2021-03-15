// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package paths

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	isWindows := runtime.GOOS == "windows"
	testCases := []struct {
		Name        string
		Expected    string
		Actual      string
		ShouldMatch bool
	}{
		{"different paths", "/var/path/a", "/var/path/b", false},
		{"strictly same paths", "/var/path/a", "/var/path/a", true},
		{"strictly same win paths", `C:\Program Files\Elastic\Agent`, `C:\Program Files\Elastic\Agent`, true},
		{"case insensitive win paths", `C:\Program Files\Elastic\Agent`, `c:\Program Files\Elastic\Agent`, isWindows},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.ShouldMatch, ArePathsEqual(tc.Expected, tc.Actual))
		})
	}
}
