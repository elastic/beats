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

package module

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/testing"
)

// receiveOneEvent receives one event from the events channel then closes the
// returned done channel. If no events are received it will close the returned
// done channel after the timeout period elapses.
func receiveOneEvent(d testing.Driver, events <-chan beat.Event, timeout time.Duration) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		select {
		case <-time.Tick(timeout):
			d.Error("error", errors.New("timeout waiting for an event"))
		case event, ok := <-events:
			if !ok {
				return
			}

			// At this point in the pipeline the error has been converted to a
			// string and written to error.message.
			if v, err := event.Fields.GetValue("error.message"); err == nil {
				if errMsg, ok := v.(string); ok {
					d.Error("error", errors.New(errMsg))
					return
				}
			}

			outputJSON(d, &event)
		}
	}()

	return done
}

func outputJSON(d testing.Driver, event *beat.Event) {
	out := event.Fields.Clone()
	out.Put("@timestamp", common.Time(event.Timestamp))
	jsonData, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		d.Error("convert error", err)
		return
	}

	d.Result(string(jsonData))
}
