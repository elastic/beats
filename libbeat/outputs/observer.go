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

// Observer provides an interface used by outputs to report common events on
// documents/events being published and I/O workload.
type Observer interface {
	NewBatch(int)     // report new batch being processed with number of events
	Acked(int)        // report number of acked events
	Failed(int)       // report number of failed events
	Dropped(int)      // report number of dropped events
	Duplicate(int)    // report number of events detected as duplicates (e.g. on resends)
	Cancelled(int)    // report number of cancelled events
	WriteError(error) // report an I/O error on write
	WriteBytes(int)   // report number of bytes being written
	ReadError(error)  // report an I/O error on read
	ReadBytes(int)    // report number of bytes being read
}

type emptyObserver struct{}

var nilObserver = (*emptyObserver)(nil)

// NewNilObserver returns an oberserver implementation, ignoring all events.
func NewNilObserver() Observer {
	return nilObserver
}

func (*emptyObserver) NewBatch(int)     {}
func (*emptyObserver) Acked(int)        {}
func (*emptyObserver) Duplicate(int)    {}
func (*emptyObserver) Failed(int)       {}
func (*emptyObserver) Dropped(int)      {}
func (*emptyObserver) Cancelled(int)    {}
func (*emptyObserver) WriteError(error) {}
func (*emptyObserver) WriteBytes(int)   {}
func (*emptyObserver) ReadError(error)  {}
func (*emptyObserver) ReadBytes(int)    {}
