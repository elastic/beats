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

package backoff

import "time"

// Backoff defines the interface for backoff strategies.
type Backoff interface {
	// Wait blocks for a duration of time governed by the backoff strategy.
	// If the timer is completed, Wait returns true, otherwise it returns false.
	Wait() bool

	// Reset resets the backoff duration to an initial value governed by the backoff strategy.
	Reset()

	// Last returns the time when the last call to Wait returned
	Last() time.Time
}

// WaitOnError is a convenience method, if an error is received it will block, if not errors is
// received, the backoff will be resetted.
func WaitOnError(b Backoff, err error) bool {
	if err == nil {
		b.Reset()
		return true
	}
	return b.Wait()
}
