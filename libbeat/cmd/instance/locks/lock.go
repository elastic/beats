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

package locks

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/gofrs/flock"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	metricproc "github.com/elastic/elastic-agent-system-metrics/metric/system/process"
)

type Locker struct {
	fileLock  *flock.Flock
	logger    *logp.Logger
	beatName  string
	filePath  string
	beatStart time.Time
}

type pidfile struct {
	Pid       int       `json:"pid"`
	WriteTime time.Time `json:"write_time"`
}

var (
	// ErrAlreadyLocked is returned when a lock on the data path is attempted but
	// unsuccessful because another Beat instance already has the lock on the same
	// data path.
	ErrAlreadyLocked = fmt.Errorf("data path already locked by another beat. Please make sure that multiple beats are not sharing the same data path (path.data).")
)

// a little wrapper for the gitpid function to make testing easier.
var pidFetch = os.Getpid

// New returns a new pid-aware file locker
// all logic, including checking for existing locks, is performed lazily
func New(beatInfo beat.Info) *Locker {
	lockfilePath := paths.Resolve(paths.Data, beatInfo.Beat+".lock")
	return &Locker{
		fileLock:  flock.New(lockfilePath),
		logger:    logp.L(),
		beatName:  beatInfo.Beat,
		filePath:  lockfilePath,
		beatStart: beatInfo.StartTime,
	}
}

