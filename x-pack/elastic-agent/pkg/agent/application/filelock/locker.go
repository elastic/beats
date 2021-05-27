// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filelock

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// ErrAppAlreadyRunning error returned when another elastic-agent is already holding the lock.
var ErrAppAlreadyRunning = fmt.Errorf("another elastic-agent is already running")

// AppLocker locks the agent.lock file inside the provided directory.
type AppLocker struct {
	lock *flock.Flock
}

// NewAppLocker creates an AppLocker that locks the agent.lock file inside the provided directory.
func NewAppLocker(dir, lockFileName string) *AppLocker {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, 0755)
	}
	return &AppLocker{
		lock: flock.New(filepath.Join(dir, lockFileName)),
	}
}

// TryLock tries to grab the lock file and returns error if it cannot.
func (a *AppLocker) TryLock() error {
	locked, err := a.lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return ErrAppAlreadyRunning
	}
	return nil
}

// Unlock releases the lock file.
func (a *AppLocker) Unlock() error {
	return a.lock.Unlock()
}
