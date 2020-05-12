package opentracing

import (
	"log"
	"os"

	basictracer "github.com/opentracing/basictracer-go"
	opentracing "github.com/opentracing/opentracing-go"
	"sourcegraph.com/sourcegraph/appdash"
)

var _ opentracing.Tracer = NewTracer(nil) // Compile time check.

// Options defines options for a Tracer.
type Options struct {
	// ShouldSample is a function that allows deterministic sampling of a trace
	// using the randomly generated Trace ID. The decision is made when a new Trace
	// is created and is propagated to all of the trace's spans. For example,
	//
	//   func(traceID int64) { return traceID % 128 == 0 }
	//
	// samples 1 in every 128 traces, approximately.
	ShouldSample func(traceID uint64) bool

	// Verbose determines whether errors are logged to stdout only once or all
	// the time. By default, Verbose is false so only the first error is logged
	// and the rest are silenced.
	Verbose bool

	// Logger is used to log critical errors that can't be collected by the
	// Appdash Collector.
	Logger *log.Logger
}

func newLogger() *log.Logger {
	return log.New(os.Stderr, "opentracing: ", log.LstdFlags)
}

// DefaultOptions creates an Option with a sampling function that always return
// true and a logger that logs errors to stderr.
func DefaultOptions() Options {
	return Options{
		ShouldSample: func(_ uint64) bool { return true },
		Logger:       newLogger(),
	}
}

// NewTracer creates a new opentracing.Tracer implementation that reports
// spans to an Appdash collector.
//
// The Tracer created by NewTracer reports all spans by default. If you want to
// sample 1 in every N spans, see NewTracerWithOptions. Spans are written to
// the underlying collector when Finish() is called on the span. It is
// possible to buffer and write span on a time interval using appdash.ChunkedCollector.
//
// For example:
//
//   collector := appdash.NewLocalCollector(myAppdashStore)
//   chunkedCollector := appdash.NewChunkedCollector(collector)
//
//   tracer := NewTracer(chunkedCollector)
//
// If writing traces to a remote Appdash collector, an appdash.RemoteCollector would
// be needed, for example:
//
//   collector := appdash.NewRemoteCollector("localhost:8700")
//   tracer := NewTracer(collector)
//
// will record all spans to a collector server on localhost:8700.
func NewTracer(c appdash.Collector) opentracing.Tracer {
	return NewTracerWithOptions(c, DefaultOptions())
}

// NewTracerWithOptions creates a new opentracing.Tracer that records spans to
// the given appdash.Collector.
func NewTracerWithOptions(c appdash.Collector, options Options) opentracing.Tracer {
	opts := basictracer.DefaultOptions()
	opts.ShouldSample = options.ShouldSample
	opts.Recorder = NewRecorder(c, options)
	return basictracer.NewWithOptions(opts)
}
