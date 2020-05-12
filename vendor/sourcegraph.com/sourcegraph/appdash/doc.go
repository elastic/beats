// Package appdash provides a Go app performance tracing suite.
//
// Appdash allows you to trace the end-to-end performance of hierarchically
// structured applications. You can, for example, measure the time and see the
// detailed information of each HTTP request and SQL query made by an entire
// distributed web application.
//
// Web Front-end
//
// The cmd/appdash tool launches a web front-end which displays a web UI for
// viewing collected app traces. It is effectively a remote collector which your
// application can connect and send events to.
//
// Timing and application-specific metadata information can be viewed in a nice
// timeline view for each span (e.g. HTTP request) and it's children.
//
// The web front-end can also be embedded in your own Go HTTP server by
// utilizing the traceapp sub-package, which is effectively what cmd/appdash
// serves internally.
//
// HTTP and SQL tracing
//
// Sub-packages for HTTP and SQL event tracing are provided for use with
// appdash, which allows it to function equivalently to Google's Dapper and
// Twitter's Zipkin performance tracing suites.
//
// Appdash Structure
//
// The most high-level structure is a Trace, which represents the performance
// of an application from start to finish (in an HTTP application, for example,
// the loading of a web page).
//
// A Trace is a tree structure that is made up of several spans, which are just
// IDs (in an HTTP application, these ID's are passed through the stack via
// a few special headers).
//
// Each span ID has a set of Events that directly correspond to it inside a
// Collector. These events can be any combination of message, log, time-span,
// or time-stamped events (the cmd/appdash web UI displays these events as
// appropriate).
//
// Inside your application, a Recorder is used to send events to a Collector,
// which can be a remote HTTP(S) collector, a local in-memory or persistent
// collector, etc. Additionally, you can implement the Collector interface
// yourself and store events however you like.
package appdash
