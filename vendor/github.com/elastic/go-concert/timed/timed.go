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

package timed

import "time"

type canceler interface {
	Done() <-chan struct{}
	Err() error
}

// Wait blocks for the configuration duration or until the passed context
// signal canceling.
// Wait return ctx.Err() if the context got cancelled early. If the duration
// has passed without the context being cancelled, Wait returns nil.
//
// Example:
//   fmt.Printf("wait for 5 seconds...")
//   if err := Wait(ctx, 5 * time.Second); err != nil {
//       fmt.Printf("shutting down")
//       return err
//   }
//   fmt.Println("done")
func Wait(ctx canceler, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Periodic executes fn on every period. Periodic returns if the context is
// cancelled.
// The underlying ticket adjusts the intervals or drops ticks to make up for
// slow runs of fn. If fn is active, Periodic will only return when fn has
// finished.
// The period must be greater than 0, otherwise Periodic panics.
//
// If fn returns an Error, then the loop is stopped and the functions error is
// returned directly. On normal termination the contexts reported error will be
// reported.
func Periodic(ctx canceler, period time.Duration, fn func() error) error {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	done := ctx.Done()
	for {
		// always check for cancel first, to not accidentally trigger another run if
		// the context is already cancelled, but we have already received another
		// ticker signal
		select {
		case <-done:
			return ctx.Err()
		default:
		}

		select {
		case <-ticker.C:
			if err := fn(); err != nil {
				return err
			}
		case <-done:
			return ctx.Err()
		}
	}
}
