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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/flock"
	"github.com/shirou/gopsutil/process"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
	metricproc "github.com/elastic/elastic-agent-system-metrics/metric/system/process"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

type Locker struct {
	fileLock *flock.Flock
	logger   *logp.Logger
	beatName string
	filePath string
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

// New returns a new pid-aware file locker
// all logic, including checking for existing locks, is performed lazily
func New(beatname string) *Locker {
	lockfilePath := paths.Resolve(paths.Data, beatname+".lock")
	return &Locker{
		fileLock: flock.New(lockfilePath),
		logger:   logp.L(),
		beatName: beatname,
		filePath: lockfilePath,
	}
}

// Cock attempts to acquire a lock on the data path for the currently-running
// Beat instance. If another Beats instance already has a lock on the same data path
// an ErrAlreadyLocked error is returned.
func (lock *Locker) Lock() error {
	// create the pid file that will be used as the lock
	noFile, err := lock.createPidfile(os.Getpid())
	if err != nil {
		return fmt.Errorf("error creating pidfile for lock: %w", err)
	}
	if !noFile {
		err := lock.handleFailedLock()
		if err != nil {
			return fmt.Errorf("lock file already existed, error recovering: %w", err)
		}
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
		return fmt.Errorf("unable to unlock data path: %w", err)
	}
	return nil
}

// ******* private helpers

// createPidfile creates the underling pidfile used by the lock
// This assumes that no previous lockfile should exist.
// returns a bool indicating if the file was created, and an error
func (lock *Locker) createPidfile(pid int) (bool, error) {
	_, err := os.Stat(lock.filePath)
	//file already exists, go into repair/handle fail mode
	if !os.IsNotExist(err) {
		return false, nil
	}

	new := pidfile{Pid: pid, WriteTime: time.Now()}
	encoded, err := json.Marshal(&new)
	if err != nil {
		return false, fmt.Errorf("error encoding json for pidfile: %w", err)
	}

	err = os.WriteFile(lock.filePath, encoded, os.FileMode(0600))
	if err != nil {
		return false, fmt.Errorf("error writing pidfile to %s: %w", lock.filePath, err)
	}
	return true, nil
}

// handleFailedLock will attempt to recover from a failed lock operation in a pid-aware way.
// The point of this is to deal with instances where an improper beat shutdown left us with
// a pre-existing pidfile for a beat process that no longer exists.
// The one argument tells the method if a pre-existing lockfile was already found.
func (lock *Locker) handleFailedLock() error {
	// read in whatever existing lockfile caused us to fail
	pf, err := lock.readExistingPidfile()
	if err != nil {
		return fmt.Errorf("error reading existing lockfile: %w", err)
	}

	// Case: the lockfile is locked, but by us. Probably a coding error,
	// and probably hard to do
	if pf.Pid == os.Getpid() {
		return fmt.Errorf("lockfile for beat has been locked twice by the same PID, potential bug.")
	}

	// Check to see if the PID found in the pidfile exists.
	// Checking if the pid exists in a cross-platform way is more fraught than it looks.
	// we have internal metrics libraries we can use for this, but all those will use APIs
	// dedicated to gathering extended process info and metrics, which can come with extra permissions hurdles,
	// making those methods more likely to return an error.
	exists, err := process.PidExistsWithContext(context.Background(), int32(pf.Pid))
	if err != nil {
		return fmt.Errorf("error looking up status for pid %d: %w", pf.Pid, err)
	}

	// Case: we've gotten a lock file for another process that's already running
	// This is the "base" lockfile case, which is two beats running from the same directory
	if exists {
		// This will make debugging easier for someone.
		state, err := metricproc.GetInfoForPid(resolve.NewTestResolver("/"), pf.Pid)
		// Above call is is auxiliary debug data, so we don't care too much if it fails
		debugString := fmt.Sprintf("process with PID %d", pf.Pid)
		if err == nil {
			debugString = fmt.Sprintf("process '%s' with pid %d", state.Name, pf.Pid)
		}
		return fmt.Errorf("connot start, data directory belongs to %s", debugString)
	}

	// Case: we have a lockfile, but the pid from the pidfile no longer exists
	// this was presumably due to the dirty shutdown.
	// Try to reset the lockfile and continue.
	lock.logger.Infof("%s shut down without removing previous lockfile, continuing", lock.beatName)

	// try to remove the lockfile
	// May or may not work, depending on os-specific details with lockfiles
	err = os.Remove(lock.fileLock.Path())
	if err != nil {
		lockfilePath := paths.Resolve(paths.Data, fmt.Sprintf("%s_%d.lock", lock.beatName, os.Getpid()))
		lock.logger.Warnf("failed to reset lockfile, cannot remove %s, continuing on with new lockfile name %s",
			lock.fileLock.Path(), lockfilePath)
		lock.filePath = lockfilePath
	}

	// reset the lockfile handler
	_, err = lock.createPidfile(os.Getpid())
	if err != nil {
		return fmt.Errorf("error creating new lockfile while recovering: %w", err)
	}
	lock.fileLock = flock.New(lock.filePath)
	return nil
}

// readExistingPidfile will read the contents of an existing pidfile
func (lock *Locker) readExistingPidfile() (pidfile, error) {
	rawPidfile, err := os.ReadFile(lock.filePath)
	if err != nil {
		return pidfile{}, fmt.Errorf("error reading pidfile from path %s", lock.filePath)
	}
	foundPidFile := pidfile{}
	err = json.Unmarshal(rawPidfile, &foundPidFile)
	if err != nil {
		return pidfile{}, fmt.Errorf("error reading JSON from pid file %s: %w", lock.filePath, err)
	}
	return foundPidFile, nil
}
