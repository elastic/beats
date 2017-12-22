// +build darwin

package file_integrity

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsevents"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
)

var underTest = false

func init() {
	underTest = flag.Lookup("test.v") != nil
}

type fsreader struct {
	stream      *fsevents.EventStream
	config      Config
	eventC      chan Event
	watchedDirs []os.FileInfo
	log         *logp.Logger
}

var flagToAction = map[fsevents.EventFlags]Action{
	fsevents.MustScanSubDirs: None,
	fsevents.UserDropped:     None,
	fsevents.KernelDropped:   None,
	fsevents.EventIDsWrapped: None,
	fsevents.HistoryDone:     None,
	// RootChanged signals that a directory along a watched path was moved
	// or deleted, or the path was created. Mapping it to `Moved` which
	// makes sense in both cases
	fsevents.RootChanged:       Moved,
	fsevents.Mount:             None,
	fsevents.Unmount:           None,
	fsevents.ItemCreated:       Created,
	fsevents.ItemRemoved:       Deleted,
	fsevents.ItemInodeMetaMod:  AttributesModified,
	fsevents.ItemRenamed:       Moved,
	fsevents.ItemModified:      Updated,
	fsevents.ItemFinderInfoMod: AttributesModified,
	fsevents.ItemChangeOwner:   AttributesModified,
	fsevents.ItemXattrMod:      AttributesModified,
	fsevents.ItemIsFile:        None,
	fsevents.ItemIsDir:         None,
	fsevents.ItemIsSymlink:     None,
}

var flagNames = map[fsevents.EventFlags]string{
	fsevents.MustScanSubDirs:   "MustScanSubDirs",
	fsevents.UserDropped:       "UserDropped",
	fsevents.KernelDropped:     "KernelDropped",
	fsevents.EventIDsWrapped:   "EventIDsWrapped",
	fsevents.HistoryDone:       "HistoryDone",
	fsevents.RootChanged:       "RootChanged",
	fsevents.Mount:             "Mount",
	fsevents.Unmount:           "Unmount",
	fsevents.ItemCreated:       "ItemCreated",
	fsevents.ItemRemoved:       "ItemRemoved",
	fsevents.ItemInodeMetaMod:  "ItemInodeMetaMod",
	fsevents.ItemRenamed:       "ItemRenamed",
	fsevents.ItemModified:      "ItemModified",
	fsevents.ItemFinderInfoMod: "ItemFinderInfoMod",
	fsevents.ItemChangeOwner:   "ItemChangeOwner",
	fsevents.ItemXattrMod:      "ItemXattrMod",
	fsevents.ItemIsFile:        "ItemIsFile",
	fsevents.ItemIsDir:         "ItemIsDir",
	fsevents.ItemIsSymlink:     "ItemIsSymlink",
}

// NewEventReader creates a new EventProducer backed by FSEvents macOS facility.
func NewEventReader(c Config) (EventProducer, error) {
	stream := &fsevents.EventStream{
		Paths: c.Paths,
		// NoDefer: Ignore Latency field and send events as fast as possible.
		//          Useful as latency has one second granularity.
		//
		// WatchRoot: Will send a notification when some element changes along
		// 			the path being watched (dir moved or deleted).
		//
		// FileEvents: Get events for files not just directories
		Flags: fsevents.NoDefer | fsevents.WatchRoot | fsevents.FileEvents,
	}

	// IgnoreSelf: Avoid infinite looping when auditbeat writes to a
	//			   watched directory. If specified tests won't work.
	if !underTest {
		stream.Flags |= fsevents.IgnoreSelf
	}

	log := logp.NewLogger(moduleName)
	var dirs []os.FileInfo
	if !c.Recursive {
		for _, path := range c.Paths {
			if info, err := getFileInfo(path); err == nil {
				dirs = append(dirs, info)
			} else {
				log.Warnw("Failed to get file info", "file_path", path, "error", err)
			}
		}
	}
	return &fsreader{
		stream:      stream,
		config:      c,
		eventC:      make(chan Event, 1),
		watchedDirs: dirs,
		log:         log,
	}, nil
}

func (r *fsreader) Start(done <-chan struct{}) (<-chan Event, error) {
	r.stream.Start()
	go r.consumeEvents(done)
	r.log.Infow("Started FSEvents watcher",
		"file_path", r.config.Paths,
		"recursive", r.config.Recursive)
	return r.eventC, nil
}

func (r *fsreader) consumeEvents(done <-chan struct{}) {
	defer close(r.eventC)
	defer r.stream.Stop()

	for {
		select {
		case <-done:
			r.log.Debug("FSEvents reader terminated")
			return
		case events := <-r.stream.Events:
			for _, event := range events {
				if !r.isWatched(event.Path) || r.config.IsExcludedPath(event.Path) {
					continue
				}
				r.log.Debugw("Received FSEvents event",
					"file_path", event.Path,
					"event_id", event.ID,
					"event_flags", flagsToString(event.Flags))

				start := time.Now()
				e := NewEvent(event.Path, flagsToAction(event.Flags), SourceFSNotify,
					r.config.MaxFileSizeBytes, r.config.HashTypes)

				e.rtt = time.Since(start)
				r.eventC <- e
			}
		}
	}
}

func flagsToAction(flags fsevents.EventFlags) Action {
	action := None
	for flag, act := range flagToAction {
		if (flags & flag) != 0 {
			action |= act
		}
	}
	return action
}

func flagsToString(flags fsevents.EventFlags) string {
	var list []string
	for key, name := range flagNames {
		if 0 != flags&key {
			list = append(list, name)
		}
	}
	return strings.Join(list, "|")
}

func getFileInfo(path string) (os.FileInfo, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	info, err := os.Lstat(path)
	return info, errors.Wrap(err, "failed to stat")
}

func (r *fsreader) isWatched(path string) bool {
	if r.config.Recursive {
		return true
	}
	dir := filepath.Dir(path)
	info, err := getFileInfo(dir)
	if err != nil {
		r.log.Warnw("failed to get file info", "file_path", dir, "error", err)
		return false
	}
	for _, dir := range r.watchedDirs {
		if os.SameFile(info, dir) {
			return true
		}
	}
	return false
}
