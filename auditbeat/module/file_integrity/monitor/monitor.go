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
	"github.com/fsnotify/fsnotify"
)

const (
	moduleName = "file_integrity"
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
