package monitor

import (
	"github.com/fsnotify/fsnotify"
)

// Watcher is an interface for a file watcher akin to fsnotify.Watcher
// with an additional Start method.
type Watcher interface {
	Add(path string) error
	Close() error
	EventChannel() <-chan fsnotify.Event
	ErrorChannel() <-chan error
	Start() error
}

// New creates a new Watcher backed by fsnotify with optional recursive
// logic.
func New(recursive bool) (Watcher, error) {
	fsnotify, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// Use our simulated recursive watches unless the fsnotify implementation
	// supports OS-provided recursive watches
	if recursive && fsnotify.SetRecursive() != nil {
		return newRecursiveWatcher(fsnotify), nil
	}
	return (*nonRecursiveWatcher)(fsnotify), nil
}
