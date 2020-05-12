// Package opentracing provides an Appdash implementation of the OpenTracing
// API.
//
// The OpenTracing specification allows for Span Tags to have an arbitrary
// value. The way the Appdash.Recorder handles this is by converting the
// tag value into a string using the default format for its type. Arbitrary
// structs have their field name included.
//
// The Appdash implementation also does not record Log payloads.
package opentracing

import (
	"fmt"
	"log"
	"sync"

	basictracer "github.com/opentracing/basictracer-go"
	"sourcegraph.com/sourcegraph/appdash"
)

// Recorder implements the basictracer.Recorder interface.
type Recorder struct {
	collector appdash.Collector
	logOnce   sync.Once
	verbose   bool
	Log       *log.Logger
}

// NewRecorder forwards basictracer.RawSpans to an appdash.Collector.
func NewRecorder(collector appdash.Collector, opts Options) *Recorder {
	if opts.Logger == nil {
		opts.Logger = newLogger()
	}
	return &Recorder{
		collector: collector,
		verbose:   opts.Verbose,
		Log:       opts.Logger,
	}
}

// RecordSpan converts a RawSpan into the Appdash representation of a span
// and records it to the underlying collector.
func (r *Recorder) RecordSpan(sp basictracer.RawSpan) {
	if !sp.Context.Sampled {
		return
	}

	spanID := appdash.SpanID{
		Span:   appdash.ID(uint64(sp.Context.SpanID)),
		Trace:  appdash.ID(uint64(sp.Context.TraceID)),
		Parent: appdash.ID(uint64(sp.ParentSpanID)),
	}

	r.collectEvent(spanID, appdash.SpanName(sp.Operation))

	// Record all of the logs.
	for _, log := range sp.Logs {
		if logs, err := materializeWithJSON(log.Fields); err != nil {
			r.logError(spanID, err)
		} else {
			r.collectEvent(spanID, appdash.LogWithTimestamp(string(logs), log.Timestamp))
		}
	}

	for key, value := range sp.Tags {
		val := []byte(fmt.Sprintf("%+v", value))
		r.collectAnnotation(spanID, appdash.Annotation{Key: key, Value: val})
	}

	for key, val := range sp.Context.Baggage {
		r.collectAnnotation(spanID, appdash.Annotation{Key: key, Value: []byte(val)})
	}

	// Add the duration to the start time to get an approximate end time.
	approxEndTime := sp.Start.Add(sp.Duration)
	r.collectEvent(spanID, appdash.Timespan{S: sp.Start, E: approxEndTime})
}

// collectEvent marshals and collects the Event.
func (r *Recorder) collectEvent(spanID appdash.SpanID, e appdash.Event) {
	ans, err := appdash.MarshalEvent(e)
	if err != nil {
		r.logError(spanID, err)
		return
	}
	r.collectAnnotation(spanID, ans...)
}

func (r *Recorder) collectAnnotation(spanID appdash.SpanID, ans ...appdash.Annotation) {
	err := r.collector.Collect(spanID, ans...)
	if err != nil {
		r.logError(spanID, err)
	}
}

// logError converts an error into a log event and collects it.
// If for whatever reason the error can't be collected, it is logged to the
// Recorder's logger if it is non-nil.
func (r *Recorder) logError(spanID appdash.SpanID, err error) {
	ans, _ := appdash.MarshalEvent(appdash.Log(err.Error()))

	// At this point, something is definitely wrong.
	if err := r.collector.Collect(spanID, ans...); err != nil {
		if r.verbose {
			r.Log.Printf("Appdash Recorder collect error: %v\n", err)
		} else {
			r.logOnce.Do(func() { r.Log.Printf("Appdash Recorder collect error: %v\n", err) })
		}
	}
}
