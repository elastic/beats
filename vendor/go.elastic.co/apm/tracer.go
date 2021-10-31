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

package apm // import "go.elastic.co/apm"

import (
	"bytes"
	"compress/zlib"
	"context"
	"io"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.elastic.co/apm/apmconfig"
	"go.elastic.co/apm/internal/apmlog"
	"go.elastic.co/apm/internal/configutil"
	"go.elastic.co/apm/internal/iochan"
	"go.elastic.co/apm/internal/ringbuffer"
	"go.elastic.co/apm/internal/wildcard"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
	"go.elastic.co/apm/transport"
	"go.elastic.co/fastjson"
)

const (
	defaultPreContext     = 3
	defaultPostContext    = 3
	gracePeriodJitter     = 0.1 // +/- 10%
	tracerEventChannelCap = 1000
)

var (
	// DefaultTracer is the default global Tracer, set at package
	// initialization time, configured via environment variables.
	//
	// This will always be initialized to a non-nil value. If any
	// of the environment variables are invalid, the corresponding
	// errors will be logged to stderr and the default values will
	// be used instead.
	DefaultTracer *Tracer
)

func init() {
	var opts TracerOptions
	opts.initDefaults(true)
	DefaultTracer = newTracer(opts)
}

// TracerOptions holds initial tracer options, for passing to NewTracerOptions.
type TracerOptions struct {
	// ServiceName holds the service name.
	//
	// If ServiceName is empty, the service name will be defined using the
	// ELASTIC_APM_SERVICE_NAME environment variable, or if that is not set,
	// the executable name.
	ServiceName string

	// ServiceVersion holds the service version.
	//
	// If ServiceVersion is empty, the service version will be defined using
	// the ELASTIC_APM_SERVICE_VERSION environment variable.
	ServiceVersion string

	// ServiceEnvironment holds the service environment.
	//
	// If ServiceEnvironment is empty, the service environment will be defined
	// using the ELASTIC_APM_ENVIRONMENT environment variable.
	ServiceEnvironment string

	// Transport holds the transport to use for sending events.
	//
	// If Transport is nil, transport.Default will be used.
	//
	// If Transport implements apmconfig.Watcher, the tracer will begin watching
	// for remote changes immediately. This behaviour can be disabled by setting
	// the environment variable ELASTIC_APM_CENTRAL_CONFIG=false.
	Transport transport.Transport

	requestDuration       time.Duration
	metricsInterval       time.Duration
	maxSpans              int
	requestSize           int
	bufferSize            int
	metricsBufferSize     int
	sampler               Sampler
	sanitizedFieldNames   wildcard.Matchers
	disabledMetrics       wildcard.Matchers
	ignoreTransactionURLs wildcard.Matchers
	captureHeaders        bool
	captureBody           CaptureBodyMode
	spanFramesMinDuration time.Duration
	stackTraceLimit       int
	active                bool
	recording             bool
	configWatcher         apmconfig.Watcher
	breakdownMetrics      bool
	propagateLegacyHeader bool
	profileSender         profileSender
	cpuProfileInterval    time.Duration
	cpuProfileDuration    time.Duration
	heapProfileInterval   time.Duration
}

