package appdash

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	errMultipleFinishCalls = errors.New("multiple Recorder.Finish calls")
)

// A Recorder is associated with a span and records annotations on the
// span by sending them to a collector.
type Recorder struct {
	// Logger, if non-nil, causes errors to be written to this logger directly
	// instead of being manually checked via the Error method.
	Logger *log.Logger

	SpanID                   // the span ID that annotations are about
	annotations []Annotation // SpanID's annotations to be collected
	finished    bool         // finished is whether Recorder.Finish was called

	collector Collector // the collector to send to

	errors   []error    // errors since the last call to Errors
	errorsMu sync.Mutex // protects errors
}

// NewRecorder creates a new recorder for the given span and
// collector. If c is nil, NewRecorder panics.
func NewRecorder(span SpanID, c Collector) *Recorder {
	if c == nil {
		panic("Collector is nil")
	}
	return &Recorder{
		SpanID:    span,
		collector: c,
	}
}

// Child creates a new Recorder with the same collector and a new
// child SpanID whose parent is this recorder's SpanID.
func (r *Recorder) Child() *Recorder {
	return NewRecorder(NewSpanID(r.SpanID), r.collector)
}

// Name sets the name of this span.
func (r *Recorder) Name(name string) {
	r.Event(SpanNameEvent{name})
}

// Msg records a Msg event (an event with a human-readable message) on
// the span.
func (r *Recorder) Msg(msg string) {
	r.Event(Msg(msg))
}

// Log records a Log event (an event with the current timestamp and a
// human-readable message) on the span.
func (r *Recorder) Log(msg string) {
	r.Event(Log(msg))
}

// LogWithTimestamp records a Log event with an explicit timestamp
func (r *Recorder) LogWithTimestamp(msg string, timestamp time.Time) {
	r.Event(LogWithTimestamp(msg, timestamp))
}

// Event records any event that implements the Event, TimespanEvent, or
// TimestampedEvent interfaces.
func (r *Recorder) Event(e Event) {
	as, err := MarshalEvent(e)
	if err != nil {
		r.error("Event", err)
		return
	}
	r.annotations = append(r.annotations, as...)
}

// Finish finishes recording and saves the recorded information to the
// underlying collector. If Finish is not called, then no data will be written
// to the underlying collector.
// Finish must be called once, otherwise r.error is called, this constraint
// ensures that collector is called once per Recorder, in order to avoid
// for performance reasons extra operations(span look up & span's annotations update)
// within the collector.
func (r *Recorder) Finish() {
	if r.finished {
		r.error("Finish", errMultipleFinishCalls)
		return
	}
	r.finished = true
	r.Annotation(r.annotations...)
}

// Annotation records raw annotations on the span.
func (r *Recorder) Annotation(as ...Annotation) {
	if err := r.failsafeAnnotation(as...); err != nil {
		r.error("Annotation", err)
	}
}

// Annotation records raw annotations on the span.
func (r *Recorder) failsafeAnnotation(as ...Annotation) error {
	return r.collector.Collect(r.SpanID, as...)
}

// Errors returns all errors encountered by the Recorder since the
// last call to Errors. After calling Errors, the Recorder's list of
// errors is emptied.
func (r *Recorder) Errors() []error {
	r.errorsMu.Lock()
	errs := r.errors
	r.errors = nil
	r.errorsMu.Unlock()
	return errs
}

func (r *Recorder) error(method string, err error) {
	logMsg := fmt.Sprintf("Recorder.%s error: %s", method, err)
	as, _ := MarshalEvent(Log(logMsg))
	r.failsafeAnnotation(as...)

	// If we have a logger, we're not doing manual error checking but rather
	// just logging all errors.
	if r.Logger != nil {
		r.Logger.Println(logMsg)
		return
	}

	r.errorsMu.Lock()
	r.errors = append(r.errors, err)
	r.errorsMu.Unlock()
}
