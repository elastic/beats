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

// Package look defines common formatters for fields/types to be used when
// generating heartbeat events.
package look

import (
	"time"

	"github.com/elastic/beats/v8/libbeat/common"

	"github.com/elastic/beats/v8/heartbeat/reason"
)

// RTT formats a round-trip-time given as time.Duration into an
// event field. The duration is stored in `{"us": rtt}`.
// TODO: This returns a time.Duration, which isn't quite right. time.Duration
// represents nanos, whereas this really returns millis. It should probably
// return a plain int64 type instead.
func RTT(rtt time.Duration) common.MapStr {
	if rtt < 0 {
		rtt = 0
	}

	return common.MapStr{
		// cast to int64 since a go duration is a nano, but we want micros
		// This makes the types less confusing because other wise the duration
		// we get back has the wrong unit
		"us": rtt / (time.Microsecond / time.Nanosecond),
	}
}

// Reason formats an error into an error event field.
func Reason(err error) common.MapStr {
	if r, ok := err.(reason.Reason); ok {
		return reason.Fail(r)
	}
	return reason.FailIO(err)
}

// Timestamp converts an event timestamp into an compatible event timestamp for
// reporting.
func Timestamp(t time.Time) common.Time {
	return common.Time(t)
}

// Status creates a service status message from an error value.
func Status(err error) string {
	if err == nil {
		return "up"
	}
	return "down"
}
