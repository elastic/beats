package kprobes

import (
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Test_InotifyWatcher(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping on non-linux")
	}

	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	watcher, err := newInotifyWatcher()
	require.NoError(t, err)

	added, err := watcher.Add(1, 1, tmpDir)
	require.NoError(t, err)
	require.True(t, added)

	added, err = watcher.Add(1, 1, filepath.Join(tmpDir, "test"))
	require.NoError(t, err)
	require.False(t, added)

	added, err = watcher.Add(2, 2, tmpDir)
	require.NoError(t, err)
	require.False(t, added)

	tmpDir2, err := os.MkdirTemp("", "kprobe_unit_test")
	defer func() {
		_ = os.RemoveAll(tmpDir2)
	}()
	added, err = watcher.Add(2, 2, tmpDir2)
	require.NoError(t, err)
	require.True(t, added)

	require.NoError(t, watcher.Close())

	_, err = watcher.Add(1, 1, tmpDir)
	require.Error(t, err)
}

func Test_InotifyWatcher_Add_Err(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping on non-linux")
	}

	watcher, err := newInotifyWatcher()
	require.NoError(t, err)

	inotifyAddWatch = func(fd int, pathname string, mask uint32) (int, error) {
		return -1, os.ErrInvalid
	}
	defer func() {
		inotifyAddWatch = unix.InotifyAddWatch
	}()

	_, err = watcher.Add(1, 1, "non_existent")
	require.ErrorIs(t, err, os.ErrInvalid)

	require.NoError(t, watcher.Close())
}

func Test_InotifyWatcher_Close_Err(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping on non-linux")
	}

	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	watcher, err := newInotifyWatcher()
	require.NoError(t, err)

	added, err := watcher.Add(1, 1, tmpDir)
	require.NoError(t, err)
	require.True(t, added)

	err = os.RemoveAll(tmpDir)
	require.NoError(t, err)

	require.Error(t, watcher.Close())
}
