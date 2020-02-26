package appender

import (
	"io"
	"os"
	"sync"

	"github.com/urso/ecslog/backend"
	"github.com/urso/ecslog/backend/layout"
)

type writer struct {
	mu         sync.Mutex
	out        io.Writer
	lvl        backend.Level
	layout     layout.Layout
	forceFlush bool
}

func NewWriter(out io.Writer, lvl backend.Level, layout layout.Factory, forceFlush bool) (backend.Backend, error) {
	l, err := layout(out)
	if err != nil {
		return nil, err
	}

	return &writer{
		out:        out,
		lvl:        lvl,
		layout:     l,
		forceFlush: forceFlush,
	}, nil
}

func Console(lvl backend.Level, layout layout.Factory) (backend.Backend, error) {
	return NewWriter(os.Stderr, lvl, layout, true)
}

func (w *writer) For(name string) backend.Backend {
	return w
}

func (w *writer) IsEnabled(lvl backend.Level) bool {
	return lvl >= w.lvl
}

func (w *writer) UseContext() bool {
	return w.layout.UseContext()
}

func (w *writer) Log(msg backend.Message) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.layout.Log(msg)
	if !w.forceFlush {
		return
	}

	// flush if output is buffered
	switch f := w.out.(type) {
	case interface{ Flush() error }:
		f.Flush()

	case interface{ Flush() bool }:
		f.Flush()

	case interface{ Flush() }:
		f.Flush()
	}
}
