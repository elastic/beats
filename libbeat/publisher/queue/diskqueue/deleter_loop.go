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

package diskqueue

import (
	"errors"
	"os"
	"time"
)

type deleterLoop struct {
	// The settings for the queue that created this loop.
	settings Settings

	// When one or more segments are ready to delete, they are sent to
	// requestChan. At most one deleteRequest may be outstanding at any time.
	requestChan chan deleterLoopRequest

	// When a request has been completely processed, a response is sent on
	// responseChan. If at least one deletion was successful, the response
	// is sent immediately. Otherwise, the deleter loop delays for
	// queueSettings.RetryWriteInterval before returning, so timed retries
	// don't have to be handled by the core loop.
	responseChan chan deleterLoopResponse
}

type deleterLoopRequest struct {
	segments []*queueSegment
}

type deleterLoopResponse struct {
	results []error
}

func newDeleterLoop(settings Settings) *deleterLoop {
	return &deleterLoop{
		settings: settings,

		requestChan:  make(chan deleterLoopRequest, 1),
		responseChan: make(chan deleterLoopResponse),
	}
}

func (dl *deleterLoop) run() {
	currentRetryInterval := dl.settings.RetryInterval
	for {
		request, ok := <-dl.requestChan
		if !ok {
			// The channel has been closed, time to shut down.
			return
		}
		results := []error{}
		deletedCount := 0
		for _, segment := range request.segments {
			path := dl.settings.segmentPath(segment.id)
			err := os.Remove(path)
			// We ignore errors caused by the file not existing: this shouldn't
			// happen, but it is still safe to report it as successfully removed.
			if err == nil || errors.Is(err, os.ErrNotExist) {
				deletedCount++
				results = append(results, nil)
			} else {
				results = append(results, err)
			}
		}
		if len(request.segments) > 0 && deletedCount == 0 {
			// If we were asked to delete segments but could not delete
			// _any_ of them, we haven't made progress. Returning an error
			// will log the issue and retry, but in this situation we
			// want to delay before retrying. The core loop itself can't
			// delay (it can never sleep or block), so we handle the
			// delay here, by waiting before sending the result.
			// The delay can be interrupted if the request channel is closed,
			// indicating queue shutdown.
			select {
			case <-time.After(currentRetryInterval):
			case <-dl.requestChan:
			}
			currentRetryInterval =
				dl.settings.nextRetryInterval(currentRetryInterval)
		} else {
			// If we made progress, reset the retry interval.
			currentRetryInterval = dl.settings.RetryInterval
		}
		dl.responseChan <- deleterLoopResponse{
			results: results,
		}
	}
}
