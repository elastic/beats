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
	return fmt.Errorf("%s: %w", lock.fileLock.Path(), ErrAlreadyLocked)
}
