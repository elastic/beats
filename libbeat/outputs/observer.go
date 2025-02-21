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
	NewBatch(int) // report new batch being processed with number of events
	NewBatchE([]publisher.Event)

	RetryableErrors(int) // report number of events with retryable errors
	PermanentErrors(int) // report number of events dropped due to permanent errors
	PermanentError(e publisher.Event)
	PermanentErrorsE([]publisher.Event)
	DuplicateEvents(int) // report number of events detected as duplicates (e.g. on resends)
	DuplicateEventsE([]publisher.Event)
	DeadLetterEvents(int) // report number of failed events ingested to dead letter index
	DeadLetterEventsE([]publisher.Event)
	AckedEvents(int) // report number of acked events
	AckedEvent(e publisher.Event)
	AckedEventsE([]publisher.Event)
	ErrTooMany(int) // report too many requests response

	BatchSplit() // report a batch was split for being too large to ingest

	WriteError(error) // report an I/O error on write
	WriteBytes(int)   // report number of bytes being written
	ReadError(error)  // report an I/O error on read
	ReadBytes(int)    // report number of bytes being read

	ReportLatency(time.Duration) // report the duration a send to the output takes
}

// Observer provides an interface used by outputs to report common events on
// documents/events being published and I/O workload.
// TODO: fix docs
type ObserverInputAware interface {
	NewBatchE([]publisher.Event)

	RetryableErrors(int) // report number of events with retryable errors
	PermanentError(e publisher.Event)
	PermanentErrorsE([]publisher.Event)
	DuplicateEventsE([]publisher.Event)
	DeadLetterEventsE([]publisher.Event)
	AckedEvent(e publisher.Event)
	AckedEventsE([]publisher.Event)
	ErrTooMany(int) // report too many requests response

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

func (*emptyObserver) NewBatch(int)                        {}
func (*emptyObserver) NewBatchE([]publisher.Event)         {}
func (*emptyObserver) ReportLatency(_ time.Duration)       {}
func (*emptyObserver) AckedEvents(int)                     {}
func (*emptyObserver) AckedEvent(publisher.Event)          {}
func (*emptyObserver) AckedEventsE([]publisher.Event)      {}
func (*emptyObserver) DeadLetterEvents(int)                {}
func (*emptyObserver) DeadLetterEventsE([]publisher.Event) {}
func (*emptyObserver) DuplicateEvents(int)                 {}
func (*emptyObserver) DuplicateEventsE([]publisher.Event)  {}
func (*emptyObserver) RetryableErrors(int)                 {}
func (*emptyObserver) PermanentErrors(int)                 {}
func (*emptyObserver) PermanentError(publisher.Event)      {}
func (*emptyObserver) PermanentErrorsE([]publisher.Event)  {}
func (*emptyObserver) BatchSplit()                         {}
func (*emptyObserver) WriteError(error)                    {}
func (*emptyObserver) WriteBytes(int)                      {}
func (*emptyObserver) ReadError(error)                     {}
func (*emptyObserver) ReadBytes(int)                       {}
func (*emptyObserver) ErrTooMany(int)                      {}
