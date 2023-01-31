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
	"fmt"
	"time"

	"github.com/gofrs/flock"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// Locker is a retrying file locker
type Locker struct {
	fileLock   *flock.Flock
	retryCount int
	retrySleep time.Duration
	logger     *logp.Logger
}

var (
	// ErrAlreadyLocked is returned when a lock on the data path is attempted but
	// unsuccessful because another Beat instance already has the lock on the same
	// data path.
	ErrAlreadyLocked = fmt.Errorf("data path already locked by another beat. Please make sure that multiple beats are not sharing the same data path (path.data)")
)

// New returns a new file locker
func New(beatInfo beat.Info) *Locker {
	return NewWithRetry(beatInfo, 4, time.Millisecond*400)
}

// NewWithRetry returns a new file locker with the given settings
func NewWithRetry(beatInfo beat.Info, retryCount int, retrySleep time.Duration) *Locker {
	lockfilePath := paths.Resolve(paths.Data, beatInfo.Beat+".lock")
	return &Locker{
		fileLock:   flock.New(lockfilePath),
		retryCount: retryCount,
		retrySleep: retrySleep,
		logger:     logp.L(),
	}
}

// Lock attempts to acquire a lock on the data path for the currently-running
// Beat instance. If another Beats instance already has a lock on the same data path
// an ErrAlreadyLocked error is returned.
func (lock *Locker) Lock() error {
	for i := 0; i < lock.retryCount; i++ {
		// note that TryLock doesn't set an os.O_EXCL flag,
		// which means that we could be locking a file we didn't create.
		// This makes it easy to recover from a failed shutdown or panic,
		// as the OS will clean up the lock and we'll re-lock the same file.
		// However, can create odd races if you're not careful, since you don't know if you're locking "your" file.
		gotLock, err := lock.fileLock.TryLock()
		if err != nil {
			return fmt.Errorf("unable to try a lock of the data path: %w", err)
		}
		if gotLock {
			return nil
		}
		lock.logger.Debugf("Could not obtain lock for file %s, retrying %d times", lock.fileLock.Path(), (lock.retryCount - i))
		time.Sleep(lock.retrySleep)
	}
<<<<<<< HEAD

	// a Process can't write to its own locked file on all platforms, write first
	_, err = fh.Write(encoded)
	if err != nil {
		return fmt.Errorf("error writing pidfile to %s: %w", lock.filePath, err)
	}

	// Exclusive lock
	isLocked, err := lock.fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("unable to lock data path: %w", err)
	}
	// case: lock could not be obtained.
	if !isLocked {
		// if we're here, things are probably unrecoverable, as we've previously checked for a lockfile. Exit.
		return fmt.Errorf("%s: %w", lock.filePath, ErrAlreadyLocked)
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

// handleFailedCreate will attempt to recover from a failed lock operation in a pid-aware way.
// The point of this is to deal with instances where an improper beat shutdown left us with
// a pre-existing pidfile for a beat process that no longer exists.
func (lock *Locker) handleFailedCreate() error {
	// First, try to lock the file as a check to see what state we're in.
	// If there's a pre-existing lock that's in effect, we probably can't recover
	// Not all OSes will fail on this.
	_, err := lock.fileLock.TryLock()
	// Case: the file already locked, and in use by another process.
	if err != nil {
		if runtime.GOOS == "windows" {
			// on windows, locks from dead PIDs will be auto-released, but it might take the OS a while.
			// However, the time it takes for the operating system to unlock these locks depends upon available system resources.
			time.Sleep(time.Second)
			_, err := lock.fileLock.TryLock()
			if err != nil {
				return fmt.Errorf("The lockfile %s is locked after a retry, another beat is probably running", lock.fileLock)
			}
		} else {
			return fmt.Errorf("The lockfile %s is already locked by another beat", lock.fileLock)
		}
	}

	// if we're here, we've locked the file
	// unlock so we can continue
	err = lock.fileLock.Unlock()
	if err != nil {
		return fmt.Errorf("error unlocking a previously found file %s after a temporary lock", lock.filePath)
	}

	// read in whatever existing lockfile caused us to fail
	pf, err := lock.readExistingPidfile()
	// Case: two beats start up simultaneously, there's a chance we could "see" the pidfile before the other process writes to it
	// or, the other beat died before it could write the pidfile.
	// Sleep, read again. If we still don't have anything, assume the other PID is dead, continue.
	if errors.Is(err, ErrLockfileEmpty) {
		lock.logger.Debugf("Found other pidfile, but no data. Retrying.")
		time.Sleep(time.Millisecond * 500)
		pf, err = lock.readExistingPidfile()
		if errors.Is(err, ErrLockfileEmpty) {
			lock.logger.Debugf("No PID found in other lockfile, continuing")
			return lock.recoverLockfile()
		} else if err != nil {
			return fmt.Errorf("error re-reading existing lockfile: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error reading existing lockfile: %w", err)
	}

	// Case: the lockfile is locked, but by us. Probably a coding error,
	// and probably hard to do
	if pf.PID == os.Getpid() {
		// the lockfile was written before the beat started, meaning we restarted and somehow got the same pid
		// in which case, continue
		if lock.beatStart.Before(pf.WriteTime) {
			return fmt.Errorf("lockfile for beat has been locked twice by the same PID, potential bug.")
		}
		lock.logger.Debugf("Beat has started with the same PID, continuing")
		return lock.recoverLockfile()
	}

	// Check to see if the PID found in the pidfile exists.
	existsState, err := findMatchingPID(pf.PID)
	// Case: we have a lockfile, but the pid from the pidfile no longer exists
	// this was presumably due to the dirty shutdown.
	// Try to reset the lockfile and continue.
	if errors.Is(err, metricproc.ProcNotExist) {
		lock.logger.Debugf("%s shut down without removing previous lockfile, continuing", lock.beatName)
		return lock.recoverLockfile()
	} else if err != nil {
		return fmt.Errorf("error looking up status for pid %d: %w", pf.PID, err)
	} else {
		// Case: the PID exists, but it's attached to a zombie process
		// In this case...we should be okay to restart?
		if existsState == metricproc.Zombie {
			lock.logger.Debugf("%s shut down without removing previous lockfile and is currently in a zombie state, continuing", lock.beatName)
			return lock.recoverLockfile()
		}
		// Case: we've gotten a lock file for another process that's already running
		// This is the "base" lockfile case, which is two beats running from the same directory
		// This is where we'll catch this particular case on Linux, due to Linux's advisory-style locks.
		return fmt.Errorf("connot start, data directory belongs to process with pid %d", pf.PID)
	}
}

// recoverLockfile attempts to remove the lockfile and continue running
// This should only be called after we're sure it's safe to ignore a pre-existing lockfile
// This will reset the internal lockfile handler when it's successful.
func (lock *Locker) recoverLockfile() error {
	// File remove may or not work, depending on os-specific details with lockfiles
	err := os.Remove(lock.fileLock.Path())
	if err != nil {
		if runtime.GOOS == "windows" {
			// retry on windows, the OS can take time to clean up
			time.Sleep(time.Second)
			err = os.Remove(lock.fileLock.Path())
			if err != nil {
				return fmt.Errorf("tried twice to remove lockfile %s on windows: %w",
					lock.fileLock.Path(), err)
			}
		} else {
			return fmt.Errorf("lockfile %s cannot be removed: %w", lock.fileLock.Path(), err)
		}

	}
	lock.fileLock = flock.New(lock.filePath)
	return nil
}

// readExistingPidfile will read the contents of an existing pidfile
// Will return ErrLockfileEmpty if no data is found in the lockfile
func (lock *Locker) readExistingPidfile() (pidfile, error) {
	rawPidfile, err := os.ReadFile(lock.filePath)
	if err != nil {
		return pidfile{}, fmt.Errorf("error reading pidfile from path %s", lock.filePath)
	}
	if len(rawPidfile) == 0 {
		return pidfile{}, ErrLockfileEmpty
	}
	foundPidFile := pidfile{}
	err = json.Unmarshal(rawPidfile, &foundPidFile)
	if err != nil {
		return pidfile{}, fmt.Errorf("error reading JSON from pid file %s: %w", lock.filePath, err)
	}
	return foundPidFile, nil
=======
	return fmt.Errorf("%s: %w", lock.fileLock.Path(), ErrAlreadyLocked)
>>>>>>> 21b6128c95 (Refactor beats lockfile to use timeout, retry (#34194))
}
