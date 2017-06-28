// +build linux freebsd openbsd netbsd windows

package file

import (
	"time"

	"github.com/fsnotify/fsnotify"
)

func NewEventReader(c Config) (EventReader, error) {
	var paths []string
	for _, filePaths := range c.Paths {
		paths = append(paths, filePaths...)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, p := range paths {
		if err := watcher.Add(p); err != nil {
			watcher.Close()
			return nil, err
		}
	}

	return &reader{watcher, c.MaxFileSize, make(chan Event, 1), make(chan error, 1)}, nil
}

type reader struct {
	watcher     *fsnotify.Watcher
	maxFileSize int64
	outC        chan Event
	errC        chan error
}

func (r *reader) Start(done <-chan struct{}) (<-chan Event, error) {
	go func() {
		defer r.watcher.Close()

		for {
			select {
			case event := <-r.watcher.Events:
				r.outC <- convertToFileEvent(event, r.maxFileSize)
			case err := <-r.watcher.Errors:
				r.errC <- err
			}
		}
	}()

	return r.outC, nil
}

func convertToFileEvent(e fsnotify.Event, maxFileSize int64) Event {
	event := Event{
		Timestamp: time.Now(),
		Path:      e.Name,
		Action:    opToAction(e.Op).String(),
	}

	addFileAttributes(&event, maxFileSize)

	return event
}

func opToAction(op fsnotify.Op) Action {
	switch op {
	case fsnotify.Create:
		return Created
	case fsnotify.Write:
		return Updated
	case fsnotify.Remove:
		return Deleted
	case fsnotify.Rename:
		return MovedTo
	case fsnotify.Chmod:
		return AttributesModified
	default:
		return Unknown
	}
}
