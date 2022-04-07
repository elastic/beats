// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/elastic/beats/v8/x-pack/heartbeat/monitors/browser/source/fixtures"

	"github.com/stretchr/testify/require"
)

func TestLocalSourceValidate(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	fixtureDir := path.Join(filepath.Dir(filename), "fixtures/todos")
	tests := []struct {
		name     string
		OrigPath string
		err      error
	}{
		{"valid", fixtureDir, nil},
		{"invalid", "/not/a/path", ErrInvalidPath("/not/a/path")},
		{"nopath", "", ErrNoPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LocalSource{OrigPath: tt.OrigPath}
			err := l.Validate()
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				require.Regexp(t, tt.err, err)
			}
		})
	}
}

func TestLocalSourceLifeCycle(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	origPath := path.Join(filepath.Dir(filename), "fixtures/todos")
	ls := LocalSource{OrigPath: origPath}
	require.NoError(t, ls.Validate())

	// Don't run the NPM commands in unit tests
	// We can leave that for E2E tests
	GoOffline()
	defer GoOnline()
	require.NoError(t, ls.Fetch())

	require.NotEmpty(t, ls.workingPath)
	fixtures.TestTodosFiles(t, ls.workingPath)

	require.NoError(t, ls.Close())
	_, err := os.Stat(ls.Workdir())
	require.True(t, os.IsNotExist(err), "Workdir %s should have been deleted", ls.Workdir())
}
