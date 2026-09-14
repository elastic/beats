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

package file

import (
	"fmt"
	"os"
	"time"
)

type rotateOpts struct {
	renameRetryDuration time.Duration
	renameRetryInterval time.Duration
}

type RotateOpt func(*rotateOpts)

func WithRenameRetries(duration, interval time.Duration) RotateOpt {
	return func(opts *rotateOpts) {
		opts.renameRetryDuration = duration
		opts.renameRetryInterval = interval
	}
}

func rename(src, dst string, options rotateOpts) error {
	// Perform a regular (non-retrying) rename unless all retry options are specified.
	if options.renameRetryDuration == 0 || options.renameRetryInterval == 0 {
		return os.Rename(src, dst)
	}

	// Attempt rename with retries every options.RenameRetryInterval until options.RenameRetryDuration
	// has elapsed. This is useful in cases where the destination file may be locked or in use.
	var err error
	for start := time.Now(); time.Since(start) < options.renameRetryDuration; time.Sleep(options.renameRetryInterval) {
		err = os.Rename(src, dst)
		if err == nil {
			// Rename succeeded; no more retries needed
			return nil
		}
	}

	if err != nil {
		return fmt.Errorf("failed to rename %s to %s after %v: %w", src, dst, options.renameRetryDuration, err)
	}

	return nil
}
