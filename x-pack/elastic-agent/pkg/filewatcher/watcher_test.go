// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filewatcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatch(t *testing.T) {
	t.Run("no files are watched", withWatch(func(t *testing.T, w *Watch) {
		r, u, err := w.scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
		assert.Equal(t, 0, len(u))
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

		r, _, err := w.scan()
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

		r, u, err := w.scan()
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, r[0], path)
		assert.Equal(t, 0, len(u))

		r, u, err = w.scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
		assert.Equal(t, 1, len(u))
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
		r, u, err := w.scan()
		require.NoError(t, err)
		assert.Equal(t, 1, len(r))
		assert.Equal(t, r[0], path)
		assert.Equal(t, 0, len(u))

		// Should not be returned since it's not modified.
		r, u, err = w.scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
		assert.Equal(t, 1, len(u))

		// Unwatch the file
		w.Unwatch(path)

		// Add new content to the file.
		ioutil.WriteFile(path, []byte("heeeelo"), 0644)

		// Should not find the file.
		r, u, err = w.scan()
		require.NoError(t, err)
		assert.Equal(t, 0, len(r))
		assert.Equal(t, 0, len(u))
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

	t.Run("update returns updated, unchanged and watched files", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path1 := filepath.Join(tmp, "hello-1.txt")
		empty, err := os.Create(path1)
		require.NoError(t, err)
		empty.Close()

		// Register the file to watch.
		w.Watch(path1)

		path2 := filepath.Join(tmp, "hello-2.txt")
		empty, err = os.Create(path2)
		require.NoError(t, err)
		empty.Close()

		w.Watch(path2)

		path3 := filepath.Join(tmp, "hello-3.txt")
		empty, err = os.Create(path3)
		require.NoError(t, err)
		empty.Close()

		w.Watch(path3)

		// Set initial state
		w.Update()

		// Reset watched files.
		w.Reset()

		// readd files
		w.Watch(path2)
		w.Watch(path3)

		// Try as much as possible to have content on disk.
		<-time.After(1 * time.Second)
		// Add new content to the file.
		f, err := os.OpenFile(path3, os.O_APPEND|os.O_WRONLY, 0600)
		require.NoError(t, err)
		f.Write([]byte("more-hello"))
		require.NoError(t, f.Sync())
		f.Close()

		s, _ := w.Update()

		require.Equal(t, 1, len(s.Updated))
		require.Equal(t, 1, len(s.Unchanged))
		require.Equal(t, 1, len(s.Unwatched))

		require.True(t, s.NeedUpdate)

		assert.Equal(t, path1, s.Unwatched[0])
		assert.Equal(t, path3, s.Updated[0])
		assert.Equal(t, path2, s.Unchanged[0])
	}))

	t.Run("should cleanup files that disapear", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path1 := filepath.Join(tmp, "hello.txt")
		empty, err := os.Create(path1)
		require.NoError(t, err)
		empty.Close()

		w.Watch(path1)
		require.True(t, w.IsWatching(path1))
		w.Reset()
		w.Cleanup()
		require.False(t, w.IsWatching(path1))
	}))

	t.Run("should allow to invalidate the cache ", withWatch(func(t *testing.T, w *Watch) {
		tmp, err := ioutil.TempDir("", "watch")
		require.NoError(t, err)
		defer os.RemoveAll(tmp)

		path1 := filepath.Join(tmp, "hello.txt")
		empty, err := os.Create(path1)
		require.NoError(t, err)
		empty.Close()

		w.Watch(path1)
		require.True(t, w.IsWatching(path1))
		w.Invalidate()
		require.True(t, len(w.Watched()) == 0)
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
