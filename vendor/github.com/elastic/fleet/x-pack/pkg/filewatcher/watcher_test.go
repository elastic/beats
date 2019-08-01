// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filewatcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	t.Run("no files are watched", withWatch(func(t *testing.T, w *Watch) {
		r, err := w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
	}))

	t.Run("newly added files are discovered", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path := filepath.Join(tmp, "hello.txt")
		empty, err := os.Create(path)
		require.NoError(t, err)
		empty.Close()

		// Register the file to watch.
		w.Watch(path)

		r, err := w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, r[0], path)
	}))

	t.Run("ignore old files", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path := filepath.Join(tmp, "hello.txt")
		empty, err := os.Create(path)
		require.NoError(t, err)
		empty.Close()

		// Register the file to watch.
		w.Watch(path)

		r, err := w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, r[0], path)

		r, err = w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
	}))

	t.Run("can unwatch a watched file", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path := filepath.Join(tmp, "hello.txt")
		empty, err := os.Create(path)
		require.NoError(t, err)
		empty.Close()

		// Register the file to watch.
		w.Watch(path)

		// Initiall found
		r, err := w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, r[0], path)

		// Should not be returned since it's not modified.
		r, err = w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))

		// Unwatch the file
		w.Unwatch(path)

		// Add new content to the file.
		ioutil.WriteFile(path, []byte("heeeelo"), 0644)

		// Should not find the file.
		r, err = w.Scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
	}))

	t.Run("can returns the list of watched files", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path := filepath.Join(tmp, "hello.txt")
		empty, err := os.Create(path)
		require.NoError(t, err)
		empty.Close()

		// Register the file to watch.
		w.Watch(path)

		assert.Equal(t, 1, len(w.Watched()))
		assert.Equal(t, path, w.Watched()[0])
		assert.True(t, w.IsWatching(path))
	}))
}

func withWatch(fn func(t *testing.T, w *Watch)) func(*testing.T) {
	return func(t *testing.T) {
		w, err := New(nil, DefaultComparer)
		if !assert.NoError(t, err) {
			return
		}
		fn(t, w)
	}
}
