# Appdash in other languages

It is possible to have other programming languages communicate with a Go-based Appdash collection server (e.g. `cmd/appdash serve`).

This enables other applications, not written in Go, to communicate performance, debug information, and logs to a Appdash collection server.

Currently only a Python client is available, see the `python/` sub-directory for more information. The rest of this document will describe the details of how using Appdash from other languages would be achieved.

# Wire Protocol

Appdash collection servers communicate with clients via Google's protobuf (varint delimited) messages.

1. A varint is sent over the wire to communicate _how large the protobuf message is_. This is done because protobuf doesn't actually handle _streams of messages_, rather just the encoding/decoding of _single messages_.
2. The actual protobuf-encoded message is sent.

The actual protobuf file (which can be used to generate code for most languages) can be found in the `internal/wire/collector.proto` file.

We will now discuss in-depth the protobuf format, and how everything works.

# CollectPacket

A CollectPacket is the high-level message structure. It is sent from a Appdash client to a remote Appdash collection server (e.g. `cmd/appdash serve`). It is composed of a single _SpanID_ and any number of _Annotations_ associated with the identified span. For example, a _CollectPacket_ would be sent to the server to say what a span's name was, how long it took, etc.

When the server receives a CollectPacket, it stores the annotations associated with the span for later (in _a Store_). These annotations are unmarshaled into _Events_ which are then displayed nicely inside Appdash's web UI, which lets you analyse the trace, etc.

# SpanID

The SpanID portion of the protobuf file looks like:

```
// SpanID is the group of information which can uniquely identify the exact
// span being collected.
required group SpanID = 1 {
	// trace is the root ID of the tree that contains all of the spans
	// related to this one.
	required fixed64 trace = 2;

	// span is an ID that probabilistically uniquely identifies this span.
	required fixed64 span = 3;

	// parent is the ID of the parent span, if any.
	optional fixed64 parent = 4;
}
```

A SpanID is made up of three ID's in total:

1. The _trace ID_ (also called the _root ID_).
2. The _span ID_.
3. The _parent ID_, or zero.

Each ID is an unsigned 64-bit integer which has no special quality other than _uniquely identifying that span_. They are random numbers and are chosen purely to avoid collision with one another.

All spans in a trace share the same _trace ID_, and each span is a distant child of a parent span or the _root span (aka. trace)_.

# Annotation

The Annotation portion of the protobuf file looks like:

```
// Annotation is any number of annotations for the span to be collected.
repeated group Annotation = 5 {
	// key is the annotation's key.
	required string key = 6;

	// value is the annotation's value, which may be either human or
	// machine readable, depending on the schema of the event that
	// generated it.
	optional bytes value = 7;
}
```

As it looks, a annotation is a very arbitrary value which _puts meaning (or "annotates")_ a specific span. It has a key and a value that is either human or machine readable. You can define your own annotations as you see fit, but for most purposes you will utilize the ones exposed by apptrace by default.

Make explicit note that although annotations _may be_ ordered by specific clients -- there is no such requirement. Any robust client or server should appropriately _handle annotations as a list_ and _expect no specific order_ of them.

# Events

Events (things like associating a name, message, log event, SQL event, or HTTP event) are _marshaled_ into a set of multiple _annotations_. These annotations are then sent over the wire in the form of a CollectPacket, described above.

What follows is a description of the events which Appdash recognizes and renders neatly in the UI, and exactly how they are marshaled into annotations.

Note that the code is psuedo code, not actual code.

## SpanNameEvent

A SpanNameEvent sets the name of a span. It is marshaled into two span annotations:

```
Annotation(key="Name", value="theNameOfTheSpan")
Annotation(key="_schema:name", value="")
```

## MsgEvent

A MsgEvent represents a message event, with human readable text. Most clients
will emit a LogEvent instead, which also contains a timestamp.

```
Annotation(key="Msg", value="hello")
Annotation(key="_schema:msg", value="")
```

## LogEvent

A LogEvent represents a log message event, with human readable text and a timestamp.

```
Annotation(key="Msg", value="hello")
Annotation(key="Time", value="2015-02-19T19:31:17.451675861-07:00")
Annotation(key="_schema:log", value="")
```

## SQLEvent

A SQLEvent represents a SQL query event. It is marshaled into several annotations:

```
Annotation(key="Tag", value="fakeTag0")
Annotation(key="ClientSend", value="2015-02-19T19:31:17.449917809-07:00")
Annotation(key="ClientRecv", value="2015-02-19T19:31:18.442917809-07:00")
Annotation(key="SQL", value="SELECT * FROM table_name;")
Annotation(key="_schema:SQL", value="")
```

## HTTPServerEvent

A HTTPServerEvent represents an HTTP server serving a single client. It includes several annotations with information about the request, it's headers, etc.

Typically it will be preceded by a SpanNameEvent with the HTTP path requested, for example:

```
Annotation(key="Name", value="localhost:8699")
Annotation(key="_schema:name", value="")
```

Information about the request is placed into `Request.Foo` keys, information about the response is placed into `Response.Foo` keys, etc. A server handling a request to `/endpoint-A` would for instance generate annotations like:

```
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-A")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Accept-Encoding", value="gzip")
Annotation(key="Request.Headers.User-Agent", value="Go 1.1 package http")
Annotation(key="Request.Headers.Span-Id", value="3b83e3e091f8946a/76dc6cbdb3863717/a4475c5cc57a69d4")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="127.0.0.1:35741")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="Response.Headers.Span-Id", value="3b83e3e091f8946a/76dc6cbdb3863717/a4475c5cc57a69d4")
Annotation(key="Response.ContentLength", value="23")
Annotation(key="Route", value="/endpoint-A")
Annotation(key="User", value="")
Annotation(key="ServerRecv", value="2015-02-21T16:36:13.248971779-07:00")
Annotation(key="ServerSend", value="2015-02-21T16:36:13.499282101-07:00")
Annotation(key="_schema:HTTPServer", value="")
```

TODO: describe `Request.Headers.Span-Id`.

## HTTPClientEvent

A HTTPClientEvent represents a HTTP client making an outbound request. Very similiar to HTTPServerEvent, it includes several annotations with information about the request, it's headers, etc.

For example, a outbound request to `/endpoint-A`:

```
Annotation(key="ClientSend", value="2015-02-21T16:36:13.113231752-07:00")
Annotation(key="ClientRecv", value="2015-02-21T16:36:13.500518641-07:00")
Annotation(key="Request.Host", value="localhost:8699")
Annotation(key="Request.RemoteAddr", value="")
Annotation(key="Request.ContentLength", value="0")
Annotation(key="Request.Method", value="GET")
Annotation(key="Request.URI", value="/endpoint-A")
Annotation(key="Request.Proto", value="HTTP/1.1")
Annotation(key="Request.Headers.Span-Id", value="3b83e3e091f8946a/76dc6cbdb3863717/a4475c5cc57a69d4")
Annotation(key="Response.Headers.Content-Type", value="text/plain; charset=utf-8")
Annotation(key="Response.Headers.Date", value="Sat, 21 Feb 2015 23:36:13 GMT")
Annotation(key="Response.Headers.Content-Length", value="23")
Annotation(key="Response.ContentLength", value="23")
Annotation(key="Response.StatusCode", value="200")
Annotation(key="_schema:HTTPClient", value="")
```

## Larger-Scale Example

A larger-scale example of the annotations generated via running `cmd/appdash demo` is available. See the `demo-annotations.md` file for more information.
