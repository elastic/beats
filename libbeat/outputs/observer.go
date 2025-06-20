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

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Observer provides an interface used by outputs to report common events on
// documents/events being published and I/O workload.
type Observer interface {
	NewBatch([]publisher.Event) // report events in a new batch being processed

	RetryableErrors([]publisher.Event)  // report event had with retryable errors
	PermanentError(publisher.Event)     // report event has been dropped due to permanent errors
	PermanentErrors([]publisher.Event)  // report events has been dropped due to permanent errors
	DuplicateEvents([]publisher.Event)  // report event has been detected as duplicates (e.g. on resends)
	DeadLetterEvents([]publisher.Event) // report failed events ingested to dead letter index
	AckedEvent(publisher.Event)         // report acked event
	AckedEvents([]publisher.Event)      // report acked events
	ErrTooMany([]publisher.Event)       // report too many requests response for the event

	BatchSplit() // report a batch was split for being too large to ingest

	WriteError(error) // report an I/O error on write
	WriteBytes(int)   // report number of bytes being written
	ReadError(error)  // report an I/O error on read
	ReadBytes(int)    // report number of bytes being read

	ReportLatency(time.Duration) // report the duration a send to the output takes
}

type emptyObserver struct{}

var nilObserver = (*emptyObserver)(nil)

// NewNilObserver returns an observer implementation, ignoring all events.
func NewNilObserver() Observer {
	return nilObserver
}

func (*emptyObserver) NewBatch([]publisher.Event)         {}
func (*emptyObserver) ReportLatency(_ time.Duration)      {}
func (*emptyObserver) AckedEvent(publisher.Event)         {}
func (*emptyObserver) AckedEvents([]publisher.Event)      {}
func (*emptyObserver) DeadLetterEvents([]publisher.Event) {}
func (*emptyObserver) DuplicateEvents([]publisher.Event)  {}
func (*emptyObserver) RetryableErrors([]publisher.Event)  {}
func (*emptyObserver) PermanentError(publisher.Event)     {}
func (*emptyObserver) PermanentErrors([]publisher.Event)  {}
func (*emptyObserver) BatchSplit()                        {}
func (*emptyObserver) WriteError(error)                   {}
func (*emptyObserver) WriteBytes(int)                     {}
func (*emptyObserver) ReadError(error)                    {}
func (*emptyObserver) ReadBytes(int)                      {}
func (*emptyObserver) ErrTooMany([]publisher.Event)       {}
