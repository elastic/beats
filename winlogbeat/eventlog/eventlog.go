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
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/sys"
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
	sys.Event
	File   string                   // Source file when event is from a file.
	API    string                   // The event log API type used to read the record.
	XML    string                   // XML representation of the event.
	Offset checkpoint.EventLogState // Position of the record within its source stream.
}

// ToMapStr returns a new MapStr containing the data from this Record.
func (e Record) ToEvent() beat.Event {
	// Windows Log Specific data
	win := common.MapStr{
		"channel":       e.Channel,
		"event_id":      e.EventIdentifier.ID,
		"provider_name": e.Provider.Name,
		"record_id":     e.RecordID,
		"task":          e.Task,
		"api":           e.API,
	}
	addOptional(win, "computer_name", e.Computer)
	addOptional(win, "kernel_time", e.Execution.KernelTime)
	addOptional(win, "keywords", e.Keywords)
	addOptional(win, "opcode", e.Opcode)
	addOptional(win, "processor_id", e.Execution.ProcessorID)
	addOptional(win, "processor_time", e.Execution.ProcessorTime)
	addOptional(win, "provider_guid", e.Provider.GUID)
	addOptional(win, "session_id", e.Execution.SessionID)
	addOptional(win, "task", e.Task)
	addOptional(win, "user_time", e.Execution.UserTime)
	addOptional(win, "version", e.Version)
	// Correlation
	addOptional(win, "activity_id", e.Correlation.ActivityID)
	addOptional(win, "related_activity_id", e.Correlation.RelatedActivityID)
	// Execution
	addOptional(win, "process.pid", e.Execution.ProcessID)
	addOptional(win, "process.thread.id", e.Execution.ThreadID)

	if e.User.Identifier != "" {
		user := common.MapStr{
			"identifier": e.User.Identifier,
		}
		win["user"] = user
		addOptional(user, "name", e.User.Name)
		addOptional(user, "domain", e.User.Domain)
		addOptional(user, "type", e.User.Type.String())
	}

	addPairs(win, "event_data", e.EventData.Pairs)
	userData := addPairs(win, "user_data", e.UserData.Pairs)
	addOptional(userData, "xml_name", e.UserData.Name.Local)

	m := common.MapStr{
		"winlog": win,
	}

	// ECS data
	m.Put("event.kind", "event")
	m.Put("event.code", e.EventIdentifier.ID)
	m.Put("event.provider", e.Provider.Name)
	addOptional(m, "event.action", e.Task)
	addOptional(m, "host.name", e.Computer)

	m.Put("event.created", time.Now())

	addOptional(m, "log.file.path", e.File)
	addOptional(m, "log.level", strings.ToLower(e.Level))
	addOptional(m, "message", sys.RemoveWindowsLineEndings(e.Message))
	// Errors
	addOptional(m, "error.code", e.RenderErrorCode)
	if len(e.RenderErr) == 1 {
		addOptional(m, "error.message", e.RenderErr[0])
	} else {
		addOptional(m, "error.message", e.RenderErr)
	}

	addOptional(m, "event.original", e.XML)

	return beat.Event{
		Timestamp: e.TimeCreated.SystemTime,
		Fields:    m,
		Private:   e.Offset,
	}
}

// addOptional adds a key and value to the given MapStr if the value is not the
// zero value for the type of v. It is safe to call the function with a nil
// MapStr.
func addOptional(m common.MapStr, key string, v interface{}) {
	if m != nil && !isZero(v) {
		m.Put(key, v)
	}
}

// addPairs adds a new dictionary to the given MapStr. The key/value pairs are
// added to the new dictionary. If any keys are duplicates, the first key/value
// pair is added and the remaining duplicates are dropped.
//
// The new dictionary is added to the given MapStr and it is also returned for
// convenience purposes.
func addPairs(m common.MapStr, key string, pairs []sys.KeyValue) common.MapStr {
	if len(pairs) == 0 {
		return nil
	}

	h := make(common.MapStr, len(pairs))
	for i, kv := range pairs {
		// Ignore empty values.
		if kv.Value == "" {
			continue
		}

		// If the key name is empty or if it the default of "Data" then
		// assign a generic name of paramN.
		k := kv.Key
		if k == "" || k == "Data" {
			k = fmt.Sprintf("param%d", i+1)
		}

		// Do not overwrite.
		_, exists := h[k]
		if !exists {
			h[k] = sys.RemoveWindowsLineEndings(kv.Value)
		} else {
			debugf("Dropping key/value (k=%s, v=%s) pair because key already "+
				"exists. event=%+v", k, kv.Value, m)
		}
	}

	if len(h) == 0 {
		return nil
	}

	m[key] = h
	return h
}

// isZero return true if the given value is the zero value for its type.
func isZero(i interface{}) bool {
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
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
