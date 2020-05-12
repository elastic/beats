package appdash

import (
	"fmt"
	"reflect"
	"time"
)

// An Event is a record of the occurrence of something.
type Event interface {
	// Schema should return the event's schema, a constant string, for example
	// the sqltrace package defines SQLEvent which returns just "SQL".
	Schema() string
}

// ImportantEvent is an event that can describe in particular which annotation
// keys it finds important. Only important annotation keys are displayed in the
// web UI by default.
type ImportantEvent interface {
	Important() []string
}

// EventMarshaler is the interface implemented by an event that can
// marshal a representation of itself into annotations.
type EventMarshaler interface {
	// MarshalEvent should marshal this event itself into a set of annotations, or
	// return an error.
	MarshalEvent() (Annotations, error)
}

// EventUnmarshaler is the interface implemented by an event that can
// unmarshal an annotation representation of itself.
type EventUnmarshaler interface {
	// UnmarshalEvent should unmarshal the given annotations into a event of the
	// same type, or return an error.
	UnmarshalEvent(Annotations) (Event, error)
}

const SchemaPrefix = "_schema:"

// MarshalEvent marshals an event into annotations.
func MarshalEvent(e Event) (Annotations, error) {
	// Handle event marshalers.
	if v, ok := e.(EventMarshaler); ok {
		as, err := v.MarshalEvent()
		if err != nil {
			return nil, err
		}
		as = append(as, Annotation{Key: SchemaPrefix + e.Schema()})
		return as, nil
	}

	var as Annotations
	flattenValue("", reflect.ValueOf(e), func(k, v string) {
		as = append(as, Annotation{Key: k, Value: []byte(v)})
	})
	as = append(as, Annotation{Key: SchemaPrefix + e.Schema()})
	return as, nil
}

// An EventSchemaUnmarshalError is when annotations are attempted to
// be unmarshaled into an event object that does not match any of the
// schemas in the annotations.
type EventSchemaUnmarshalError struct {
	Found  []string // schemas found in the annotations
	Target string   // schema of the target event
}

func (e *EventSchemaUnmarshalError) Error() string {
	return fmt.Sprintf("event: can't unmarshal annotations with schemas %v into event of schema %s", e.Found, e.Target)
}

// UnmarshalEvent unmarshals annotations into an event.
func UnmarshalEvent(as Annotations, e Event) error {
	aSchemas := as.schemas()
	schemaOK := false
	for _, s := range aSchemas {
		if s == e.Schema() {
			schemaOK = true
			break
		}
	}
	if !schemaOK {
		return &EventSchemaUnmarshalError{Found: aSchemas, Target: e.Schema()}
	}

	// Handle event unmarshalers.
	if v, ok := e.(EventUnmarshaler); ok {
		ev, err := v.UnmarshalEvent(as)
		if err != nil {
			return err
		}
		reflect.Indirect(reflect.ValueOf(e)).Set(reflect.ValueOf(ev))
		return nil
	}

	unflattenValue("", reflect.ValueOf(&e), reflect.TypeOf(&e), mapToKVs(as.StringMap()))
	return nil
}

// RegisterEvent registers an event type for use with UnmarshalEvents.
//
// Events must be registered with this package in order for unmarshaling to
// work. Much like the image package, sometimes blank imports will be used for
// packages that register Appdash events with this package:
//
//  import(
//      _ "sourcegraph.com/sourcegraph/appdash/httptrace"
//      _ "sourcegraph.com/sourcegraph/appdash/sqltrace"
//  )
//
func RegisterEvent(e Event) {
	if _, present := registeredEvents[e.Schema()]; present {
		panic("event schema is already registered: " + e.Schema())
	}
	if e.Schema() == "" {
		panic("event schema is empty")
	}
	registeredEvents[e.Schema()] = e
}

var registeredEvents = map[string]Event{} // event schema -> event type

func init() {
	RegisterEvent(SpanNameEvent{})
	RegisterEvent(logEvent{})
	RegisterEvent(msgEvent{})
	RegisterEvent(timespanEvent{})
	RegisterEvent(Timespan{})
}

// UnmarshalEvents unmarshals all events found in anns into
// events. Any schemas found in anns that were not registered (using
// RegisterEvent) are ignored; missing a schema is not an error.
func UnmarshalEvents(anns Annotations, events *[]Event) error {
	schemas := anns.schemas()
	for _, schema := range schemas {
		ev := registeredEvents[schema]
		if ev == nil {
			continue
		}
		evv := reflect.New(reflect.TypeOf(ev))
		if err := UnmarshalEvent(anns, evv.Interface().(Event)); err != nil {
			return err
		}
		*events = append(*events, evv.Elem().Interface().(Event))
	}
	return nil
}

// A SpanNameEvent event sets a span's name.
type SpanNameEvent struct{ Name string }

func (SpanNameEvent) Schema() string { return "name" }

// SpanName returns an Event containing a human readable Span name.
func SpanName(name string) Event {
	return SpanNameEvent{Name: name}
}

// Msg returns an Event that contains only a human-readable message.
func Msg(msg string) Event {
	return msgEvent{Msg: msg}
}

type msgEvent struct {
	Msg string
}

func (msgEvent) Schema() string { return "msg" }

// A TimespanEvent is an Event with a start and an end time.
type TimespanEvent interface {
	Event
	Start() time.Time
	End() time.Time
}

// timespanEvent implements the TimespanEvent interface.
type timespanEvent struct {
	S, E time.Time
}

func (timespanEvent) Schema() string      { return "timespan" }
func (ev timespanEvent) Start() time.Time { return ev.S }
func (ev timespanEvent) End() time.Time   { return ev.E }

// Timespan is an event that satisfies the appdash.TimespanEvent interface.
// This is used to show its beginning and end times of a span.
type Timespan struct {
	S time.Time `trace:"Span.Start"`
	E time.Time `trace:"Span.End"`
}

func (s Timespan) Schema() string   { return "Timespan" }
func (s Timespan) Start() time.Time { return s.S }
func (s Timespan) End() time.Time   { return s.E }

// A TimestampedEvent is an Event with a timestamp.
type TimestampedEvent interface {
	Timestamp() time.Time
}

// Log returns an Event whose timestamp is the current time that
// contains only a human-readable message.
func Log(msg string) Event {
	return logEvent{Msg: msg, Time: time.Now()}
}

// LogWithTimestamp returns an Event with an explicit timestamp that contains
// only a human readable message.
func LogWithTimestamp(msg string, timestamp time.Time) logEvent {
	return logEvent{Msg: msg, Time: timestamp}
}

type logEvent struct {
	Msg  string
	Time time.Time
}

func (logEvent) Schema() string { return "log" }

func (e *logEvent) Timestamp() time.Time { return e.Time }
