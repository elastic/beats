// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

const lockFileName = "agent.lock"

var ErrAppAlreadyRunning = fmt.Errorf("another elastic-agent is already running")

type AppLocker struct {
	lock *flock.Flock
}

func NewAppLocker(dir string) *AppLocker {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, 0755)
	}
	return &AppLocker{
		lock: flock.New(filepath.Join(dir, lockFileName)),
	}
}

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

func (a *AppLocker) Unlock() error {
	return a.lock.Unlock()
}
