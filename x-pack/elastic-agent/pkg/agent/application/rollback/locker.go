// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

const watcherLockFile = "watcher.lock"

// ErrAlreadyLocked is returned when lock is already taken.
var ErrAlreadyLocked = errors.New("watcher already locked")

// Locker locks the agent.lock file inside the provided directory.
type Locker struct {
	lock *flock.Flock
}

// NewLocker creates an Locker that locks the agent.lock file inside the provided directory.
func NewLocker(dir string) *Locker {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, 0755)
	}
	return &Locker{
		lock: flock.New(filepath.Join(dir, watcherLockFile)),
	}
}

// TryLock tries to grab the lock file and returns error if it cannot.
func (a *Locker) TryLock() error {
	locked, err := a.lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return ErrAlreadyLocked
	}
	return nil
}

// Unlock releases the lock file.
func (a *Locker) Unlock() error {
	return a.lock.Unlock()
}
