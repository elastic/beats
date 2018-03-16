// Package txfiletest provides utilities for testing on top of txfile.
package txfiletest

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/elastic/go-txfile"
	"github.com/elastic/go-txfile/internal/cleanup"
)

// TestFile wraps a txfile.File structure for testing.
type TestFile struct {
	*txfile.File
	t    testT
	Path string
	opts txfile.Options
}

type testT interface {
	Error(...interface{})
	Fatal(...interface{})
}

// SetupTestFile creates a new testfile in a temporary directory.
// The teardown function will remove the directory and the temporary file.
func SetupTestFile(t testT, opts txfile.Options) (tf *TestFile, teardown func()) {
	if opts.PageSize == 0 {
		opts.PageSize = 4096
	}

	ok := false
	path, cleanPath := SetupPath(t, "")
	defer cleanup.IfNot(&ok, cleanPath)

	tf = &TestFile{Path: path, t: t, opts: opts}
	tf.Open()

	ok = true
	return tf, func() {
		tf.Close()
		cleanPath()
	}
}

// Reopen tries to close and open the file again.
func (f *TestFile) Reopen() {
	f.Close()
	f.Open()
}

// Close the test file.
func (f *TestFile) Close() {
	if f.File != nil {
		if err := f.File.Close(); err != nil {
			f.t.Fatal("close failed on reopen")
		}
		f.File = nil
	}
}

// Open opens the file if it has been closed.
// The File pointer will be changed.
func (f *TestFile) Open() {
	if f.File != nil {
		return
	}

	tmp, err := txfile.Open(f.Path, os.ModePerm, f.opts)
	if err != nil {
		f.t.Fatal("reopen failed")
	}
	f.File = tmp
}

// SetupPath creates a temporary directory for testing.
// Use the teardown function to remove the directory again.
func SetupPath(t testT, file string) (dir string, teardown func()) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	if file == "" {
		file = "test.dat"
	}
	return path.Join(dir, file), func() {
		os.RemoveAll(dir)
	}
}
