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
	"time"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

type Publisher interface {
	Publish(records []Record) error
}

func Run(
	ctx context.Context,
	api EventLog,
	evtCheckpoint checkpoint.EventLogState,
	publisher Publisher,
	log *logp.Logger,
) error {
	// setup closing the API if either the run function is signaled asynchronously
	// to shut down or when returning after io.EOF
	cancelCtx, cancelFn := ctxtool.WithFunc(ctx, func() {
		if err := api.Close(); err != nil {
			log.Errorw("Error while closing Windows Event Log access", "error", err)
		}
	})
	defer cancelFn()

	// Flag used to detect repeat "channel not found" errors, eliminating log spam.
	channelNotFoundErrDetected := false

runLoop:
	for {
		//nolint:nilerr // only log error if we are not shutting down
		if cancelCtx.Err() != nil {
			return nil
		}

		openErr := api.Open(evtCheckpoint)

		switch {
		case IsRecoverable(openErr):
			log.Errorw("Encountered recoverable error when opening Windows Event Log", "error", openErr)
			_ = timed.Wait(cancelCtx, 5*time.Second)
			continue
		case !api.IsFile() && IsChannelNotFound(openErr):
			if !channelNotFoundErrDetected {
				log.Errorw("Encountered channel not found error when opening Windows Event Log", "error", openErr)
			} else {
				log.Debugw("Encountered channel not found error when opening Windows Event Log", "error", openErr)
			}
			channelNotFoundErrDetected = true
			_ = timed.Wait(cancelCtx, 5*time.Second)
			continue
		case openErr != nil:
			return fmt.Errorf("failed to open Windows Event Log channel %q: %w", api.Channel(), openErr)
		}
		channelNotFoundErrDetected = false

		log.Debug("Windows Event Log opened successfully")

		// read loop
		for cancelCtx.Err() == nil {
			records, err := api.Read()
			if IsRecoverable(err) {
				log.Errorw("Encountered recoverable error when reading from Windows Event Log", "error", err)
				if resetErr := api.Reset(); resetErr != nil {
					log.Errorw("Error resetting Windows Event Log handle", "error", resetErr)
				}
				continue runLoop
			}
			if !api.IsFile() && IsChannelNotFound(err) {
				log.Errorw("Encountered channel not found error when reading from Windows Event Log", "error", err)
				if resetErr := api.Reset(); resetErr != nil {
					log.Errorw("Error resetting Windows Event Log handle", "error", resetErr)
				}
				continue runLoop
			}

			if err != nil {
				if errors.Is(err, io.EOF) {
					log.Debugw("End of Winlog event stream reached", "error", err)
					return nil
				}

				//nolint:nilerr // only log error if we are not shutting down
				if cancelCtx.Err() != nil {
					return nil
				}

				log.Errorw("Error occurred while reading from Windows Event Log", "error", err)
				return err
			}
			if len(records) == 0 {
				_ = timed.Wait(cancelCtx, time.Second)
				continue
			}

			if err := publisher.Publish(records); err != nil {
				return err
			}
		}
	}
}
