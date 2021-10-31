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

package model // import "go.elastic.co/apm/model"

import (
	"net/http"
	"net/url"
	"time"
)

// Service represents the service handling transactions being traced.
type Service struct {
	// Name is the immutable name of the service.
	Name string `json:"name,omitempty"`

	// Version is the version of the service, if it has one.
	Version string `json:"version,omitempty"`

	// Environment is the name of the service's environment, if it has
	// one, e.g. "production" or "staging".
	Environment string `json:"environment,omitempty"`

	// Agent holds information about the Elastic APM agent tracing this
	// service's transactions.
	Agent *Agent `json:"agent,omitempty"`

	// Framework holds information about the service's framework, if any.
	Framework *Framework `json:"framework,omitempty"`

	// Language holds information about the programming language in which
	// the service is written.
	Language *Language `json:"language,omitempty"`

	// Runtime holds information about the programming language runtime
	// running this service.
	Runtime *Runtime `json:"runtime,omitempty"`

	// Node holds unique information about each service node
	Node *ServiceNode `json:"node,omitempty"`
}

// Agent holds information about the Elastic APM agent.
type Agent struct {
	// Name is the name of the Elastic APM agent, e.g. "Go".
	Name string `json:"name"`

	// Version is the version of the Elastic APM agent, e.g. "1.0.0".
	Version string `json:"version"`
}

// Framework holds information about the framework (typically web)
// used by the service.
type Framework struct {
	// Name is the name of the framework.
	Name string `json:"name"`

	// Version is the version of the framework.
	Version string `json:"version"`
}

// Language holds information about the programming language used.
type Language struct {
	// Name is the name of the programming language.
	Name string `json:"name"`

	// Version is the version of the programming language.
	Version string `json:"version,omitempty"`
}

// Runtime holds information about the programming language runtime.
type Runtime struct {
	// Name is the name of the programming language runtime.
	Name string `json:"name"`

	// Version is the version of the programming language runtime.
	Version string `json:"version"`
}

// ServiceNode holds unique information about each service node
type ServiceNode struct {
	// ConfiguredName holds the name of the service node
	ConfiguredName string `json:"configured_name,omitempty"`
}

// System represents the system (operating system and machine) running the
// service.
type System struct {
	// Architecture is the system's hardware architecture.
	Architecture string `json:"architecture,omitempty"`

	// Hostname is the system's hostname.
	Hostname string `json:"hostname,omitempty"`

	// Platform is the system's platform, or operating system name.
	Platform string `json:"platform,omitempty"`

	// Container describes the container running the service.
	Container *Container `json:"container,omitempty"`

	// Kubernetes describes the kubernetes node and pod running the service.
	Kubernetes *Kubernetes `json:"kubernetes,omitempty"`
}

// Process represents an operating system process.
type Process struct {
	// Pid is the process ID.
	Pid int `json:"pid"`

	// Ppid is the parent process ID, if known.
	Ppid *int `json:"ppid,omitempty"`

	// Title is the title of the process.
	Title string `json:"title,omitempty"`

	// Argv holds the command line arguments used to start the process.
	Argv []string `json:"argv,omitempty"`
}

// Container represents the container (e.g. Docker) running the service.
type Container struct {
	// ID is the unique container ID.
	ID string `json:"id"`
}

// Kubernetes describes properties of the Kubernetes node and pod in which
// the service is running.
type Kubernetes struct {
	// Namespace names the Kubernetes namespace in which the pod exists.
	Namespace string `json:"namespace,omitempty"`

	// Node describes the Kubernetes node running the service's pod.
	Node *KubernetesNode `json:"node,omitempty"`

	// Pod describes the Kubernetes pod running the service.
	Pod *KubernetesPod `json:"pod,omitempty"`
}

// KubernetesNode describes a Kubernetes node.
type KubernetesNode struct {
	// Name holds the node name.
	Name string `json:"name,omitempty"`
}

// KubernetesPod describes a Kubernetes pod.
type KubernetesPod struct {
	// Name holds the pod name.
	Name string `json:"name,omitempty"`

	// UID holds the pod UID.
	UID string `json:"uid,omitempty"`
}

