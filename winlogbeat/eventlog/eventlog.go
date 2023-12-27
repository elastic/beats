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
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Debug selectors used in this package.
const (
	debugSelector = "eventlog"
)

// Debug logging functions for this package.
var (
	debugf = logp.MakeDebug(debugSelector)
)

// EventLog is an interface to a Windows Event Log.
type EventLog interface {
	// Open the event log. state points to the last successfully read event
	// in this event log. Read will resume from the next record. To start reading
	// from the first event specify a zero-valued EventLogState.
	Open(state checkpoint.EventLogState) error

	// Read records from the event log. If io.EOF is returned you should stop
	// reading and close the log.
	Read() ([]Record, error)

	// Reset closes the event log channel to allow recovering from recoverable
	// errors. Open must be successfully called after a Reset before Read may
	// be called.
	Reset() error

	// Close the event log. It should not be re-opened after closing.
	Close() error

	// Name returns the event log's name.
	Name() string

	// Channel returns the event log's channel name.
	Channel() string

	// IsFile returns true if the event log is an evtx file.
	IsFile() bool
}

// Record represents a single event from the log.
type Record struct {
	winevent.Event
	File   string                   // Source file when event is from a file.
	API    string                   // The event log API type used to read the record.
	XML    string                   // XML representation of the event.
	Offset checkpoint.EventLogState // Position of the record within its source stream.
}

// ToEvent returns a new beat.Event containing the data from this Record.
func (e Record) ToEvent() beat.Event {
	win := e.Fields()

	_ = win.Delete("time_created")
	_, _ = win.Put("api", e.API)

	m := mapstr.M{
		"winlog": win,
	}

	// ECS data
	_, _ = m.Put("event.created", time.Now())

	eventCode, _ := win.GetValue("event_id")
	_, _ = m.Put("event.code", eventCode)
	_, _ = m.Put("event.kind", "event")
	_, _ = m.Put("event.provider", e.Provider.Name)

	rename(m, "winlog.outcome", "event.outcome")
	rename(m, "winlog.level", "log.level")
	rename(m, "winlog.message", "message")
	rename(m, "winlog.error.code", "error.code")
	rename(m, "winlog.error.message", "error.message")

	winevent.AddOptional(m, "log.file.path", e.File)
	winevent.AddOptional(m, "event.original", e.XML)
	winevent.AddOptional(m, "event.action", e.Task)
	winevent.AddOptional(m, "host.name", e.Computer)

	return beat.Event{
		Timestamp: e.TimeCreated.SystemTime,
		Fields:    m,
		Private:   e.Offset,
	}
}

// rename will rename a map entry overriding any previous value
func rename(m mapstr.M, oldKey, newKey string) {
	v, err := m.GetValue(oldKey)
	if err != nil {
		return
	}
	_, _ = m.Put(newKey, v)
	_ = m.Delete(oldKey)
}