// Lock attempts to acquire a lock on the data path for the currently-running
// Beat instance. If another Beats instance already has a lock on the same data path
// an ErrAlreadyLocked error is returned.
// This lock is pid-aware, and will attempt to recover if the lockfile is taken by a PID that no longer exists.
func (lock *Locker) Lock() error {
	new := pidfile{Pid: pidFetch(), WriteTime: time.Now()}
	encoded, err := json.Marshal(&new)
	if err != nil {
		return fmt.Errorf("error encoding json for pidfile: %w", err)
	}

	// The combination of O_CREATE and O_EXCL will ensure we return an error if we don't
	// manage to create the file
	fh, openErr := os.OpenFile(lock.filePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(openErr) {
		err = lock.handleFailedLock()
		if err != nil {
			return fmt.Errorf("cannot obtain lockfile: %w", err)
		}
		// At this point, the filepath should be unique to a given pid, and not just a beatname
		// If something fails here, it's probably unrecoverable
		fh, err = os.OpenFile(lock.filePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("cannot re-obtain lockfile %s: %w", lock.filePath, err)
		}
	} else if openErr != nil {
		return fmt.Errorf("error creating lockfile %s: %w", lock.filePath, err)
	}

	// This will be a shared/read lock,
	// if we need to manage a previous lock from another beat instance,
	// we'll need to read from the pid file
	isLocked, err := lock.fileLock.TryRLock()
	if err != nil {
		return fmt.Errorf("unable to lock data path: %w", err)
	}
	// case: lock could not be obtained.
	if !isLocked {
		// if we're here, things are probably unrecoverable, as we've previously checked for a lockfile. Exit.
		return ErrAlreadyLocked
	}

	// write after we have the lock
	_, err = fh.Write(encoded)
	if err != nil {
		return fmt.Errorf("error writing pidfile to %s: %w", lock.filePath, err)
	}

	return nil
}

// Unlock attempts to release the lock on a data path previously acquired via Lock().
func (lock *Locker) Unlock() error {
	err := lock.fileLock.Unlock()
	if err != nil {
		return fmt.Errorf("unable to unlock data path: %w", err)
	}

	err = os.Remove(lock.fileLock.Path())
	if err != nil {
		return fmt.Errorf("unable to unlock data path file %s: %w", lock.fileLock.Path(), err)
	}
	return nil
}

// ******* private helpers

// handleFailedLock will attempt to recover from a failed lock operation in a pid-aware way.
// The point of this is to deal with instances where an improper beat shutdown left us with
// a pre-existing pidfile for a beat process that no longer exists.
// The one argument tells the method if a pre-existing lockfile was already found.
func (lock *Locker) handleFailedLock() error {
	// read in whatever existing lockfile caused us to fail
	gotData, pf, err := lock.readExistingPidfile()
	if err != nil {
		return fmt.Errorf("error reading existing lockfile: %w", err)
	}

	// Case: two beats start up simultaneously, there's a chance we could could "see" the pidfile before the other process writes to it
	// or, the other beat died before it could write the pidfile.
	// Sleep, read again. If we still don't have anything, assume the other PID is dead, continue.
	if !gotData {
		lock.logger.Debugf("Found other pidfile, but no data. Retrying.")
		time.Sleep(time.Millisecond * 500)
		gotData, pf, err = lock.readExistingPidfile()
		if err != nil {
			return fmt.Errorf("error re-reading existing lockfile: %w", err)
		}
		if !gotData {
			lock.logger.Debugf("No PID found in other lockfile, continuing")
			return lock.recoverLockfile()
		}
	}

	// Case: the lockfile is locked, but by us. Probably a coding error,
	// and probably hard to do
	if pf.Pid == os.Getpid() {
		// the lockfile was written before the beat started, meaning we restarted and somehow got the same pid
		// in which case, continue
		if lock.beatStart.Before(pf.WriteTime) {
			return fmt.Errorf("lockfile for beat has been locked twice by the same PID, potential bug.")
		}
		lock.logger.Debugf("Beat has started with the same PID, continuing")
		return lock.recoverLockfile()
	}

	// Check to see if the PID found in the pidfile exists.
	existsState, err := findMatchingPID(pf.Pid)
	// Case: we have a lockfile, but the pid from the pidfile no longer exists
	// this was presumably due to the dirty shutdown.
	// Try to reset the lockfile and continue.
	if errors.Is(err, metricproc.ProcNotExist) {
		lock.logger.Debugf("%s shut down without removing previous lockfile, continuing", lock.beatName)
		return lock.recoverLockfile()
	} else if err != nil {
		return fmt.Errorf("error looking up status for pid %d: %w", pf.Pid, err)
	} else {
		// Case: the PID exists, but it's attached to a zombie process
		// In this case...we should be okay to restart?
		if existsState == metricproc.Zombie {
			lock.logger.Debugf("%s shut down without removing previous lockfile and is currently in a zombie state, continuing", lock.beatName)
			return lock.recoverLockfile()
		}
		// Case: we've gotten a lock file for another process that's already running
		// This is the "base" lockfile case, which is two beats running from the same directory
		return fmt.Errorf("connot start, data directory belongs to process with pid %d", pf.Pid)
	}
}

// recoverLockfile attempts to remove the lockfile and continue running
// This should only be called after we're sure it's safe to ignore a pre-existing locfile
// This will reset the internal lockfile path when it's successful.
func (lock *Locker) recoverLockfile() error {
	// File remove or may not work, depending on os-specific details with lockfiles
	err := os.Remove(lock.fileLock.Path())
	if err != nil {
		rname := rand.New(rand.NewSource(time.Now().UnixNano()))
		lockfilePath := paths.Resolve(paths.Data, fmt.Sprintf("%s_%d.lock", lock.beatName, rname.Int()))
		// Per @leehinman, on windows the lock release can be dependent on OS resources. Retry.
		if runtime.GOOS == "windows" {
			time.Sleep(time.Second)
			err = os.Remove(lock.fileLock.Path())
			if err != nil {
				lock.logger.Warnf("tried twice to remove lockfile %s on windows, continuing on with new lockfile name %s",
					lockfilePath, err)
				lock.filePath = lockfilePath
			}
		} else {
			lock.logger.Warnf("failed to reset lockfile, cannot remove %s, continuing on with new lockfile name %s",
				lock.fileLock.Path(), lockfilePath)
			lock.filePath = lockfilePath
		}

	}
	lock.fileLock = flock.New(lock.filePath)
	return nil
}

// readExistingPidfile will read the contents of an existing pidfile
// Will return false,pidfile{},nil if no data is found in the lockfile
func (lock *Locker) readExistingPidfile() (bool, pidfile, error) {
	rawPidfile, err := os.ReadFile(lock.filePath)
	if err != nil {
		return false, pidfile{}, fmt.Errorf("error reading pidfile from path %s", lock.filePath)
	}
	if len(rawPidfile) == 0 {
		return false, pidfile{}, nil
	}
	foundPidFile := pidfile{}
	err = json.Unmarshal(rawPidfile, &foundPidFile)
	if err != nil {
		return false, pidfile{}, fmt.Errorf("error reading JSON from pid file %s: %w", lock.filePath, err)
	}
	return true, foundPidFile, nil
}
