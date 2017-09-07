// +build linux freebsd openbsd netbsd windows darwin

package file

import (
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/elastic/beats/libbeat/logp"
)

// NewEventReader creates a new EventReader backed by fsnotify.
func NewEventReader(c Config) (EventReader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &reader{watcher, c, make(chan Event, 1), make(chan error, 1)}, nil
}

type reader struct {
	watcher *fsnotify.Watcher
	config  Config
	outC    chan Event
	errC    chan error
}

func (r *reader) Start(done <-chan struct{}) (<-chan Event, error) {
	for _, p := range r.config.Paths {
		if err := r.watcher.Add(p); err != nil {
			if err == syscall.EMFILE {
				logp.Warn("%v Failed to watch %v: %v (check the max number of "+
					"open files allowed with 'ulimit -a')", logPrefix, p, err)
			} else {
				logp.Warn("%v Failed to watch %v: %v", logPrefix, p, err)
			}
		}
	}

	go func() {
		defer close(r.outC)
		defer r.watcher.Close()

		for {
			select {
			case event := <-r.watcher.Events:
				if event.Name == "" {
					continue
				}
				r.outC <- convertToFileEvent(event, r.config.MaxFileSizeBytes, r.config.HashTypes)
			case err := <-r.watcher.Errors:
				r.errC <- err
			}
		}
	}()

	return r.outC, nil
}

func convertToFileEvent(e fsnotify.Event, maxFileSize uint64, hashTypes []string) Event {
	event := Event{
		Timestamp: time.Now().UTC(),
		Path:      e.Name,
		Action:    opToAction(e.Op).String(),
	}

	var err error
	event.Info, err = Stat(event.Path)
	if err != nil {
		event.errors = append(event.errors, err)
	}
	if event.Info == nil {
		return event
	}

	switch event.Info.Type {
	case "file":
		if uint64(event.Info.Size) <= maxFileSize {
			hashes, err := hashFile(event.Path, hashTypes...)
			if err != nil {
				event.errors = append(event.errors, err)
			} else {
				event.Hashes = hashes
			}
		}
	case "symlink":
		event.TargetPath, _ = filepath.EvalSymlinks(event.Path)
	}

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
		return Moved
	case fsnotify.Chmod:
		return AttributesModified
	default:
		return Unknown
	}
}