// Cloud represents the cloud in which the service is running.
type Cloud struct {
	// Provider is the cloud provider name, e.g. aws, azure, gcp.
	Provider string `json:"provider"`

	// Region is the cloud region name, e.g. us-east-1.
	Region string `json:"region,omitempty"`

	// AvailabilityZone is the cloud availability zone name, e.g. us-east-1a.
	AvailabilityZone string `json:"availability_zone,omitempty"`

	// Instance holds information about the cloud instance (virtual machine).
	Instance *CloudInstance `json:"instance,omitempty"`

	// Machine also holds information about the cloud instance (virtual machine).
	Machine *CloudMachine `json:"machine,omitempty"`

	// Account holds information about the cloud account.
	Account *CloudAccount `json:"account,omitempty"`

	// Project holds information about the cloud project.
	Project *CloudProject `json:"project,omitempty"`
}

// CloudInstance holds information about a cloud instance (virtual machine).
type CloudInstance struct {
	// ID holds the cloud instance identifier.
	ID string `json:"id,omitempty"`

	// ID holds the cloud instance name.
	Name string `json:"name,omitempty"`
}

// CloudMachine holds information about a cloud instance (virtual machine).
type CloudMachine struct {
	// Type holds the cloud instance type, e.g. t2.medium.
	Type string `json:"type,omitempty"`
}

// CloudAccount holds information about a cloud account.
type CloudAccount struct {
	// ID holds the cloud account identifier.
	ID string `json:"id,omitempty"`

	// ID holds the cloud account name.
	Name string `json:"name,omitempty"`
}

// CloudProject holds information about a cloud project.
type CloudProject struct {
	// ID holds the cloud project identifier.
	ID string `json:"id,omitempty"`

	// Name holds the cloud project name.
	Name string `json:"name,omitempty"`
}

// Transaction represents a transaction handled by the service.
type Transaction struct {
	// ID holds the 64-bit hex-encoded transaction ID.
	ID SpanID `json:"id"`

	// TraceID holds the ID of the trace that this transaction is a part of.
	TraceID TraceID `json:"trace_id"`

	// ParentID holds the ID of the transaction's parent span or transaction.
	ParentID SpanID `json:"parent_id,omitempty"`

	// Name holds the name of the transaction.
	Name string `json:"name"`

	// Type identifies the service-domain specific type of the request,
	// e.g. "request" or "backgroundjob".
	Type string `json:"type"`

	// Timestamp holds the time at which the transaction started.
	Timestamp Time `json:"timestamp"`

	// Duration records how long the transaction took to complete,
	// in milliseconds.
	Duration float64 `json:"duration"`

	// Result holds the result of the transaction, e.g. the status code
	// for HTTP requests.
	Result string `json:"result,omitempty"`

	// Context holds contextual information relating to the transaction.
	Context *Context `json:"context,omitempty"`

	// Sampled indicates that the transaction was sampled, and
	// includes all available information. Non-sampled transactions
	// omit Context.
	//
	// If Sampled is unspecified (nil), it is equivalent to setting
	// it to true.
	Sampled *bool `json:"sampled,omitempty"`

	// SampleRate holds the sample rate in effect when the trace was started,
	// if known. This is used by the server to aggregate transaction metrics.
	SampleRate *float64 `json:"sample_rate,omitempty"`

	// SpanCount holds statistics on spans within a transaction.
	SpanCount SpanCount `json:"span_count"`

	// Outcome holds the transaction outcome: success, failure, or unknown.
	Outcome string `json:"outcome,omitempty"`
}

// SpanCount holds statistics on spans within a transaction.
type SpanCount struct {
	// Dropped holds the number of spans dropped within a transaction.
	// This does not include spans that were started and dropped due
	// to full buffers, network errors, etc.
	Dropped int `json:"dropped"`

	// Started holds the number of spans started within a transaction.
	Started int `json:"started"`
}

// Span represents a span within a transaction.
type Span struct {
	// Name holds the name of the span.
	Name string `json:"name"`

	// Timestamp holds the time at which the span started.
	Timestamp Time `json:"timestamp"`

	// Duration holds the duration of the span, in milliseconds.
	Duration float64 `json:"duration"`

	// Type identifies the overarching type of the span,
	// e.g. "db" or "external".
	Type string `json:"type"`

	// Subtype identifies the subtype of the span,
	// e.g. "mysql" or "http".
	Subtype string `json:"subtype,omitempty"`

	// Action identifies the action that is being undertaken, e.g. "query".
	Action string `json:"action,omitempty"`

	// ID holds the ID of the span.
	ID SpanID `json:"id"`

	// TransactionID holds the ID of the transaction of which the span is a part.
	TransactionID SpanID `json:"transaction_id,omitempty"`

	// TraceID holds the ID of the trace that this span is a part of.
	TraceID TraceID `json:"trace_id"`

	// ParentID holds the ID of the span's parent (span or transaction).
	ParentID SpanID `json:"parent_id,omitempty"`

	// SampleRate holds the sample rate in effect when the trace was started,
	// if known. This is used by the server to aggregate span metrics.
	SampleRate *float64 `json:"sample_rate,omitempty"`

	// Context holds contextual information relating to the span.
	Context *SpanContext `json:"context,omitempty"`

	// Stacktrace holds stack frames corresponding to the span.
	Stacktrace []StacktraceFrame `json:"stacktrace,omitempty"`

	// Outcome holds the span outcome: success, failure, or unknown.
	Outcome string `json:"outcome,omitempty"`
}