// initDefaults updates opts with default values.
func (opts *TracerOptions) initDefaults(continueOnError bool) error {
	var errs []error
	failed := func(err error) bool {
		if err == nil {
			return false
		}
		errs = append(errs, err)
		return true
	}

	requestDuration, err := initialRequestDuration()
	if failed(err) {
		requestDuration = defaultAPIRequestTime
	}

	metricsInterval, err := initialMetricsInterval()
	if err != nil {
		metricsInterval = defaultMetricsInterval
		errs = append(errs, err)
	}

	requestSize, err := initialAPIRequestSize()
	if err != nil {
		requestSize = int(defaultAPIRequestSize)
		errs = append(errs, err)
	}

	bufferSize, err := initialAPIBufferSize()
	if err != nil {
		bufferSize = int(defaultAPIBufferSize)
		errs = append(errs, err)
	}

	metricsBufferSize, err := initialMetricsBufferSize()
	if err != nil {
		metricsBufferSize = int(defaultMetricsBufferSize)
		errs = append(errs, err)
	}

	maxSpans, err := initialMaxSpans()
	if failed(err) {
		maxSpans = defaultMaxSpans
	}

	sampler, err := initialSampler()
	if failed(err) {
		sampler = nil
	}

	captureHeaders, err := initialCaptureHeaders()
	if failed(err) {
		captureHeaders = defaultCaptureHeaders
	}

	captureBody, err := initialCaptureBody()
	if failed(err) {
		captureBody = CaptureBodyOff
	}

	spanFramesMinDuration, err := initialSpanFramesMinDuration()
	if failed(err) {
		spanFramesMinDuration = defaultSpanFramesMinDuration
	}

	stackTraceLimit, err := initialStackTraceLimit()
	if failed(err) {
		stackTraceLimit = defaultStackTraceLimit
	}

	active, err := initialActive()
	if failed(err) {
		active = true
	}

	recording, err := initialRecording()
	if failed(err) {
		recording = true
	}

	centralConfigEnabled, err := initialCentralConfigEnabled()
	if failed(err) {
		centralConfigEnabled = true
	}

	breakdownMetricsEnabled, err := initialBreakdownMetricsEnabled()
	if failed(err) {
		breakdownMetricsEnabled = true
	}

	propagateLegacyHeader, err := initialUseElasticTraceparentHeader()
	if failed(err) {
		propagateLegacyHeader = true
	}

	cpuProfileInterval, cpuProfileDuration, err := initialCPUProfileIntervalDuration()
	if failed(err) {
		cpuProfileInterval = 0
		cpuProfileDuration = 0
	}
	heapProfileInterval, err := initialHeapProfileInterval()
	if failed(err) {
		heapProfileInterval = 0
	}

	if opts.ServiceName != "" {
		err := validateServiceName(opts.ServiceName)
		if failed(err) {
			opts.ServiceName = ""
		}
	}

	if len(errs) != 0 && !continueOnError {
		return errs[0]
	}
	for _, err := range errs {
		log.Printf("[apm]: %s", err)
	}

	opts.requestDuration = requestDuration
	opts.metricsInterval = metricsInterval
	opts.requestSize = requestSize
	opts.bufferSize = bufferSize
	opts.metricsBufferSize = metricsBufferSize
	opts.maxSpans = maxSpans
	opts.sampler = sampler
	opts.sanitizedFieldNames = initialSanitizedFieldNames()
	opts.disabledMetrics = initialDisabledMetrics()
	opts.ignoreTransactionURLs = initialIgnoreTransactionURLs()
	opts.breakdownMetrics = breakdownMetricsEnabled
	opts.captureHeaders = captureHeaders
	opts.captureBody = captureBody
	opts.spanFramesMinDuration = spanFramesMinDuration
	opts.stackTraceLimit = stackTraceLimit
	opts.active = active
	opts.recording = recording
	opts.propagateLegacyHeader = propagateLegacyHeader
	if opts.Transport == nil {
		opts.Transport = transport.Default
	}
	if centralConfigEnabled {
		if cw, ok := opts.Transport.(apmconfig.Watcher); ok {
			opts.configWatcher = cw
		}
	}
	if ps, ok := opts.Transport.(profileSender); ok {
		opts.profileSender = ps
		opts.cpuProfileInterval = cpuProfileInterval
		opts.cpuProfileDuration = cpuProfileDuration
		opts.heapProfileInterval = heapProfileInterval
	}

	serviceName, serviceVersion, serviceEnvironment := initialService()
	if opts.ServiceName == "" {
		opts.ServiceName = serviceName
	}
	if opts.ServiceVersion == "" {
		opts.ServiceVersion = serviceVersion
	}
	if opts.ServiceEnvironment == "" {
		opts.ServiceEnvironment = serviceEnvironment
	}
	return nil
}

// Tracer manages the sampling and sending of transactions to
// Elastic APM.
//
// Transactions are buffered until they are flushed (forcibly
// with a Flush call, or when the flush timer expires), or when
// the maximum transaction queue size is reached. Failure to
// send will be periodically retried. Once the queue limit has
// been reached, new transactions will replace older ones in
// the queue.
//
// Errors are sent as soon as possible, but will buffered and
// later sent in bulk if the tracer is busy, or otherwise cannot
// send to the server, e.g. due to network failure. There is
// a limit to the number of errors that will be buffered, and
// once that limit has been reached, new errors will be dropped
// until the queue is drained.
//
// The exported fields be altered or replaced any time up until
// any Tracer methods have been invoked.
type Tracer struct {
	Transport transport.Transport
	Service   struct {
		Name        string
		Version     string
		Environment string
	}

	process *model.Process
	system  *model.System

	active            int32
	bufferSize        int
	metricsBufferSize int
	closing           chan struct{}
	closed            chan struct{}
	forceFlush        chan chan<- struct{}
	forceSendMetrics  chan chan<- struct{}
	configCommands    chan tracerConfigCommand
	configWatcher     chan apmconfig.Watcher
	events            chan tracerEvent
	breakdownMetrics  *breakdownMetrics
	profileSender     profileSender

	statsMu sync.Mutex
	stats   TracerStats

	// instrumentationConfig_ must only be accessed and mutated
	// using Tracer.instrumentationConfig() and Tracer.setInstrumentationConfig().
	instrumentationConfigInternal *instrumentationConfig

	errorDataPool       sync.Pool
	spanDataPool        sync.Pool
	transactionDataPool sync.Pool
}

// NewTracer returns a new Tracer, using the default transport,
// and with the specified service name and version if specified.
// This is equivalent to calling NewTracerOptions with a
// TracerOptions having ServiceName and ServiceVersion set to
// the provided arguments.
//
// NOTE when this package is imported, DefaultTracer is initialised
// using environment variables for configuration. When creating a
// tracer with NewTracer or NewTracerOptions, you should close
// apm.DefaultTracer if it is not needed, e.g. by calling
// apm.DefaultTracer.Close() in an init function.
func NewTracer(serviceName, serviceVersion string) (*Tracer, error) {
	return NewTracerOptions(TracerOptions{
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
	})
}

// NewTracerOptions returns a new Tracer using the provided options.
// See TracerOptions for details on the options, and their default
// values.
//
// NOTE when this package is imported, DefaultTracer is initialised
// using environment variables for configuration. When creating a
// tracer with NewTracer or NewTracerOptions, you should close
// apm.DefaultTracer if it is not needed, e.g. by calling
// apm.DefaultTracer.Close() in an init function.
func NewTracerOptions(opts TracerOptions) (*Tracer, error) {
	if err := opts.initDefaults(false); err != nil {
		return nil, err
	}
	return newTracer(opts), nil
}

