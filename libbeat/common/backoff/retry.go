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

<<<<<<<< HEAD:auditbeat/module/file_integrity/fileinfo_bsd.go
//go:build freebsd || openbsd || netbsd || darwin

package file_integrity
========
package backoff
>>>>>>>> 6b6941eed ([gcp] Add metadata cache (#44432)):libbeat/common/backoff/retry.go

import (
	"syscall"
	"time"
)

<<<<<<<< HEAD:auditbeat/module/file_integrity/fileinfo_bsd.go
func fileTimes(stat *syscall.Stat_t) (atime, mtime, ctime time.Time) {
	return time.Unix(0, stat.Atimespec.Nano()).UTC(),
		time.Unix(0, stat.Mtimespec.Nano()).UTC(),
		time.Unix(0, stat.Mtimespec.Nano()).UTC()
========
type Retryer struct {
	maxRetries      int
	initialInterval time.Duration
	maxInterval     time.Duration
}

func NewRetryer(maxRetries int, initialInterval, maxInterval time.Duration) *Retryer {
	return &Retryer{
		maxRetries:      maxRetries,
		initialInterval: initialInterval,
		maxInterval:     maxInterval,
	}
}

// Retry attempts to execute the provided function `fn` with a retry mechanism.
// It uses an exponential backoff strategy and retries up to a maximum number of attempts.
func (r *Retryer) Retry(ctx context.Context, fn func() error) (err error) {
	backoff := NewExpBackoff(ctx.Done(), r.initialInterval, r.maxInterval)

	for numTries := 0; ; numTries++ {
		err = fn()
		if err == nil {
			// function succeeded
			break
		}

		if numTries >= r.maxRetries {
			// maxRetries hit
			break
		}

		if !backoff.Wait() {
			// context cancelled
			break
		}
	}

	return err
>>>>>>>> 6b6941eed ([gcp] Add metadata cache (#44432)):libbeat/common/backoff/retry.go
}
