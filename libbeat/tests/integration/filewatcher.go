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

//This file was contributed to by generative AI

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher       *fsnotify.Watcher
	target        string
	watchDir      bool
	mu            sync.Mutex
	stopChan      chan struct{}
	eventCallback func(event fsnotify.Event)
	t             testing.TB
}

// NewFileWatcher creates a new FileWatcher instance, if `target` is a file
// it will watch only that file, if it is a directory, all files in the
// directory will be watched.
func NewFileWatcher(t testing.TB, target string) *FileWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %s", err)
	}

	return &FileWatcher{
		watcher:  watcher,
		target:   target,
		stopChan: make(chan struct{}),
		t:        t,
	}
}

// Start begins watching the file's directory for changes.
func (f *FileWatcher) Start() {
	st, err := os.Stat(f.target)
	if err != nil {
		f.t.Fatalf("cannot stat file: %s", err)
	}

	var dir string
	if st.IsDir() {
		f.watchDir = true
		dir = f.target
	} else {
		dir = filepath.Dir(f.target)
	}

	if err := f.watcher.Add(dir); err != nil {
		f.t.Fatalf("failed to add directory to watcher: %s", err)
	}

	go f.watch()
}

// Stop stops the file-watching process.
func (f *FileWatcher) Stop() {
	close(f.stopChan)
	f.watcher.Close()
}

// SetEventCallback sets a callback function for file events.
// To check the event that happened, use event.Has.
func (f *FileWatcher) SetEventCallback(callback func(event fsnotify.Event)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eventCallback = callback
}

// watch processes file events and errors.
func (f *FileWatcher) watch() {
	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}

			if event.Name == f.target || f.watchDir {
				f.mu.Lock()
				if f.eventCallback != nil {
					f.eventCallback(event)
				}
				f.mu.Unlock()
			}

		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}

			f.t.Errorf("FileWatcher failed: %s", err)
			return

		case <-f.stopChan:
			return
		}
	}
}
