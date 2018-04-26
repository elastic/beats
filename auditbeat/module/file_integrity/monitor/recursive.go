package monitor

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

type recursiveWatcher struct {
	inner  *fsnotify.Watcher
	tree   FileTree
	eventC chan fsnotify.Event
	done   chan bool
}

func newRecursiveWatcher(inner *fsnotify.Watcher) *recursiveWatcher {
	return &recursiveWatcher{
		inner:  inner,
		tree:   FileTree{},
		eventC: make(chan fsnotify.Event, 1),
	}
}

func (watcher *recursiveWatcher) Start() error {
	watcher.done = make(chan bool, 1)
	go watcher.forwardEvents()
	return nil
}

func (watcher *recursiveWatcher) Add(path string) error {
	return watcher.addRecursive(path)
}

func (watcher *recursiveWatcher) Close() error {
	if watcher.done != nil {
		// has been Started(), goroutine takes care of cleanup
		close(watcher.done)
		return nil
	}
	// not started
	return watcher.close()
}

func (watcher *recursiveWatcher) EventChannel() <-chan fsnotify.Event {
	return watcher.eventC
}

func (watcher *recursiveWatcher) ErrorChannel() <-chan error {
	return watcher.inner.Errors
}

func (watcher *recursiveWatcher) addRecursive(path string) error {
	var errs multierror.Errors
	err := filepath.Walk(path, func(path string, info os.FileInfo, fnErr error) error {
		if fnErr != nil {
			errs = append(errs, errors.Wrapf(fnErr, "error walking path '%s'", path))
			// If FileInfo is not nil, the directory entry can be processed
			// even if there was some error
			if info == nil {
				return nil
			}
		}
		var err error
		if info.IsDir() {
			if err = watcher.tree.AddDir(path); err == nil {
				if err = watcher.inner.Add(path); err != nil {
					errs = append(errs, errors.Wrapf(err, "failed adding watcher to '%s'", path))
					return nil
				}
			}
		} else {
			err = watcher.tree.AddFile(path)
		}
		return err
	})
	if err != nil {
		errs = append(errs, errors.Wrapf(err, "failed to walk path '%s'", path))
	}
	return errs.Err()
}

func (watcher *recursiveWatcher) close() error {
	close(watcher.eventC)
	return watcher.inner.Close()
}

func (watcher *recursiveWatcher) forwardEvents() error {
	defer watcher.close()

	for {
		select {
		case <-watcher.done:
			return nil

		case event, ok := <-watcher.inner.Events:
			if !ok {
				return nil
			}
			if event.Name == "" {
				continue
			}
			switch event.Op {
			case fsnotify.Create:
				if err := watcher.addRecursive(event.Name); err != nil {
					watcher.inner.Errors <- errors.Wrapf(err, "unable to recurse path '%s'", event.Name)
				}
				watcher.tree.Visit(event.Name, PreOrder, func(path string, _ bool) error {
					watcher.eventC <- fsnotify.Event{
						Name: path,
						Op:   event.Op,
					}
					return nil
				})

			case fsnotify.Remove:
				watcher.tree.Visit(event.Name, PostOrder, func(path string, _ bool) error {
					watcher.eventC <- fsnotify.Event{
						Name: path,
						Op:   event.Op,
					}
					return nil
				})
				watcher.tree.Remove(event.Name)

			// Handling rename (move) as a special case to give this recursion
			// the same semantics as macOS FSEvents:
			// - Removal of a dir notifies removal for all files inside it
			// - Moving a dir away sends only one notification for this dir
			case fsnotify.Rename:
				watcher.tree.Remove(event.Name)
				fallthrough

			default:
				watcher.eventC <- event
			}
		}
	}
}