func newTracer(opts TracerOptions) *Tracer {
	t := &Tracer{
		Transport:         opts.Transport,
		process:           &currentProcess,
		system:            &localSystem,
		closing:           make(chan struct{}),
		closed:            make(chan struct{}),
		forceFlush:        make(chan chan<- struct{}),
		forceSendMetrics:  make(chan chan<- struct{}),
		configCommands:    make(chan tracerConfigCommand),
		configWatcher:     make(chan apmconfig.Watcher),
		events:            make(chan tracerEvent, tracerEventChannelCap),
		active:            1,
		breakdownMetrics:  newBreakdownMetrics(),
		bufferSize:        opts.bufferSize,
		metricsBufferSize: opts.metricsBufferSize,
		profileSender:     opts.profileSender,
		instrumentationConfigInternal: &instrumentationConfig{
			local: make(map[string]func(*instrumentationConfigValues)),
		},
	}
	t.Service.Name = opts.ServiceName
	t.Service.Version = opts.ServiceVersion
	t.Service.Environment = opts.ServiceEnvironment
	t.breakdownMetrics.enabled = opts.breakdownMetrics

	// Initialise local transaction config.
	t.setLocalInstrumentationConfig(envRecording, func(cfg *instrumentationConfigValues) {
		cfg.recording = opts.recording
	})
	t.setLocalInstrumentationConfig(envCaptureBody, func(cfg *instrumentationConfigValues) {
		cfg.captureBody = opts.captureBody
	})
	t.setLocalInstrumentationConfig(envCaptureHeaders, func(cfg *instrumentationConfigValues) {
		cfg.captureHeaders = opts.captureHeaders
	})
	t.setLocalInstrumentationConfig(envMaxSpans, func(cfg *instrumentationConfigValues) {
		cfg.maxSpans = opts.maxSpans
	})
	t.setLocalInstrumentationConfig(envTransactionSampleRate, func(cfg *instrumentationConfigValues) {
		cfg.sampler = opts.sampler
		cfg.extendedSampler, _ = opts.sampler.(ExtendedSampler)
	})
	t.setLocalInstrumentationConfig(envSpanFramesMinDuration, func(cfg *instrumentationConfigValues) {
		cfg.spanFramesMinDuration = opts.spanFramesMinDuration
	})
	t.setLocalInstrumentationConfig(envStackTraceLimit, func(cfg *instrumentationConfigValues) {
		cfg.stackTraceLimit = opts.stackTraceLimit
	})
	t.setLocalInstrumentationConfig(envUseElasticTraceparentHeader, func(cfg *instrumentationConfigValues) {
		cfg.propagateLegacyHeader = opts.propagateLegacyHeader
	})
	t.setLocalInstrumentationConfig(envSanitizeFieldNames, func(cfg *instrumentationConfigValues) {
		cfg.sanitizedFieldNames = opts.sanitizedFieldNames
	})
	t.setLocalInstrumentationConfig(envIgnoreURLs, func(cfg *instrumentationConfigValues) {
		cfg.ignoreTransactionURLs = opts.ignoreTransactionURLs
	})
	if apmlog.DefaultLogger != nil {
		defaultLogLevel := apmlog.DefaultLogger.Level()
		t.setLocalInstrumentationConfig(apmlog.EnvLogLevel, func(cfg *instrumentationConfigValues) {
			// Revert to the original, local, log level when
			// the centrally defined log level is removed.
			apmlog.DefaultLogger.SetLevel(defaultLogLevel)
		})
	}

	if !opts.active {
		t.active = 0
		close(t.closed)
		return t
	}

	go t.loop()
	t.configCommands <- func(cfg *tracerConfig) {
		cfg.recording = opts.recording
		cfg.cpuProfileInterval = opts.cpuProfileInterval
		cfg.cpuProfileDuration = opts.cpuProfileDuration
		cfg.heapProfileInterval = opts.heapProfileInterval
		cfg.metricsInterval = opts.metricsInterval
		cfg.requestDuration = opts.requestDuration
		cfg.requestSize = opts.requestSize
		cfg.disabledMetrics = opts.disabledMetrics
		cfg.preContext = defaultPreContext
		cfg.postContext = defaultPostContext
		cfg.metricsGatherers = []MetricsGatherer{newBuiltinMetricsGatherer(t)}
		if apmlog.DefaultLogger != nil {
			cfg.logger = apmlog.DefaultLogger
		}
	}
	if opts.configWatcher != nil {
		t.configWatcher <- opts.configWatcher
	}
	return t
}

// tracerConfig holds the tracer's runtime configuration, which may be modified
// by sending a tracerConfigCommand to the tracer's configCommands channel.
type tracerConfig struct {
	recording               bool
	requestSize             int
	requestDuration         time.Duration
	metricsInterval         time.Duration
	logger                  WarningLogger
	metricsGatherers        []MetricsGatherer
	contextSetter           stacktrace.ContextSetter
	preContext, postContext int
	disabledMetrics         wildcard.Matchers
	cpuProfileDuration      time.Duration
	cpuProfileInterval      time.Duration
	heapProfileInterval     time.Duration
}

type tracerConfigCommand func(*tracerConfig)

