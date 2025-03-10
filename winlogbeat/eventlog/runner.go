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

package eventlog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

type Publisher interface {
	Publish(records []Record) error
}

func Run(
	reporter status.StatusReporter,
	ctx context.Context,
	api EventLog,
	evtCheckpoint checkpoint.EventLogState,
	publisher Publisher,
	log *logp.Logger,
) error {
	reporter.UpdateStatus(status.Starting, fmt.Sprintf("Starting to read from %s", api.Channel()))
	// setup closing the API if either the run function is signaled asynchronously
	// to shut down or when returning after io.EOF
	cancelCtx, cancelFn := ctxtool.WithFunc(ctx, func() {
		if err := api.Close(); err != nil {
			log.Errorw("error while closing Windows Event Log access", "error", err)
		}
	})
	defer cancelFn()

	openErrHandler := newExponentialLimitedBackoff(log, 5*time.Second, time.Minute, func(err error) bool {
		if IsRecoverable(err, api.IsFile()) {
			reporter.UpdateStatus(status.Degraded, fmt.Sprintf("Retrying to open %s: %v", api.Channel(), err))
			log.Errorw("encountered recoverable error when opening Windows Event Log", "error", err)
			return true
		}
		return false
	})

	readErrHandler := newExponentialLimitedBackoff(log, 5*time.Second, time.Minute, func(err error) bool {
		if IsRecoverable(err, api.IsFile()) {
			reporter.UpdateStatus(status.Degraded, fmt.Sprintf("Retrying to read from %s: %v", api.Channel(), err))
			log.Errorw("encountered recoverable error when reading from Windows Event Log", "error", err)
			if resetErr := api.Reset(); resetErr != nil {
				log.Errorw("error resetting Windows Event Log handle", "error", resetErr)
			}
			return true
		}
		return false
	})

runLoop:
	for cancelCtx.Err() == nil {
		openErr := api.Open(evtCheckpoint)
		if openErr != nil {
			if openErrHandler.backoff(cancelCtx, openErr) {
				continue runLoop
			}
			//nolint:nilerr // only log error if we are not shutting down
			if cancelCtx.Err() != nil {
				break runLoop
			}
			reporter.UpdateStatus(status.Failed, fmt.Sprintf("Failed to open %s: %v", api.Channel(), openErr))
			return fmt.Errorf("failed to open Windows Event Log channel %q: %w", api.Channel(), openErr)
		}

		log.Debug("windows event log opened successfully")

		// read loop
		for cancelCtx.Err() == nil {
			reporter.UpdateStatus(status.Running, fmt.Sprintf("Reading from %s", api.Channel()))
			records, readErr := api.Read()
			if readErr != nil {
				if readErrHandler.backoff(cancelCtx, readErr) {
					continue runLoop
				}

				if errors.Is(readErr, io.EOF) {
					log.Debugw("end of Winlog event stream reached", "error", readErr)
					break runLoop
				}

				//nolint:nilerr // only log error if we are not shutting down
				if cancelCtx.Err() != nil {
					break runLoop
				}

				reporter.UpdateStatus(status.Failed, fmt.Sprintf("Failed to read from %s: %v", api.Channel(), readErr))
				log.Errorw("error occurred while reading from Windows Event Log", "error", readErr)

				return readErr
			}

			if len(records) == 0 {
				_ = timed.Wait(cancelCtx, time.Second)
				continue
			}

			if err := publisher.Publish(records); err != nil {
				reporter.UpdateStatus(status.Failed, fmt.Sprintf("Publisher error: %v", err))
				return err
			}
		}
	}
	reporter.UpdateStatus(status.Stopped, "")
	return nil
}

type exponentialLimitedBackoff struct {
	log              *logp.Logger
	initialDelay     time.Duration
	maxDelay         time.Duration
	currentDelay     time.Duration
	backoffCondition func(error) bool
}

func newExponentialLimitedBackoff(log *logp.Logger, initialDelay, maxDelay time.Duration, errCondition func(error) bool) *exponentialLimitedBackoff {
	b := &exponentialLimitedBackoff{
		log:              log,
		initialDelay:     initialDelay,
		maxDelay:         maxDelay,
		backoffCondition: errCondition,
	}
	b.reset()
	return b
}

func (b *exponentialLimitedBackoff) backoff(ctx context.Context, err error) bool {
	if !b.backoffCondition(err) {
		b.reset()
		return false
	}
	b.log.Debugf("backing off, waiting for %v", b.currentDelay)
	select {
	case <-ctx.Done():
		return false
	case <-time.After(b.currentDelay):
		// Calculate the next delay, doubling it but not exceeding maxDelay
		b.currentDelay = time.Duration(math.Min(float64(b.maxDelay), float64(b.currentDelay*2)))
		return true
	}
}

func (b *exponentialLimitedBackoff) reset() {
	b.currentDelay = b.initialDelay
}