// SpanContext holds contextual information relating to the span.
type SpanContext struct {
	// Destination holds information about a destination service.
	Destination *DestinationSpanContext `json:"destination,omitempty"`

	// Database holds contextual information for database
	// operation spans.
	Database *DatabaseSpanContext `json:"db,omitempty"`

	// HTTP holds contextual information for HTTP client request spans.
	HTTP *HTTPSpanContext `json:"http,omitempty"`

	// Tags holds user-defined key/value pairs.
	Tags IfaceMap `json:"tags,omitempty"`
}

// DestinationSpanContext holds contextual information about the destination
// for a span that relates to an operation involving an external service.
type DestinationSpanContext struct {
	// Address holds the network address of the destination service.
	// This may be a hostname, FQDN, or (IPv4 or IPv6) network address.
	Address string `json:"address,omitempty"`

	// Port holds the network port for the destination service.
	Port int `json:"port,omitempty"`

	// Service holds additional destination service context.
	Service *DestinationServiceSpanContext `json:"service,omitempty"`
}

// DestinationServiceSpanContext holds contextual information about a
// destination service,.
type DestinationServiceSpanContext struct {
	// Type holds the destination service type.
	Type string `json:"type,omitempty"`

	// Name holds the destination service name.
	Name string `json:"name,omitempty"`

	// Resource identifies the destination service
	// resource, e.g. a URI or message queue name.
	Resource string `json:"resource,omitempty"`
}

// DatabaseSpanContext holds contextual information for database
// operation spans.
type DatabaseSpanContext struct {
	// Instance holds the database instance name.
	Instance string `json:"instance,omitempty"`

	// Statement holds the database statement (e.g. query).
	Statement string `json:"statement,omitempty"`

	// RowsAffected holds the number of rows affected by the
	// database operation.
	RowsAffected *int64 `json:"rows_affected,omitempty"`

	// Type holds the database type. For any SQL database,
	// this should be "sql"; for others, the lower-cased
	// database category, e.g. "cassandra", "hbase", "redis".
	Type string `json:"type,omitempty"`

	// User holds the username used for database access.
	User string `json:"user,omitempty"`
}

// HTTPSpanContext holds contextual information for HTTP client request spans.
type HTTPSpanContext struct {
	// URL is the request URL.
	URL *url.URL

	// StatusCode holds the HTTP response status code.
	StatusCode int `json:"status_code,omitempty"`
}

// Context holds contextual information relating to a transaction or error.
type Context struct {
	// Custom holds custom context relating to the transaction or error.
	Custom IfaceMap `json:"custom,omitempty"`

	// Request holds details of the HTTP request relating to the
	// transaction or error, if relevant.
	Request *Request `json:"request,omitempty"`

	// Response holds details of the HTTP response relating to the
	// transaction or error, if relevant.
	Response *Response `json:"response,omitempty"`

	// User holds details of the authenticated user relating to the
	// transaction or error, if relevant.
	User *User `json:"user,omitempty"`

	// Tags holds user-defined key/value pairs.
	Tags IfaceMap `json:"tags,omitempty"`

	// Service holds values to overrides service-level metadata.
	Service *Service `json:"service,omitempty"`
}

// User holds information about an authenticated user.
type User struct {
	// Username holds the username of the user.
	Username string `json:"username,omitempty"`

	// ID identifies the user, e.g. a primary key. This may be
	// a string or number.
	ID string `json:"id,omitempty"`

	// Email holds the email address of the user.
	Email string `json:"email,omitempty"`
}

