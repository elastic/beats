// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package monitor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/joeshaw/multierror"

	"github.com/elastic/elastic-agent-libs/logp"
)

type recursiveWatcher struct {
	inner   *fsnotify.Watcher
	tree    FileTree
	eventC  chan fsnotify.Event
	done    chan bool
	addC    chan string
	addErrC chan error
	log     *logp.Logger

	isExcludedPath func(path string) bool
}

func newRecursiveWatcher(inner *fsnotify.Watcher, IsExcludedPath func(path string) bool) *recursiveWatcher {
	return &recursiveWatcher{
		inner:          inner,
		tree:           FileTree{},
		eventC:         make(chan fsnotify.Event, 1),
		addC:           make(chan string),
		addErrC:        make(chan error),
		log:            logp.NewLogger(moduleName),
		isExcludedPath: IsExcludedPath,
	}
}

func (watcher *recursiveWatcher) Start() error {
	watcher.done = make(chan bool, 1)
	go watcher.forwardEvents()
	return nil
}

func (watcher *recursiveWatcher) Add(path string) error {
	if watcher.done != nil {
		watcher.addC <- path
		return <-watcher.addErrC
	}
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
	if watcher.isExcludedPath(path) {
		return nil
	}

	var errs multierror.Errors
	err := filepath.Walk(path, func(path string, info os.FileInfo, fnErr error) error {
		if watcher.isExcludedPath(path) {
			return nil
		}

		if fnErr != nil {
			errs = append(errs, fmt.Errorf("error walking path '%s': %w", path, fnErr))
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
					errs = append(errs, fmt.Errorf("failed adding watcher to '%s': %w", path, err))
					return nil
				}
			}
		} else {
			err = watcher.tree.AddFile(path)
		}
		return err
	})
	watcher.log.Debugw("Added recursive watch", "path", path)

	if err != nil {
		errs = append(errs, fmt.Errorf("failed to walk path '%s': %w", path, err))
	}
	return errs.Err()
}

func (watcher *recursiveWatcher) close() error {
	close(watcher.eventC)
	return watcher.inner.Close()
}

func (watcher *recursiveWatcher) deliver(ev fsnotify.Event) {
	for {
		select {
		case <-watcher.done:
			return

		case path := <-watcher.addC:
			watcher.addErrC <- watcher.addRecursive(path)

		case watcher.eventC <- ev:
			return
		}
	}
}

func (watcher *recursiveWatcher) forwardEvents() error {
	defer watcher.close()

	for {
		select {
		case <-watcher.done:
			return nil

		case path := <-watcher.addC:
			watcher.addErrC <- watcher.addRecursive(path)

		case event, ok := <-watcher.inner.Events:
			if !ok {
				return nil
			}
			if event.Name == "" {
				continue
			}
			switch event.Op {
			case fsnotify.Create:
				err := watcher.addRecursive(event.Name)
				if err != nil {
					watcher.inner.Errors <- fmt.Errorf("failed to add created path '%s': %w", event.Name, err)
				}
				err = watcher.tree.Visit(event.Name, PreOrder, func(path string, _ bool) error {
					watcher.deliver(fsnotify.Event{
						Name: path,
						Op:   event.Op,
					})
					return nil
				})
				if err != nil {
					watcher.inner.Errors <- fmt.Errorf("failed to visit created path '%s': %w", event.Name, err)
				}

			case fsnotify.Remove:
				err := watcher.tree.Visit(event.Name, PostOrder, func(path string, _ bool) error {
					watcher.deliver(fsnotify.Event{
						Name: path,
						Op:   event.Op,
					})
					return nil
				})
				if err != nil {
					watcher.inner.Errors <- fmt.Errorf("failed to visit removed path '%s': %w", event.Name, err)
				}

				err = watcher.tree.Remove(event.Name)
				if err != nil {
					watcher.inner.Errors <- fmt.Errorf("failed to visit removed path '%s': %w", event.Name, err)
				}

			// Handling rename (move) as a special case to give this recursion
			// the same semantics as macOS FSEvents:
			// - Removal of a dir notifies removal for all files inside it
			// - Moving a dir away sends only one notification for this dir
			case fsnotify.Rename:
				err := watcher.tree.Remove(event.Name)
				if err != nil {
					watcher.inner.Errors <- fmt.Errorf("failed to remove path '%s': %w", event.Name, err)
				}
				fallthrough

			default:
				watcher.deliver(event)
			}
		}
	}
}