// Close closes the Tracer, preventing transactions from being
// sent to the APM server.
func (t *Tracer) Close() {
	select {
	case <-t.closing:
	default:
		close(t.closing)
	}
	<-t.closed
}

// Flush waits for the Tracer to flush any transactions and errors it currently
// has queued to the APM server, the tracer is stopped, or the abort channel
// is signaled.
func (t *Tracer) Flush(abort <-chan struct{}) {
	flushed := make(chan struct{}, 1)
	select {
	case t.forceFlush <- flushed:
		select {
		case <-abort:
		case <-flushed:
		case <-t.closed:
		}
	case <-t.closed:
	}
}

// Recording reports whether the tracer is recording events. Instrumentation
// may use this to avoid creating transactions, spans, and metrics when the
// tracer is configured to not record.
//
// Recording will also return false if the tracer is inactive.
func (t *Tracer) Recording() bool {
	return t.instrumentationConfig().recording && t.Active()
}

// Active reports whether the tracer is active. If the tracer is inactive,
// no transactions or errors will be sent to the Elastic APM server.
func (t *Tracer) Active() bool {
	return atomic.LoadInt32(&t.active) == 1
}

// ShouldPropagateLegacyHeader reports whether instrumentation should
// propagate the legacy "Elastic-Apm-Traceparent" header in addition to
// the standard W3C "traceparent" header.
//
// This method will be removed in a future major version when we remove
// support for propagating the legacy header.
func (t *Tracer) ShouldPropagateLegacyHeader() bool {
	return t.instrumentationConfig().propagateLegacyHeader
}

// SetRequestDuration sets the maximum amount of time to keep a request open
// to the APM server for streaming data before closing the stream and starting
// a new request.
func (t *Tracer) SetRequestDuration(d time.Duration) {
	t.sendConfigCommand(func(cfg *tracerConfig) {
		cfg.requestDuration = d
	})
}

// SetMetricsInterval sets the metrics interval -- the amount of time in
// between metrics samples being gathered.
func (t *Tracer) SetMetricsInterval(d time.Duration) {
	t.sendConfigCommand(func(cfg *tracerConfig) {
		cfg.metricsInterval = d
	})
}

// SetContextSetter sets the stacktrace.ContextSetter to be used for
// setting stacktrace source context. If nil (which is the initial
// value), no context will be set.
func (t *Tracer) SetContextSetter(setter stacktrace.ContextSetter) {
	t.sendConfigCommand(func(cfg *tracerConfig) {
		cfg.contextSetter = setter
	})
}

// SetLogger sets the Logger to be used for logging the operation of
// the tracer.
//
// If logger implements WarningLogger, its Warningf method will be used
// for logging warnings. Otherwise, warnings will logged using Debugf.
//
// The tracer is initialized with a default logger configured with the
// environment variables ELASTIC_APM_LOG_FILE and ELASTIC_APM_LOG_LEVEL.
// Calling SetLogger will replace the default logger.
func (t *Tracer) SetLogger(logger Logger) {
	t.sendConfigCommand(func(cfg *tracerConfig) {
		cfg.logger = makeWarningLogger(logger)
	})
}

// SetSanitizedFieldNames sets the wildcard patterns that will be used to
// match cookie and form field names for sanitization. Fields matching any
// of the the supplied patterns will have their values redacted. If
// SetSanitizedFieldNames is called with no arguments, then no fields
// will be redacted.
//
// Configuration via Kibana takes precedence over local configuration, so
// if sanitized_field_names has been configured via Kibana, this call will
// not have any effect until/unless that configuration has been removed.
func (t *Tracer) SetSanitizedFieldNames(patterns ...string) error {
	var matchers wildcard.Matchers
	if len(patterns) != 0 {
		matchers = make(wildcard.Matchers, len(patterns))
		for i, p := range patterns {
			matchers[i] = configutil.ParseWildcardPattern(p)
		}
	}
	t.setLocalInstrumentationConfig(envSanitizeFieldNames, func(cfg *instrumentationConfigValues) {
		cfg.sanitizedFieldNames = matchers
	})
	return nil
}

// SetIgnoreTransactionURLs sets the wildcard patterns that will be used to
// ignore transactions with matching URLs.
func (t *Tracer) SetIgnoreTransactionURLs(pattern string) error {
	t.setLocalInstrumentationConfig(envIgnoreURLs, func(cfg *instrumentationConfigValues) {
		cfg.ignoreTransactionURLs = configutil.ParseWildcardPatterns(pattern)
	})
	return nil
}

// RegisterMetricsGatherer registers g for periodic (or forced) metrics
// gathering by t.
//
// RegisterMetricsGatherer returns a function which will deregister g.
// It may safely be called multiple times.
func (t *Tracer) RegisterMetricsGatherer(g MetricsGatherer) func() {
	// Wrap g in a pointer-to-struct, so we can safely compare.
	wrapped := &struct{ MetricsGatherer }{MetricsGatherer: g}
	t.sendConfigCommand(func(cfg *tracerConfig) {
		cfg.metricsGatherers = append(cfg.metricsGatherers, wrapped)
	})
	deregister := func(cfg *tracerConfig) {
		for i, g := range cfg.metricsGatherers {
			if g != wrapped {
				continue
			}
			cfg.metricsGatherers = append(cfg.metricsGatherers[:i], cfg.metricsGatherers[i+1:]...)
		}
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			t.sendConfigCommand(deregister)
		})
	}
}

