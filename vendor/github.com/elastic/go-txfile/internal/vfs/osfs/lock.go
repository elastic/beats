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

package osfs

import (
	"errors"

	"github.com/gofrs/flock"

	"github.com/elastic/go-txfile/internal/vfs"
)

const (
	lockExt = ".lock"
)

type lockState struct {
	*flock.Flock
}

var (
	errAlreadyLocked     = errors.New("file is already locked")
	errNotLocked         = errors.New("file is not locked")
	errCanNotBeLockedNow = errors.New("file can not be locked right now")
)

func (f *File) Lock(exclusive, blocking bool) error {
	err := f.doLock(blocking)
	return f.wrapErrKind("file/lock", vfs.ErrLockFailed, err)
}

func (f *File) Unlock() error {
	err := f.doUnlock()
	return f.wrapErrKind("file/unlock", vfs.ErrLockFailed, err)
}

func (f *File) doLock(blocking bool) error {
	if f.state.lock.Flock != nil {
		return errAlreadyLocked
	}

	var ok bool
	var err error
	lock := flock.NewFlock(f.Name() + lockExt)
	if blocking {
		err = lock.Lock()
		ok = err == nil
	} else {
		ok, err = lock.TryLock()
	}

	if err != nil {
		return err
	}
	if !ok {
		return errCanNotBeLockedNow
	}

	f.state.lock.Flock = lock
	return nil
}

func (f *File) doUnlock() error {
	if f.state.lock.Flock == nil {
		return errNotLocked
	}

	err := f.state.lock.Unlock()
	if err == nil {
		f.state.lock.Flock = nil
	}
	return err
}