// Error represents an error occurring in the service.
type Error struct {
	// Timestamp holds the time at which the error occurred.
	Timestamp Time `json:"timestamp"`

	// ID holds the 128-bit hex-encoded error ID.
	ID TraceID `json:"id"`

	// TraceID holds the ID of the trace within which the error occurred.
	TraceID TraceID `json:"trace_id,omitempty"`

	// ParentID holds the ID of the transaction within which the error
	// occurred.
	ParentID SpanID `json:"parent_id,omitempty"`

	// TransactionID holds the ID of the transaction within which the error occurred.
	TransactionID SpanID `json:"transaction_id,omitempty"`

	// Culprit holds the name of the function which
	// produced the error.
	Culprit string `json:"culprit,omitempty"`

	// Context holds contextual information relating to the error.
	Context *Context `json:"context,omitempty"`

	// Exception holds details of the exception (error or panic)
	// to which this error relates.
	Exception Exception `json:"exception,omitempty"`

	// Log holds additional information added when logging the error.
	Log Log `json:"log,omitempty"`

	// Transaction holds information about the transaction within which the error occurred.
	Transaction ErrorTransaction `json:"transaction,omitempty"`
}

// ErrorTransaction holds information about the transaction within which an error occurred.
type ErrorTransaction struct {
	// Sampled indicates that the transaction was sampled.
	Sampled *bool `json:"sampled,omitempty"`

	// Type holds the transaction type.
	Type string `json:"type,omitempty"`
}

// Exception represents an exception: an error or panic.
type Exception struct {
	// Message holds the error message.
	Message string `json:"message"`

	// Code holds the error code. This may be a number or a string.
	Code ExceptionCode `json:"code,omitempty"`

	// Type holds the type of the exception.
	Type string `json:"type,omitempty"`

	// Module holds the exception type's module namespace.
	Module string `json:"module,omitempty"`

	// Attributes holds arbitrary exception-type specific attributes.
	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// Stacktrace holds stack frames corresponding to the exception.
	Stacktrace []StacktraceFrame `json:"stacktrace,omitempty"`

	// Handled indicates whether or not the error was caught and handled.
	Handled bool `json:"handled"`

	// Cause holds the causes of this error.
	Cause []Exception `json:"cause,omitempty"`
}

// ExceptionCode represents an exception code as either a number or a string.
type ExceptionCode struct {
	String string
	Number float64
}

// StacktraceFrame describes a stack frame.
type StacktraceFrame struct {
	// AbsolutePath holds the absolute path of the source file for the
	// stack frame.
	AbsolutePath string `json:"abs_path,omitempty"`

	// File holds the base filename of the source file for the stack frame.
	File string `json:"filename"`

	// Line holds the line number of the source for the stack frame.
	Line int `json:"lineno"`

	// Column holds the column number of the source for the stack frame.
	Column *int `json:"colno,omitempty"`

	// Module holds the module to which the frame belongs. For Go, we
	// use the package path (e.g. "net/http").
	Module string `json:"module,omitempty"`

	// Classname holds the name of the class to which the frame belongs.
	Classname string `json:"classname,omitempty"`

	// Function holds the name of the function to which the frame belongs.
	Function string `json:"function,omitempty"`

	// LibraryFrame indicates whether or not the frame corresponds to
	// library or user code.
	LibraryFrame bool `json:"library_frame,omitempty"`

	// ContextLine holds the line of source code to which the frame
	// corresponds.
	ContextLine string `json:"context_line,omitempty"`

	// PreContext holds zero or more lines of source code preceding the
	// line corresponding to the frame.
	PreContext []string `json:"pre_context,omitempty"`

	// PostContext holds zero or more lines of source code proceeding the
	// line corresponding to the frame.
	PostContext []string `json:"post_context,omitempty"`

	// Vars holds local variables for this stack frame.
	Vars map[string]interface{} `json:"vars,omitempty"`
}

// Log holds additional information added when logging an error.
type Log struct {
	// Message holds the logged error message.
	Message string `json:"message"`

	// Level holds the severity of the log record.
	Level string `json:"level,omitempty"`

	// LoggerName holds the name of the logger used.
	LoggerName string `json:"logger_name,omitempty"`

	// ParamMessage holds a parameterized message,  e.g.
	// "Could not connect to %s". The string is not interpreted,
	// but may be used for grouping errors.
	ParamMessage string `json:"param_message,omitempty"`

	// Stacktrace holds stack frames corresponding to the error.
	Stacktrace []StacktraceFrame `json:"stacktrace,omitempty"`
}