// SetConfigWatcher sets w as the config watcher.
//
// By default, the tracer will be configured to use the transport for
// watching config, if the transport implements apmconfig.Watcher. This
// can be overridden by calling SetConfigWatcher.
//
// If w is nil, config watching will be stopped.
//
// Calling SetConfigWatcher will discard any previously observed remote
// config, reverting to local config until a config change from w is
// observed.
func (t *Tracer) SetConfigWatcher(w apmconfig.Watcher) {
	select {
	case t.configWatcher <- w:
	case <-t.closing:
	case <-t.closed:
	}
}

func (t *Tracer) sendConfigCommand(cmd tracerConfigCommand) {
	select {
	case t.configCommands <- cmd:
	case <-t.closing:
	case <-t.closed:
	}
}

// SetRecording enables or disables recording of future events.
//
// SetRecording does not affect in-flight events.
func (t *Tracer) SetRecording(r bool) {
	t.setLocalInstrumentationConfig(envRecording, func(cfg *instrumentationConfigValues) {
		// Update instrumentation config to disable transactions and errors.
		cfg.recording = r
	})
	t.sendConfigCommand(func(cfg *tracerConfig) {
		// Consult t.instrumentationConfig() as local config may not be in effect,
		// or there may have been a concurrent change to instrumentation config.
		cfg.recording = t.instrumentationConfig().recording
	})
}

// SetSampler sets the sampler the tracer.
//
// It is valid to pass nil, in which case all transactions will be sampled.
//
// Configuration via Kibana takes precedence over local configuration, so
// if sampling has been configured via Kibana, this call will not have any
// effect until/unless that configuration has been removed.
func (t *Tracer) SetSampler(s Sampler) {
	t.setLocalInstrumentationConfig(envTransactionSampleRate, func(cfg *instrumentationConfigValues) {
		cfg.sampler = s
		cfg.extendedSampler, _ = s.(ExtendedSampler)
	})
}

// SetMaxSpans sets the maximum number of spans that will be added
// to a transaction before dropping spans.
//
// Passing in zero will disable all spans, while negative values will
// permit an unlimited number of spans.
func (t *Tracer) SetMaxSpans(n int) {
	t.setLocalInstrumentationConfig(envMaxSpans, func(cfg *instrumentationConfigValues) {
		cfg.maxSpans = n
	})
}

// SetSpanFramesMinDuration sets the minimum duration for a span after which
// we will capture its stack frames.
func (t *Tracer) SetSpanFramesMinDuration(d time.Duration) {
	t.setLocalInstrumentationConfig(envMaxSpans, func(cfg *instrumentationConfigValues) {
		cfg.spanFramesMinDuration = d
	})
}

// SetStackTraceLimit sets the the maximum number of stack frames to collect
// for each stack trace. If limit is negative, then all frames will be collected.
func (t *Tracer) SetStackTraceLimit(limit int) {
	t.setLocalInstrumentationConfig(envMaxSpans, func(cfg *instrumentationConfigValues) {
		cfg.stackTraceLimit = limit
	})
}

// SetCaptureHeaders enables or disables capturing of HTTP headers.
func (t *Tracer) SetCaptureHeaders(capture bool) {
	t.setLocalInstrumentationConfig(envMaxSpans, func(cfg *instrumentationConfigValues) {
		cfg.captureHeaders = capture
	})
}

// SetCaptureBody sets the HTTP request body capture mode.
func (t *Tracer) SetCaptureBody(mode CaptureBodyMode) {
	t.setLocalInstrumentationConfig(envMaxSpans, func(cfg *instrumentationConfigValues) {
		cfg.captureBody = mode
	})
}

// SendMetrics forces the tracer to gather and send metrics immediately,
// blocking until the metrics have been sent or the abort channel is
// signalled.
func (t *Tracer) SendMetrics(abort <-chan struct{}) {
	sent := make(chan struct{}, 1)
	select {
	case t.forceSendMetrics <- sent:
		select {
		case <-abort:
		case <-sent:
		case <-t.closed:
		}
	case <-t.closed:
	}
}

// Stats returns the current TracerStats. This will return the most
// recent values even after the tracer has been closed.
func (t *Tracer) Stats() TracerStats {
	t.statsMu.Lock()
	stats := t.stats
	t.statsMu.Unlock()
	return stats
}

