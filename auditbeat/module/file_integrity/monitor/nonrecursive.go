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

import "github.com/fsnotify/fsnotify"

type nonRecursiveWatcher fsnotify.Watcher

func (*nonRecursiveWatcher) Start() error {
	return nil
}

func (watcher *nonRecursiveWatcher) Add(path string) error {
	return (*fsnotify.Watcher)(watcher).Add(path)
}

func (watcher *nonRecursiveWatcher) Close() error {
	return (*fsnotify.Watcher)(watcher).Close()
}

func (watcher *nonRecursiveWatcher) EventChannel() <-chan fsnotify.Event {
	return (*fsnotify.Watcher)(watcher).Events
}

func (watcher *nonRecursiveWatcher) ErrorChannel() <-chan error {
	return (*fsnotify.Watcher)(watcher).Errors
}
