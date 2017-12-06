// +build linux freebsd openbsd netbsd windows

package file

import (
	"errors"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/elastic/beats/libbeat/logp"
)

type reader struct {
	watcher *fsnotify.Watcher
	config  Config
	eventC  chan Event
}

// NewEventReader creates a new EventProducer backed by fsnotify.
func NewEventReader(c Config) (EventProducer, error) {
	if c.Recursive {
		return nil, errors.New("recursive file auditing not supported in this platform (see file.recursive)")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &reader{
		watcher: watcher,
		config:  c,
		eventC:  make(chan Event, 1),
	}, nil
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

	go r.consumeEvents()
	logp.Info("%v started fsnotify watcher", logPrefix)
	return r.eventC, nil
}

func (r *reader) consumeEvents() {
	defer close(r.eventC)
	defer r.watcher.Close()

	for {
		select {
		case event := <-r.watcher.Events:
			if event.Name == "" {
				continue
			}
			debugf("Received fsnotify event: path=%v action=%v",
				event.Name, event.Op.String())

			start := time.Now()
			e := NewEvent(event.Name, opToAction(event.Op), SourceFSNotify,
				r.config.MaxFileSizeBytes, r.config.HashTypes)
			e.rtt = time.Since(start)

			r.eventC <- e
		case err := <-r.watcher.Errors:
			logp.Warn("%v fsnotify watcher error: %v", logPrefix, err)
		}
	}
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
		return 0
	}
}