func (t *Tracer) loop() {
	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()
	defer close(t.closed)
	defer atomic.StoreInt32(&t.active, 0)

	var req iochan.ReadRequest
	var requestBuf bytes.Buffer
	var metadata []byte
	var gracePeriod time.Duration = -1
	var flushed chan<- struct{}
	var requestBufTransactions, requestBufSpans, requestBufErrors, requestBufMetricsets uint64
	zlibWriter, _ := zlib.NewWriterLevel(&requestBuf, zlib.BestSpeed)
	zlibFlushed := true
	zlibClosed := false
	iochanReader := iochan.NewReader()
	requestBytesRead := 0
	requestActive := false
	closeRequest := false
	flushRequest := false
	requestResult := make(chan error, 1)
	requestTimer := time.NewTimer(0)
	requestTimerActive := false
	if !requestTimer.Stop() {
		<-requestTimer.C
	}

	// Run another goroutine to perform the blocking requests,
	// communicating with the tracer loop to obtain stream data.
	sendStreamRequest := make(chan time.Duration)
	done := make(chan struct{})
	defer func() {
		close(sendStreamRequest)
		<-done
	}()
	go func() {
		defer close(done)
		jitterRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for gracePeriod := range sendStreamRequest {
			if gracePeriod > 0 {
				select {
				case <-time.After(jitterDuration(gracePeriod, jitterRand, gracePeriodJitter)):
				case <-ctx.Done():
				}
			}
			requestResult <- t.Transport.SendStream(ctx, iochanReader)
		}
	}()

	var breakdownMetricsLimitWarningLogged bool
	var stats TracerStats
	var metrics Metrics
	var sentMetrics chan<- struct{}
	var gatheringMetrics bool
	var metricsTimerStart time.Time
	metricsBuffer := ringbuffer.New(t.metricsBufferSize)
	gatheredMetrics := make(chan struct{}, 1)
	metricsTimer := time.NewTimer(0)
	if !metricsTimer.Stop() {
		<-metricsTimer.C
	}

	var lastConfigChange map[string]string
	var configChanges <-chan apmconfig.Change
	var stopConfigWatcher func()
	defer func() {
		if stopConfigWatcher != nil {
			stopConfigWatcher()
		}
	}()

	cpuProfilingState := newCPUProfilingState(t.profileSender)
	heapProfilingState := newHeapProfilingState(t.profileSender)

	var cfg tracerConfig
	buffer := ringbuffer.New(t.bufferSize)
	buffer.Evicted = func(h ringbuffer.BlockHeader) {
		switch h.Tag {
		case errorBlockTag:
			stats.ErrorsDropped++
		case spanBlockTag:
			stats.SpansDropped++
		case transactionBlockTag:
			stats.TransactionsDropped++
		}
	}
	modelWriter := modelWriter{
		buffer:        buffer,
		metricsBuffer: metricsBuffer,
		cfg:           &cfg,
		stats:         &stats,
	}

	handleTracerConfigCommand := func(cmd tracerConfigCommand) {
		var oldMetricsInterval time.Duration
		if cfg.recording {
			oldMetricsInterval = cfg.metricsInterval
		}
		cmd(&cfg)
		var metricsInterval, cpuProfileInterval, cpuProfileDuration, heapProfileInterval time.Duration
		if cfg.recording {
			metricsInterval = cfg.metricsInterval
			cpuProfileInterval = cfg.cpuProfileInterval
			cpuProfileDuration = cfg.cpuProfileDuration
			heapProfileInterval = cfg.heapProfileInterval
		}

		cpuProfilingState.updateConfig(cpuProfileInterval, cpuProfileDuration)
		heapProfilingState.updateConfig(heapProfileInterval, 0)
		if !gatheringMetrics && metricsInterval != oldMetricsInterval {
			if metricsTimerStart.IsZero() {
				if metricsInterval > 0 {
					metricsTimer.Reset(metricsInterval)
					metricsTimerStart = time.Now()
				}
			} else {
				if metricsInterval <= 0 {
					metricsTimerStart = time.Time{}
					if !metricsTimer.Stop() {
						<-metricsTimer.C
					}
				} else {
					alreadyPassed := time.Since(metricsTimerStart)
					if alreadyPassed >= metricsInterval {
						metricsTimer.Reset(0)
					} else {
						metricsTimer.Reset(metricsInterval - alreadyPassed)
					}
				}
			}
		}
	}

	for {
		var gatherMetrics bool
		select {
		case <-t.closing:
			cancelContext() // informs transport that EOF is expected
			iochanReader.CloseRead(io.EOF)
			return
		case cmd := <-t.configCommands:
			handleTracerConfigCommand(cmd)
			continue
		case cw := <-t.configWatcher:
			if configChanges != nil {
				stopConfigWatcher()
				t.updateRemoteConfig(cfg.logger, lastConfigChange, nil)
				lastConfigChange = nil
				configChanges = nil
			}
			if cw == nil {
				continue
			}
			var configWatcherContext context.Context
			var watchParams apmconfig.WatchParams
			watchParams.Service.Name = t.Service.Name
			watchParams.Service.Environment = t.Service.Environment
			configWatcherContext, stopConfigWatcher = context.WithCancel(ctx)
			configChanges = cw.WatchConfig(configWatcherContext, watchParams)
			// Silence go vet's "possible context leak" false positive.
			// We call a previous stopConfigWatcher before reassigning
			// the variable, and we have a defer at the top level of the
			// loop method that will call the final stopConfigWatcher
			// value on method exit.
			_ = stopConfigWatcher
			continue
		case change, ok := <-configChanges:
			if !ok {
				configChanges = nil
				continue
			}
			if change.Err != nil {
				if cfg.logger != nil {
					cfg.logger.Errorf("config request failed: %s", change.Err)
				}
			} else {
				t.updateRemoteConfig(cfg.logger, lastConfigChange, change.Attrs)
				lastConfigChange = change.Attrs
				handleTracerConfigCommand(func(cfg *tracerConfig) {
					cfg.recording = t.instrumentationConfig().recording
				})
			}
			continue
		case event := <-t.events:
			switch event.eventType {
			case transactionEvent:
				if !t.breakdownMetrics.recordTransaction(event.tx.TransactionData) {
					if !breakdownMetricsLimitWarningLogged && cfg.logger != nil {
						cfg.logger.Warningf("%s", breakdownMetricsLimitWarning)
						breakdownMetricsLimitWarningLogged = true
					}
				}
				modelWriter.writeTransaction(event.tx.Transaction, event.tx.TransactionData)
			case spanEvent:
				modelWriter.writeSpan(event.span.Span, event.span.SpanData)
			case errorEvent:
				modelWriter.writeError(event.err)
				// Flush the buffer to transmit the error immediately.
				flushRequest = true
			}
		case <-requestTimer.C:
			requestTimerActive = false
			closeRequest = true
		case <-metricsTimer.C:
			metricsTimerStart = time.Time{}
			gatherMetrics = !gatheringMetrics
		case sentMetrics = <-t.forceSendMetrics:
			if cfg.recording {
				if !metricsTimerStart.IsZero() {
					if !metricsTimer.Stop() {
						<-metricsTimer.C
					}
					metricsTimerStart = time.Time{}
				}
				gatherMetrics = !gatheringMetrics
			}
		case <-gatheredMetrics:
			modelWriter.writeMetrics(&metrics)
			gatheringMetrics = false
			flushRequest = true
			if cfg.recording && cfg.metricsInterval > 0 {
				metricsTimerStart = time.Now()
				metricsTimer.Reset(cfg.metricsInterval)
			}
		case <-cpuProfilingState.timer.C:
			cpuProfilingState.start(ctx, cfg.logger, t.metadataReader())
		case <-cpuProfilingState.finished:
			cpuProfilingState.resetTimer()
		case <-heapProfilingState.timer.C:
			heapProfilingState.start(ctx, cfg.logger, t.metadataReader())
		case <-heapProfilingState.finished:
			heapProfilingState.resetTimer()
		case flushed = <-t.forceFlush:
			// Drain any objects buffered in the channels.
			for n := len(t.events); n > 0; n-- {
				event := <-t.events
				switch event.eventType {
				case transactionEvent:
					if !t.breakdownMetrics.recordTransaction(event.tx.TransactionData) {
						if !breakdownMetricsLimitWarningLogged && cfg.logger != nil {
							cfg.logger.Warningf("%s", breakdownMetricsLimitWarning)
							breakdownMetricsLimitWarningLogged = true
						}
					}
					modelWriter.writeTransaction(event.tx.Transaction, event.tx.TransactionData)
				case spanEvent:
					modelWriter.writeSpan(event.span.Span, event.span.SpanData)
				case errorEvent:
					modelWriter.writeError(event.err)
				}
			}
			if !requestActive && buffer.Len() == 0 && metricsBuffer.Len() == 0 {
				flushed <- struct{}{}
				continue
			}
			closeRequest = true
		case req = <-iochanReader.C:
		case err := <-requestResult:
			if err != nil {
				stats.Errors.SendStream++
				gracePeriod = nextGracePeriod(gracePeriod)
				if cfg.logger != nil {
					logf := cfg.logger.Debugf
					if err, ok := err.(*transport.HTTPError); ok && err.Response.StatusCode == 404 {
						// 404 typically means the server is too old, meaning
						// the error is due to a misconfigured environment.
						logf = cfg.logger.Errorf
					}
					logf("request failed: %s (next request in ~%s)", err, gracePeriod)
				}
			} else {
				gracePeriod = -1 // Reset grace period after success.
				stats.TransactionsSent += requestBufTransactions
				stats.SpansSent += requestBufSpans
				stats.ErrorsSent += requestBufErrors
				if cfg.logger != nil {
					s := func(n uint64) string {
						if n != 1 {
							return "s"
						}
						return ""
					}
					cfg.logger.Debugf(
						"sent request with %d transaction%s, %d span%s, %d error%s, %d metricset%s",
						requestBufTransactions, s(requestBufTransactions),
						requestBufSpans, s(requestBufSpans),
						requestBufErrors, s(requestBufErrors),
						requestBufMetricsets, s(requestBufMetricsets),
					)
				}
			}
			if !stats.isZero() {
				t.statsMu.Lock()
				t.stats.accumulate(stats)
				t.statsMu.Unlock()
				stats = TracerStats{}
			}
			if sentMetrics != nil && requestBufMetricsets > 0 {
				sentMetrics <- struct{}{}
				sentMetrics = nil
			}
			if flushed != nil {
				flushed <- struct{}{}
				flushed = nil
			}
			if req.Buf != nil {
				// req will be canceled by CloseRead below.
				req.Buf = nil
			}
			iochanReader.CloseRead(io.EOF)
			iochanReader = iochan.NewReader()
			flushRequest = false
			closeRequest = false
			requestActive = false
			requestBytesRead = 0
			requestBuf.Reset()
			requestBufTransactions = 0
			requestBufSpans = 0
			requestBufErrors = 0
			requestBufMetricsets = 0
			if requestTimerActive {
				if !requestTimer.Stop() {
					<-requestTimer.C
				}
				requestTimerActive = false
			}
		}

		if !stats.isZero() {
			t.statsMu.Lock()
			t.stats.accumulate(stats)
			t.statsMu.Unlock()
			stats = TracerStats{}
		}

		if gatherMetrics {
			gatheringMetrics = true
			metrics.disabled = cfg.disabledMetrics
			t.gatherMetrics(ctx, cfg.metricsGatherers, &metrics, cfg.logger, gatheredMetrics)
			if cfg.logger != nil {
				cfg.logger.Debugf("gathering metrics")
			}
		}

		if !requestActive {
			if buffer.Len() == 0 && metricsBuffer.Len() == 0 {
				continue
			}
			sendStreamRequest <- gracePeriod
			if metadata == nil {
				metadata = t.jsonRequestMetadata()
			}
			zlibWriter.Reset(&requestBuf)
			zlibWriter.Write(metadata)
			zlibFlushed = false
			zlibClosed = false
			requestActive = true
			requestTimer.Reset(cfg.requestDuration)
			requestTimerActive = true
		}

		if !closeRequest || !zlibClosed {
			for requestBytesRead+requestBuf.Len() < cfg.requestSize {
				if metricsBuffer.Len() > 0 {
					if _, _, err := metricsBuffer.WriteBlockTo(zlibWriter); err == nil {
						requestBufMetricsets++
						zlibWriter.Write([]byte("\n"))
						zlibFlushed = false
						if sentMetrics != nil {
							// SendMetrics was called: close the request
							// off so we can inform the user when the
							// metrics have been processed.
							closeRequest = true
						}
					}
					continue
				}
				if buffer.Len() == 0 {
					break
				}
				if h, _, err := buffer.WriteBlockTo(zlibWriter); err == nil {
					switch h.Tag {
					case transactionBlockTag:
						requestBufTransactions++
					case spanBlockTag:
						requestBufSpans++
					case errorBlockTag:
						requestBufErrors++
					}
					zlibWriter.Write([]byte("\n"))
					zlibFlushed = false
				}
			}
			if !closeRequest {
				closeRequest = requestBytesRead+requestBuf.Len() >= cfg.requestSize
			}
		}
		if closeRequest {
			if !zlibClosed {
				zlibWriter.Close()
				zlibClosed = true
			}
		} else if flushRequest && !zlibFlushed {
			zlibWriter.Flush()
			flushRequest = false
			zlibFlushed = true
		}

		if req.Buf == nil || requestBuf.Len() == 0 {
			continue
		}
		const zlibHeaderLen = 2
		if requestBytesRead+requestBuf.Len() > zlibHeaderLen {
			n, err := requestBuf.Read(req.Buf)
			if closeRequest && err == nil && requestBuf.Len() == 0 {
				err = io.EOF
			}
			req.Respond(n, err)
			req.Buf = nil
			if n > 0 {
				requestBytesRead += n
			}
		}
	}
}

