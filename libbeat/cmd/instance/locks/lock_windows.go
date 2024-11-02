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
	"os"
)

// Unlock attempts to release the lock on a data path previously acquired via Lock(). This will unlock the file before it removes it.
func (lock *Locker) Unlock() error {
	// Removing a file that's locked seems to be an unsupported or undefined, and will often fail on Windows.
	// Reverse the order of operations, and unlock first, then remove.
	// This will slightly increase the odds of a race on Windows if we're in a tight restart loop,
	// as another beat can swoop in and lock the file before this beat removes it.
	err := lock.fileLock.Unlock()
	if err != nil {
		return fmt.Errorf("unable to unlock data path: %w", err)
	}

	err = os.Remove(lock.fileLock.Path())
	if err != nil {
		lock.logger.Warnf("could not remove lockfile at %s: %s", lock.fileLock.Path(), err)
	}

	return nil
}
