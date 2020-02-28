// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package appender

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/urso/ecslog/backend"
	"github.com/urso/ecslog/backend/layout"
)

type file struct {
	f          *os.File
	buf        *bufio.Writer
	lvl        backend.Level
	mu         sync.Mutex
	layout     layout.Layout
	forceFlush bool
}

func File(
	lvl backend.Level,
	path string,
	perm os.FileMode,
	layout layout.Factory,
	bufferSize int,
	immediateFlush bool,
) (backend.Backend, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return nil, err
	}

	var buf *bufio.Writer
	var out io.Writer = f
	if bufferSize >= 0 {
		buf = bufio.NewWriterSize(f, bufferSize)
		out = buf
	}

	l, err := layout(out)
	if err != nil {
		return nil, err
	}

	return &file{
		f:          f,
		lvl:        lvl,
		buf:        buf,
		layout:     l,
		forceFlush: buf != nil && immediateFlush,
	}, nil
}

func (f *file) For(name string) backend.Backend {
	return f
}

func (f *file) IsEnabled(lvl backend.Level) bool {
	return lvl >= f.lvl
}

func (f *file) UseContext() bool {
	return f.layout.UseContext()
}

func (f *file) Log(msg backend.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.layout.Log(msg)
	if !f.forceFlush {
		return
	}

	f.buf.Flush()
}
