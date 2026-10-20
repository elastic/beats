// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func receiverTestHome(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return t.TempDir()
	}
	// macOS's default TMPDIR can exceed the Unix socket limit once osquery
	// appends its extension route. Keep the fallback socket base private and short.
	home, err := os.MkdirTemp("/tmp", "osq-")
	require.NoError(t, err, "create a short private receiver home")
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(home), "remove receiver home after shutdown")
	})
	t.Setenv("TMPDIR", home)
	return home
}
