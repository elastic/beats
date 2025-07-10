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

package client

import (
	"context"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
)

const maxRetries = 3
const initialInterval = 500 * time.Millisecond
const maxInterval = 1 * time.Minute

// Retry attempts to execute the provided function `fn` with a retry mechanism.
// It uses an exponential backoff strategy and retries up to a maximum number of attempts.
func Retry(ctx context.Context, fn func() error) (err error) {
	expBackoff := backoff.NewExpBackoff(ctx.Done(), initialInterval, maxInterval)

	for numTries := 0; ; numTries++ {
		err = fn()
		if err == nil {
			// function succeeded
			break
		}

		if numTries >= maxRetries {
			// maxRetries hit
			break
		}

		if !expBackoff.Wait() {
			// context cancelled
			break
		}
	}

	return err
}
