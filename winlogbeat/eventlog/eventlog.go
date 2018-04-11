package eventlog

import (
	"expvar"
	"fmt"
	"reflect"
	"strconv"
	"syscall"

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

	// Read records from the event log.
	Read() ([]Record, error)

	// Close the event log. It should not be re-opened after closing.
	Close() error

	// Name returns the event log's name.
	Name() string
}

// Record represents a single event from the log.
type Record struct {
	sys.Event
	API    string                   // The event log API type used to read the record.
	XML    string                   // XML representation of the event.
	Offset checkpoint.EventLogState // Position of the record within its source stream.
}

// ToMapStr returns a new MapStr containing the data from this Record.
func (e Record) ToEvent() beat.Event {
	m := common.MapStr{
		"type":          e.API,
		"log_name":      e.Channel,
		"source_name":   e.Provider.Name,
		"computer_name": e.Computer,
		"record_number": strconv.FormatUint(e.RecordID, 10),
		"event_id":      e.EventIdentifier.ID,
	}

	addOptional(m, "xml", e.XML)
	addOptional(m, "provider_guid", e.Provider.GUID)
	addOptional(m, "version", e.Version)
	addOptional(m, "level", e.Level)
	addOptional(m, "task", e.Task)
	addOptional(m, "opcode", e.Opcode)
	addOptional(m, "keywords", e.Keywords)
	addOptional(m, "message", sys.RemoveWindowsLineEndings(e.Message))
	addOptional(m, "message_error", e.RenderErr)

	// Correlation
	addOptional(m, "activity_id", e.Correlation.ActivityID)
	addOptional(m, "related_activity_id", e.Correlation.RelatedActivityID)

	// Execution
	addOptional(m, "process_id", e.Execution.ProcessID)
	addOptional(m, "thread_id", e.Execution.ThreadID)
	addOptional(m, "processor_id", e.Execution.ProcessorID)
	addOptional(m, "session_id", e.Execution.SessionID)
	addOptional(m, "kernel_time", e.Execution.KernelTime)
	addOptional(m, "user_time", e.Execution.UserTime)
	addOptional(m, "processor_time", e.Execution.ProcessorTime)

	if e.User.Identifier != "" {
		user := common.MapStr{
			"identifier": e.User.Identifier,
		}
		m["user"] = user

		addOptional(user, "name", e.User.Name)
		addOptional(user, "domain", e.User.Domain)
		addOptional(user, "type", e.User.Type.String())
	}

	addPairs(m, "event_data", e.EventData.Pairs)
	userData := addPairs(m, "user_data", e.UserData.Pairs)
	addOptional(userData, "xml_name", e.UserData.Name.Local)

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
		m[key] = v
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
			debugf("Droping key/value (k=%s, v=%s) pair because key already "+
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