// jsonRequestMetadata returns a JSON-encoded metadata object that features
// at the head of every request body. This is called exactly once, when the
// first request is made.
func (t *Tracer) jsonRequestMetadata() []byte {
	var json fastjson.Writer
	json.RawString(`{"metadata":`)
	t.encodeRequestMetadata(&json)
	json.RawString("}\n")
	return json.Bytes()
}

// metadataReader returns an io.Reader that holds the JSON-encoded metadata,
// suitable for including in a profile request.
func (t *Tracer) metadataReader() io.Reader {
	var metadata fastjson.Writer
	t.encodeRequestMetadata(&metadata)
	return bytes.NewReader(metadata.Bytes())
}

func (t *Tracer) encodeRequestMetadata(json *fastjson.Writer) {
	service := makeService(t.Service.Name, t.Service.Version, t.Service.Environment)
	json.RawString(`{"system":`)
	t.system.MarshalFastJSON(json)
	json.RawString(`,"process":`)
	t.process.MarshalFastJSON(json)
	json.RawString(`,"service":`)
	service.MarshalFastJSON(json)
	if cloud := getCloudMetadata(); cloud != nil {
		json.RawString(`,"cloud":`)
		cloud.MarshalFastJSON(json)
	}
	if len(globalLabels) > 0 {
		json.RawString(`,"labels":`)
		globalLabels.MarshalFastJSON(json)
	}
	json.RawByte('}')
}

