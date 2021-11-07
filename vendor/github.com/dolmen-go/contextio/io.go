/*
Copyright 2018 Olivier Mengué

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package contextio provides Writer and Reader that stop accepting/providing
// data when an attached context is canceled.
package contextio

import (
	"context"
	"io"
)

type writer struct {
	ctx context.Context
	w   io.Writer
}

type copier struct {
	writer
}

// NewWriter wraps an io.Writer to handle context cancellation.
//
// Context state is checked BEFORE every Write.
//
// The returned Writer also implements io.ReaderFrom to allow io.Copy to select
// the best strategy while still checking the context state before every chunk transfer.
func NewWriter(ctx context.Context, w io.Writer) io.Writer {
	if w, ok := w.(*copier); ok && ctx == w.ctx {
		return w
	}
	return &copier{writer{ctx: ctx, w: w}}
}

// Write implements io.Writer, but with context awareness.
func (w *writer) Write(p []byte) (n int, err error) {
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
		return w.w.Write(p)
	}
}

type reader struct {
	ctx context.Context
	r   io.Reader
}

// NewReader wraps an io.Reader to handle context cancellation.
//
// Context state is checked BEFORE every Read.
func NewReader(ctx context.Context, r io.Reader) io.Reader {
	if r, ok := r.(*reader); ok && ctx == r.ctx {
		return r
	}
	return &reader{ctx: ctx, r: r}
}

func (r *reader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}

// ReadFrom implements interface io.ReaderFrom, but with context awareness.
//
// This should allow efficient copying allowing writer or reader to define the chunk size.
func (w *copier) ReadFrom(r io.Reader) (n int64, err error) {
	if _, ok := w.w.(io.ReaderFrom); ok {
		// Let the original Writer decide the chunk size.
		return io.Copy(w.writer.w, &reader{ctx: w.ctx, r: r})
	}
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
		// The original Writer is not a ReaderFrom.
		// Let the Reader decide the chunk size.
		return io.Copy(&w.writer, r)
	}
}

// NewCloser wraps an io.Reader to handle context cancellation.
//
// Context state is checked BEFORE any Close.
func NewCloser(ctx context.Context, c io.Closer) io.Closer {
	return &closer{ctx: ctx, c: c}
}

type closer struct {
	ctx context.Context
	c   io.Closer
}

func (c *closer) Close() error {
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
		return c.c.Close()
	}
}
