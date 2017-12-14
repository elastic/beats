package file_integrity

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner(t *testing.T) {
	dir := setupTestDir(t)
	defer os.RemoveAll(dir)

	config := defaultConfig
	config.Paths = []string{
		dir,
		filepath.Join(dir, "a"),
		"/does/not/exist",
	}

	t.Run("non-recursive", func(t *testing.T) {
		reader, err := NewFileSystemScanner(config)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		var events []Event
		for event := range eventC {
			events = append(events, event)
		}
		assert.Len(t, events, 7)
	})

	t.Run("recursive", func(t *testing.T) {
		c := config
		c.Recursive = true

		reader, err := NewFileSystemScanner(c)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		var foundRecursivePath bool

		var events []Event
		for event := range eventC {
			events = append(events, event)
			if filepath.Base(event.Path) == "c" {
				foundRecursivePath = true
			}
		}

		assert.Len(t, events, 8)
		assert.True(t, foundRecursivePath, "expected subdir/c to be included")
	})

	// This smoke tests the rate limit code path, but does not validate the rate.
	t.Run("with rate limit", func(t *testing.T) {
		c := config
		c.ScanRateBytesPerSec = 1024 * 5

		reader, err := NewFileSystemScanner(c)
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		defer close(done)

		eventC, err := reader.Start(done)
		if err != nil {
			t.Fatal(err)
		}

		if err != nil {
			t.Fatal(err)
		}

		var events []Event
		for event := range eventC {
			events = append(events, event)
		}

		assert.Len(t, events, 7)
	})
}

func setupTestDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "audit-file-scan")
	if err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(filepath.Join(dir, "a"), []byte("file a"), 0600); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(filepath.Join(dir, "b"), []byte("file b"), 0600); err != nil {
		t.Fatal(err)
	}

	if err = os.Symlink(filepath.Join(dir, "b"), filepath.Join(dir, "link_to_b")); err != nil {
		t.Fatal(err)
	}

	if err = os.Mkdir(filepath.Join(dir, "subdir"), 0700); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(filepath.Join(dir, "subdir", "c"), []byte("file c"), 0600); err != nil {
		t.Fatal(err)
	}

	if err = os.Symlink(filepath.Join(dir, "subdir"), filepath.Join(dir, "link_to_subdir")); err != nil {
		t.Fatal(err)
	}

	return dir
}
