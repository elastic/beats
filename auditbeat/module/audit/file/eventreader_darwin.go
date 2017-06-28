package file

import (
	"time"

	"github.com/fsnotify/fsevents"
)

func NewEventReader(c Config) (EventReader, error) {
	var paths []string
	for _, filePaths := range c.Paths {
		paths = append(paths, filePaths...)
	}

	es := &fsevents.EventStream{
		Paths:   paths,
		Latency: 500 * time.Millisecond,
		Flags:   fsevents.FileEvents | fsevents.WatchRoot,
	}

	return &reader{es, c.MaxFileSize, make(chan Event, 1)}, nil
}

type reader struct {
	stream      *fsevents.EventStream
	maxFileSize int64
	out         chan Event
}

func (r *reader) Start(done <-chan struct{}) (<-chan Event, error) {
	r.stream.Start()
	ec := r.stream.Events

	go func() {
		defer r.stream.Stop()

		for {
			select {
			case <-done:
				return
			case events := <-ec:
				for _, e := range events {
					r.out <- convertToFileEvent(e, r.maxFileSize)
				}
			}
		}
	}()

	return r.out, nil
}

func convertToFileEvent(e fsevents.Event, maxFileSize int64) Event {
	event := Event{
		Timestamp: time.Now(),
		Path:      e.Path,
		Action:    flagsToAction(e.Flags).String(),
	}

	addFileAttributes(&event, maxFileSize)

	return event
}

func flagsToAction(f fsevents.EventFlags) Action {
	switch {
	case f&fsevents.ItemCreated > 0:
		return Created
	case f&fsevents.ItemModified > 0:
		return Updated
	case f&fsevents.ItemRemoved > 0:
		return Deleted
	case f&fsevents.ItemRenamed > 0:
		return MovedTo
	case f&fsevents.ItemChangeOwner > 0,
		f&fsevents.ItemInodeMetaMod > 0,
		f&fsevents.ItemXattrMod > 0,
		f&fsevents.ItemFinderInfoMod > 0:
		return AttributesModified
	case f&fsevents.Unmount > 0:
		return Unmounted
	case f&fsevents.MustScanSubDirs > 0:
		return CollisionWithin
	default:
		return Unknown
	}
}
