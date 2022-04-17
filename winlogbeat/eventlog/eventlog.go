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
	"expvar"
	"strconv"
	"syscall"
	"time"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/winlogbeat/checkpoint"
	"github.com/menderesk/beats/v7/winlogbeat/sys/winevent"
)

// Debug selectors used in this package.
const (
	debugSelector  = "eventlog"
	detailSelector = "eventlog_detail"
)

// Debug logging functions for this package.
var (
	debugf  = logp.MakeDebug(debugSelector)
	detailf = logp.MakeDebug(detailSelector)
)

var (
	// dropReasons contains counters for the number of dropped events for each
	// reason.
	dropReasons = expvar.NewMap("drop_reasons")

	// readErrors contains counters for the read error types that occur.
	readErrors = expvar.NewMap("read_errors")
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

	// Close the event log. It should not be re-opened after closing.
	Close() error

	// Name returns the event log's name.
	Name() string
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

	win.Delete("time_created")
	win.Put("api", e.API)

	m := common.MapStr{
		"winlog": win,
	}

	// ECS data
	m.Put("event.created", time.Now())

	eventCode, _ := win.GetValue("event_id")
	m.Put("event.code", eventCode)
	m.Put("event.kind", "event")
	m.Put("event.provider", e.Provider.Name)

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
func rename(m common.MapStr, oldKey, newKey string) {
	v, err := m.GetValue(oldKey)
	if err != nil {
		return
	}
	m.Put(newKey, v)
	m.Delete(oldKey)
}

// incrementMetric increments a value in the specified expvar.Map. The key
// should be a windows syscall.Errno or a string. Any other types will be
// reported under the "other" key.
func incrementMetric(v *expvar.Map, key interface{}) {
	switch t := key.(type) {
	default:
		v.Add("other", 1)
	case string:
		v.Add(t, 1)
	case syscall.Errno:
		v.Add(strconv.Itoa(int(t)), 1)
	}
}
