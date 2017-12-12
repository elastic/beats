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
	if recursive {
		return newRecursiveWatcher(fsnotify), nil
	}
	return (*nonRecursiveWatcher)(fsnotify), nil
}
