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

package outputs

import "time"

// Observer provides an interface used by outputs to report common events on
// documents/events being published and I/O workload.
type Observer interface {
	NewBatch(int) // report new batch being processed with number of events

	CancelledEvents(int) // report number of events whose Publish call was cancelled for reasons unrelated to ingestion error.
	RetryableErrors(int) // report number of events with retryable errors
	PermanentErrors(int) // report number of events dropped due to permanent errors
	DuplicateEvents(int) // report number of events detected as duplicates (e.g. on resends)
	AckedEvents(int)     // report number of acked events
	ErrTooMany(int)      // report too many requests response

	BatchSplit() // report a batch was split for being too large to ingest

	WriteError(error) // report an I/O error on write
	WriteBytes(int)   // report number of bytes being written
	ReadError(error)  // report an I/O error on read
	ReadBytes(int)    // report number of bytes being read

	ReportLatency(time.Duration) // report the duration a send to the output takes
}

type emptyObserver struct{}

var nilObserver = (*emptyObserver)(nil)

// NewNilObserver returns an oberserver implementation, ignoring all events.
func NewNilObserver() Observer {
	return nilObserver
}

func (*emptyObserver) NewBatch(int)                  {}
func (*emptyObserver) ReportLatency(_ time.Duration) {}
func (*emptyObserver) AckedEvents(int)               {}
func (*emptyObserver) DuplicateEvents(int)           {}
func (*emptyObserver) RetryableErrors(int)           {}
func (*emptyObserver) PermanentErrors(int)           {}
func (*emptyObserver) CancelledEvents(int)           {}
func (*emptyObserver) BatchSplit()                   {}
func (*emptyObserver) WriteError(error)              {}
func (*emptyObserver) WriteBytes(int)                {}
func (*emptyObserver) ReadError(error)               {}
func (*emptyObserver) ReadBytes(int)                 {}
func (*emptyObserver) ErrTooMany(int)                {}