// gatherMetrics gathers metrics from each of the registered
// metrics gatherers. Once all gatherers have returned, a value
// will be sent on the "gathered" channel.
func (t *Tracer) gatherMetrics(ctx context.Context, gatherers []MetricsGatherer, m *Metrics, l Logger, gathered chan<- struct{}) {
	timestamp := model.Time(time.Now().UTC())
	var group sync.WaitGroup
	for _, g := range gatherers {
		group.Add(1)
		go func(g MetricsGatherer) {
			defer group.Done()
			gatherMetrics(ctx, g, m, l)
		}(g)
	}
	go func() {
		group.Wait()
		for _, m := range m.transactionGroupMetrics {
			m.Timestamp = timestamp
		}
		for _, m := range m.metrics {
			m.Timestamp = timestamp
		}
		gathered <- struct{}{}
	}()
}

type tracerEventType int

const (
	transactionEvent tracerEventType = iota
	spanEvent
	errorEvent
)

type tracerEvent struct {
	eventType tracerEventType

	// err is set only if eventType == errorEvent.
	err *ErrorData

	// tx is set only if eventType == transactionEvent.
	tx struct {
		*Transaction
		// Transaction.TransactionData is nil at the
		// point tracerEvent is created (to signify
		// that the transaction is ended), so we pass
		// it along side.
		*TransactionData
	}

	// span is set only if eventType == spanEvent.
	span struct {
		*Span
		// Span.SpanData is nil at the point tracerEvent
		// is created (to signify that the span is ended),
		// so we pass it along side.
		*SpanData
	}
}
