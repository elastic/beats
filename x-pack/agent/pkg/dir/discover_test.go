// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dir

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {
	t.Run("support wildcards patterns", withFiles([]string{"hello", "helllooo"}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(filepath.Join(dst, "hel*"))
		require.NoError(t, err)
		assert.Equal(t, 2, len(r))
	}))

	t.Run("support direct file", withFiles([]string{"hello", "helllooo"}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(filepath.Join(dst, "hello"))
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
	}))

	t.Run("support direct file and pattern", withFiles([]string{"hello", "helllooo", "agent.yml"}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(
			filepath.Join(dst, "hel*"),
			filepath.Join(dst, "agent.yml"),
		)
		require.NoError(t, err)
		assert.Equal(t, 3, len(r))
	}))

	t.Run("support direct file and pattern", withFiles([]string{"hello", "helllooo", "agent.yml"}, func(
		dst string,
		t *testing.T,
	) {
		r, err := DiscoverFiles(filepath.Join(dst, "donotmatch.yml"))
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
	}))
}

func withFiles(files []string, fn func(dst string, t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		tmp, _ := ioutil.TempDir("", "watch")
		defer os.RemoveAll(tmp)

		for _, file := range files {
			path := filepath.Join(tmp, file)
			empty, _ := os.Create(path)
			empty.Close()
		}

		fn(tmp, t)
	}
}