// Request represents an HTTP request.
type Request struct {
	// URL is the request URL.
	URL URL `json:"url"`

	// Method holds the HTTP request method.
	Method string `json:"method"`

	// Headers holds the request headers.
	Headers Headers `json:"headers,omitempty"`

	// Body holds the request body, if body capture is enabled.
	Body *RequestBody `json:"body,omitempty"`

	// HTTPVersion holds the HTTP version of the request.
	HTTPVersion string `json:"http_version,omitempty"`

	// Cookies holds the parsed cookies.
	Cookies Cookies `json:"cookies,omitempty"`

	// Env holds environment information passed from the
	// web framework to the request handler.
	Env map[string]string `json:"env,omitempty"`

	// Socket holds transport-level information.
	Socket *RequestSocket `json:"socket,omitempty"`
}

// Cookies holds a collection of HTTP cookies.
type Cookies []*http.Cookie

// RequestBody holds a request body.
//
// Exactly one of Raw or Form must be set.
type RequestBody struct {
	// Raw holds the raw body content.
	Raw string

	// Form holds the form data from POST, PATCH, or PUT body parameters.
	Form url.Values
}

// Headers holds a collection of HTTP headers.
type Headers []Header

// Header holds an HTTP header, with one or more values.
type Header struct {
	Key    string
	Values []string
}

// RequestSocket holds transport-level information relating to an HTTP request.
type RequestSocket struct {
	// Encrypted indicates whether or not the request was sent
	// as an SSL/HTTPS request.
	Encrypted bool `json:"encrypted,omitempty"`

	// RemoteAddress holds the remote address for the request.
	RemoteAddress string `json:"remote_address,omitempty"`
}

// URL represents a server-side (transaction) request URL,
// broken down into its constituent parts.
type URL struct {
	// Full is the full URL, e.g.
	// "https://example.com:443/search/?q=elasticsearch#top".
	Full string `json:"full,omitempty"`

	// Protocol is the scheme of the URL, e.g. "https".
	Protocol string `json:"protocol,omitempty"`

	// Hostname is the hostname for the URL, e.g. "example.com".
	Hostname string `json:"hostname,omitempty"`

	// Port is the port number in the URL, e.g. "443".
	Port string `json:"port,omitempty"`

	// Path is the path of the URL, e.g. "/search".
	Path string `json:"pathname,omitempty"`

	// Search is the query string of the URL, e.g. "q=elasticsearch".
	Search string `json:"search,omitempty"`

	// Hash is the fragment for references, e.g. "top" in the
	// URL example provided for Full.
	Hash string `json:"hash,omitempty"`
}

// Response represents an HTTP response.
type Response struct {
	// StatusCode holds the HTTP response status code.
	StatusCode int `json:"status_code,omitempty"`

	// Headers holds the response headers.
	Headers Headers `json:"headers,omitempty"`

	// HeadersSent indicates whether or not headers were sent
	// to the client.
	HeadersSent *bool `json:"headers_sent,omitempty"`

	// Finished indicates whether or not the response was finished.
	Finished *bool `json:"finished,omitempty"`
}

// Time is a timestamp, formatted as a number of microseconds since January 1, 1970 UTC.
type Time time.Time

// TraceID holds a 128-bit trace ID.
type TraceID [16]byte

// SpanID holds a 64-bit span ID. Despite its name, this is used for
// both spans and transactions.
type SpanID [8]byte

// Metrics holds a set of metric samples, with an optional set of labels.
type Metrics struct {
	// Timestamp holds the time at which the metric samples were taken.
	Timestamp Time `json:"timestamp"`

	// Transaction optionally holds the name and type of transactions
	// with which these metrics are associated.
	Transaction MetricsTransaction `json:"transaction,omitempty"`

	// Span optionally holds the type and subtype of the spans with
	// which these metrics are associated.
	Span MetricsSpan `json:"span,omitempty"`

	// Labels holds a set of labels associated with the metrics.
	// The labels apply uniformly to all metric samples in the set.
	//
	// NOTE(axw) the schema calls the field "tags", but we use
	// "labels" for agent-internal consistency. Labels aligns better
	// with the common schema, anyway.
	Labels StringMap `json:"tags,omitempty"`

	// Samples holds a map of metric samples, keyed by metric name.
	Samples map[string]Metric `json:"samples"`
}

// MetricsTransaction holds transaction identifiers for metrics.
type MetricsTransaction struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// MetricsSpan holds span identifiers for metrics.
type MetricsSpan struct {
	Type    string `json:"type,omitempty"`
	Subtype string `json:"subtype,omitempty"`
}

// Metric holds metric values.
type Metric struct {
	// Value holds the metric value.
	Value float64 `json:"value"`
}
