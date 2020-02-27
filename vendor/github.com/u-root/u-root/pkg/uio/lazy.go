// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uio

import (
	"io"
	"os"
)

// LazyOpener is a lazy io.Reader.
//
// LazyOpener will use a given open function to derive an io.Reader when Read
// is first called on the LazyOpener.
type LazyOpener struct {
	r    io.Reader
	err  error
	open func() (io.Reader, error)
}

// NewLazyOpener returns a lazy io.Reader based on `open`.
func NewLazyOpener(open func() (io.Reader, error)) io.ReadCloser {
	return &LazyOpener{open: open}
}

// Read implements io.Reader.Read lazily.
//
// If called for the first time, the underlying reader will be obtained and
// then used for the first and subsequent calls to Read.
func (lr *LazyOpener) Read(p []byte) (int, error) {
	if lr.r == nil && lr.err == nil {
		lr.r, lr.err = lr.open()
	}
	if lr.err != nil {
		return 0, lr.err
	}
	return lr.r.Read(p)
}

// Close implements io.Closer.Close.
func (lr *LazyOpener) Close() error {
	if c, ok := lr.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// ReadAtCloser is an io.ReaderAt and an io.Closer.
type ReadAtCloser interface {
	io.ReaderAt
	io.Closer
}

// LazyOpenerAt is a lazy io.ReaderAt.
//
// LazyOpenerAt will use a given open function to derive an io.ReaderAt when
// ReadAt is first called.
type LazyOpenerAt struct {
	r    io.ReaderAt
	err  error
	open func() (io.ReaderAt, error)
}

// NewLazyFile returns a lazy ReaderAt opened from path.
func NewLazyFile(path string) ReadAtCloser {
	if len(path) == 0 {
		return nil
	}
	return NewLazyOpenerAt(func() (io.ReaderAt, error) {
		return os.Open(path)
	})
}

// NewLazyOpenerAt returns a lazy io.ReaderAt based on `open`.
func NewLazyOpenerAt(open func() (io.ReaderAt, error)) ReadAtCloser {
	return &LazyOpenerAt{open: open}
}

// ReadAt implements io.ReaderAt.ReadAt.
func (loa *LazyOpenerAt) ReadAt(p []byte, off int64) (int, error) {
	if loa.r == nil && loa.err == nil {
		loa.r, loa.err = loa.open()
	}
	if loa.err != nil {
		return 0, loa.err
	}
	return loa.r.ReadAt(p, off)
}

// Close implements io.Closer.Close.
func (loa *LazyOpenerAt) Close() error {
	if c, ok := loa.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
