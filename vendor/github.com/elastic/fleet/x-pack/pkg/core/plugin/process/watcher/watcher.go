// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package watcher

import (
	"errors"
	"os"
	"sync"

	"github.com/urso/ecslog"
)

// CloseReason is the reason of notifying subscriber.
// Can be either that process was crashed or stopped (unwatched)
type CloseReason int

const (
	// ProcessUnwatched means process was unwatched and will no longer be watched
	ProcessUnwatched CloseReason = iota
	// ProcessCrashed means process crashed
	ProcessCrashed
	// ProcessClosed means process exited with success status code
	ProcessClosed
)

var (
	// ErrProcessAlreadyExited is an error returned when watcher tries to watch
	// already exited process
	ErrProcessAlreadyExited = errors.New("process already exited")
)

// Watcher watches processes for crashes
type Watcher struct {
	sync.Mutex
	logger    *ecslog.Logger
	watchList map[int]chan CloseReason
}

// NewProcessWatcher creates a new instance of a process watcher
func NewProcessWatcher(l *ecslog.Logger) *Watcher {
	w := Watcher{
		watchList: make(map[int]chan CloseReason),
		logger:    l,
	}

	return &w
}

// Watch starts watching over a process with a specified proc
// Returns a channel through which it notifies about a process status
// If process is not found it returns os.ErrNotFound
func (w *Watcher) Watch(proc *os.Process) (<-chan CloseReason, error) {
	if proc == nil {
		return nil, ErrProcessAlreadyExited
	}
	// there will be only one result, we don't block write and goroutine exit
	closeChan := make(chan CloseReason, 1)

	w.Lock()
	w.watchList[proc.Pid] = closeChan
	w.Unlock()

	go func() {
		pid := proc.Pid
		state, err := proc.Wait()
		if err != nil {
			// process is not a child - some OSs requires process to be child
			w.externalProcess(proc)
		}

		w.Lock()
		defer w.Unlock()
		ch, found := w.watchList[pid]
		if !found {
			w.logger.Errorf("process with PID '%d' not found in watcher for closing", pid)
			return
		}

		if state != nil && state.Success() {
			ch <- ProcessClosed
		} else {
			ch <- ProcessCrashed
		}
		close(ch)
		delete(w.watchList, pid)

	}()

	return closeChan, nil
}

// UnWatch disables watching over a process.
// All subscribers will be notified by ProcessUnwatched.
func (w *Watcher) UnWatch(pid int) error {
	w.Lock()
	defer w.Unlock()

	ch, found := w.watchList[pid]
	if !found {
		return nil
	}

	ch <- ProcessUnwatched
	close(ch)
	delete(w.watchList, pid)

	return nil
}
