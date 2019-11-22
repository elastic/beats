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

package instance

import (
	"os"

	"github.com/pkg/errors"
	flock "github.com/theckman/go-flock"

	"github.com/elastic/beats/libbeat/paths"
)

var (
	// ErrAlreadyLocked is returned when a lock on the data path is attempted but
	// unsuccessful because another Beat instance already has the lock on the same
	// data path.
	ErrAlreadyLocked = errors.New("data path already locked by another beat")
)

type locker struct {
	fl *flock.Flock
}

func newLocker(b *Beat) *locker {
	lockfilePath := paths.Resolve(paths.Data, b.Info.Beat+".lock")
	return &locker{
		fl: flock.NewFlock(lockfilePath),
	}
}

// lock attemps to acquire a lock on the data path for the currently-running
// Beat instance. If another Beats instance already has a lock on the same data path
// an ErrAlreadyLocked error is returned.
func (l *locker) lock() error {
	isLocked, err := l.fl.TryLock()
	if err != nil {
		return errors.Wrap(err, "unable to lock data path")
	}

	if !isLocked {
		return ErrAlreadyLocked
	}

	return nil
}

// unlock attempts to release the lock on a data path previously acquired via Lock().
func (l *locker) unlock() error {
	err := l.fl.Unlock()
	if err != nil {
		return errors.Wrap(err, "unable to unlock data path")
	}

	err = os.Remove(l.fl.Path())
	if err != nil {
		return errors.Wrap(err, "unable to unlock data path")
	}

	return nil
}
