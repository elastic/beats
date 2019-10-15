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

package registrar

import (
	"path/filepath"

	"github.com/gofrs/flock"

	"github.com/pkg/errors"
)

var (
	// ErrAlreadyLocked is returned when a registry lock is attempted but unsuccessful because
	// another Filebeat instance already has the lock on the registry.
	ErrAlreadyLocked = errors.New("registry already locked by another filebeat")
)

// lock attemps to acquire a lock on the registry for the currently-running
// Filebeat instance. If another Filebeat instance already has a lock on the registry
// an ErrAlreadyLocked error is returned.
func (r *Registrar) lock() error {
	lock := r.newLock()

	isLocked, err := lock.TryLock()
	if err != nil {
		return errors.Wrap(err, "unable to lock registry")
	}

	if !isLocked {
		return ErrAlreadyLocked
	}

	return nil
}

// unlock attempts to release the lock on a registry previously acquired via Lock().
func (r *Registrar) unlock() error {
	lock := r.newLock()

	err := lock.Unlock()
	if err != nil {
		return errors.Wrap(err, "unable to unlock registry")
	}

	return nil
}

func (r *Registrar) newLock() *flock.Flock {
	registryPath := filepath.Dir(r.registryFile)
	lockfilePath := filepath.Join(registryPath, "filebeat.lock")

	return flock.New(lockfilePath)
}
