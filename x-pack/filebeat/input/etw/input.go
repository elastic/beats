// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/reader/etw"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/go-concert/ctxtool"

	"github.com/rcrowley/go-metrics"
	"golang.org/x/sys/windows"
)

const (
	inputName = "etw"

	// Reconnect backoff bounds. A realtime session that is restarted by its
	// owner usually comes back within seconds, and events produced while no
	// consumer is attached are lost, so the first retries are quick. The cap
	// keeps a session that never returns from being polled too eagerly.
	reconnectInitialBackoff = time.Second
	reconnectMaxBackoff     = 30 * time.Second
)

// It abstracts the underlying operations needed to work with ETW, allowing for easier
// testing and decoupling from the Windows-specific ETW API.
type sessionOperator interface {
	newSession(config config) (*etw.Session, error)
	attachToExistingSession(session *etw.Session) error
	createRealtimeSession(session *etw.Session) error
	startConsumer(session *etw.Session) error
	stopSession(session *etw.Session) error
	resetSession(session *etw.Session)
}

type realSessionOperator struct{}

func (op *realSessionOperator) newSession(config config) (*etw.Session, error) {
	return etw.NewSession(convertConfig(config))
}

func (op *realSessionOperator) attachToExistingSession(session *etw.Session) error {
	return session.AttachToExistingSession()
}

func (op *realSessionOperator) createRealtimeSession(session *etw.Session) error {
	return session.CreateRealtimeSession()
}

func (op *realSessionOperator) startConsumer(session *etw.Session) error {
	return session.StartConsumer()
}

func (op *realSessionOperator) stopSession(session *etw.Session) error {
	return session.StopSession()
}

func (op *realSessionOperator) resetSession(session *etw.Session) {
	session.Reset()
}

// etwInput struct holds the configuration and state for the ETW input
type etwInput struct {
	log        *logp.Logger
	metrics    *inputMetrics
	config     config
	etwSession *etw.Session
	publisher  stateless.Publisher
	operator   sessionOperator
	health     *eventHealth
	// backoff paces reconnection attempts. Nil means use the package
	// defaults; tests set a short one.
	backoff backoff.Backoff
}

func Plugin() input.Plugin {
	return input.Plugin{
		Name:      inputName,
		Stability: feature.Stable,
		Info:      "Collect ETW logs.",
		Manager:   stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	return &etwInput{
		config:   conf,
		operator: &realSessionOperator{},
	}, nil
}

func (e *etwInput) Name() string { return inputName }

func (e *etwInput) Test(_ input.TestContext) error {
	return nil
}

// Run starts the ETW session and processes incoming events.
//
// For realtime sessions Run does not return when the session ends. Windows
// makes ProcessTrace return ERROR_SUCCESS when another controller stops the
// session (for example `logman stop <session> -ets`), which used to look like
// a clean exit and left the input stopped until Filebeat was restarted. Run
// now treats that as a lost session: it reports Degraded and keeps trying to
// create or attach to the session again, with backoff, until it is consuming
// events, the input is cancelled, or an attempt fails with an error that
// waiting will not fix, which reports Failed.
func (e *etwInput) Run(ctx input.Context, publisher stateless.Publisher) error {
	var err error

	// Lifecycle states are reported straight to the context; the agent
	// drops repeats of the state it already has. Per-event health goes
	// through e.health, which is the hot path and needs the counting logic.
	ctx.UpdateStatus(status.Starting, "")
	e.health = &eventHealth{
		reporter:          ctx,
		failureThreshold:  e.config.FailureThreshold,
		recoveryThreshold: e.config.RecoveryThreshold,
	}

	// Initialize a new ETW session with the provided configuration
	e.etwSession, err = e.operator.newSession(e.config)
	if err != nil {
		ctx.UpdateStatus(status.Failed, "failed to initialize ETW session: "+errDetail(err))
		return fmt.Errorf("error initializing ETW session: %w", err)
	}
	// The same session object, and so the same Callback value, is reused for
	// every reconnect. StartConsumer hands Callback to syscall.NewCallback,
	// which dedupes on the function value but never frees registrations and
	// caps them process-wide, so a fresh closure per attempt would leak.
	e.etwSession.Callback = e.consumeEvent
	e.publisher = publisher
	e.metrics = newInputMetrics(e.etwSession.Name, ctx.MetricsRegistry, ctx.Logger)

	// Set up logger with session information
	e.log = ctx.Logger.With("session", e.etwSession.Name)
	e.log.Info("Starting " + inputName + " input")
	defer e.log.Info(inputName + " input stopped")

	b := e.backoff
	if b == nil {
		b = backoff.NewEqualJitterBackoff(reconnectInitialBackoff, reconnectMaxBackoff)
	}
	cancelCtx := ctxtool.FromCanceller(ctx.Cancelation)

	// The first pass fails fast so that a bad configuration or missing
	// privileges surface as Failed with the runner's error log. Once the
	// session has worked, losing it is treated as transient: it is most
	// likely being restarted by whoever owns it, so we stay Degraded and
	// keep trying to get it back for as long as the input runs. Only
	// failures that waiting can fix are retried, though; see isTransient.
	for attempt := 0; ; attempt++ {
		err = e.runSession(ctx, cancelCtx, attempt == 0)
		switch {
		case cancelCtx.Err() != nil:
			// Shutting down. The consumer was stopped by us, so whatever it
			// returned is not a failure.
			return nil
		case !e.etwSession.Realtime:
			// Reading an .etl file: ProcessTrace returning means the file
			// has been read to the end. There is nothing to reconnect to.
			if err != nil {
				ctx.UpdateStatus(status.Failed, statusMessage(err))
			}
			return err
		case err != nil && (attempt == 0 || !isTransient(err)):
			ctx.UpdateStatus(status.Failed, statusMessage(err))
			return err
		}

		if err != nil {
			e.log.Warnw("reconnecting to ETW session failed, will retry", "attempt", attempt, "error", err)
			ctx.UpdateStatus(status.Degraded, fmt.Sprintf("%s; reconnecting (attempt %d)", statusMessage(err), attempt+1))
		} else {
			// The session was consuming and then ended without us asking:
			// another controller stopped it. Start the backoff over, since
			// the previous schedule was for a different outage.
			e.log.Warn("ETW session stopped by another controller, reconnecting")
			ctx.UpdateStatus(status.Degraded, fmt.Sprintf("ETW session %q stopped; reconnecting (attempt %d)",
				e.etwSession.Name, attempt+1))
			b.Reset()
		}
		if !b.Wait(cancelCtx) {
			return nil
		}
		// Counted after the wait so that it only covers attempts that are
		// made, not one that shutdown interrupted.
		e.metrics.reconnects.Inc()
		e.operator.resetSession(e.etwSession)
	}
}

// isTransient reports whether a reconnect failure is one that waiting can
// fix. Only "session is not running" qualifies: the owner has stopped the
// session and has not started it again yet, which is exactly what the
// reconnect loop exists to wait out. Anything else, such as access denied or
// a provider that cannot be enabled, is the same class of error that fails
// the first pass and does not clear by retrying, so the input reports Failed
// rather than sitting Degraded forever.
func isTransient(err error) bool {
	return errors.Is(err, etw.ERROR_WMI_INSTANCE_NOT_FOUND)
}

// runSession connects to the session and consumes it until ProcessTrace
// returns, then stops the session so its handles are released before any
// reconnect. cancelCtx is ctx.Cancelation as a context.Context. first selects
// the lifecycle statuses: only the first pass reports Configuring, so a
// reconnect stays Degraded until it is consuming again.
func (e *etwInput) runSession(ctx input.Context, cancelCtx context.Context, first bool) error {
	if e.etwSession.Realtime {
		if first {
			ctx.UpdateStatus(status.Configuring, "")
		}
		if err := e.connect(); err != nil {
			// connect can fail part way: StartTrace succeeds and enabling a
			// provider does not. That leaves a session running with no
			// providers, and the next attempt would hit ERROR_ALREADY_EXISTS,
			// attach to it and collect nothing. Tear down whatever connect
			// built; StopSession is a no-op if it built nothing.
			e.Close()
			return err
		}
	}

	// Stop the session on cancellation so that ProcessTrace returns. The
	// same stop runs when the consumer returns for any other reason, to
	// close the trace handle; OnceFunc makes the two paths safe together.
	stop := sync.OnceFunc(e.Close)
	unregister := context.AfterFunc(cancelCtx, stop)
	defer func() {
		unregister()
		stop()
	}()

	e.log.Debug("starting ETW consumer")
	defer e.log.Debug("stopped ETW consumer")
	// startConsumer opens the trace and then blocks inside ProcessTrace for
	// the life of the session, so there is no point after it returns where
	// we could report Running. Report it up front; if opening the trace
	// fails the caller moves to Failed or Degraded right after.
	ctx.UpdateStatus(status.Running, "")
	// A Degraded that eventHealth reported for the previous session has just
	// been overwritten by Running; its counters must agree or it would never
	// report Degraded again.
	e.health.reset()
	if err := e.operator.startConsumer(e.etwSession); err != nil {
		e.metrics.errors.Inc()
		return &sessionError{
			status: fmt.Sprintf("ETW consumer for session %q failed: %s", e.etwSession.Name, errDetail(err)),
			err:    fmt.Errorf("failed running ETW consumer: %w", err),
		}
	}
	return nil
}

// connect creates the realtime session or attaches to an existing one. When
// creating fails with ERROR_ALREADY_EXISTS the input attaches instead, which
// is also how it picks a session back up after a reconnect races the owner.
func (e *etwInput) connect() error {
	switch e.etwSession.NewSession {
	case true:
		createErr := e.operator.createRealtimeSession(e.etwSession)
		if createErr == nil {
			e.log.Debug("created new session")
			return nil
		}
		if !errors.Is(createErr, etw.ERROR_ALREADY_EXISTS) {
			return &sessionError{
				status: fmt.Sprintf("failed to create realtime session %q for provider %s: %s",
					e.etwSession.Name, e.config.provider(), errDetail(createErr)),
				err: fmt.Errorf("realtime session could not be created: %w", createErr),
			}
		}
		e.log.Debug("session already exists, trying to attach to it")
		fallthrough
	case false:
		if err := e.operator.attachToExistingSession(e.etwSession); err != nil {
			return &sessionError{
				status: fmt.Sprintf("failed to attach to session %q: %s", e.etwSession.Name, errDetail(err)),
				err:    fmt.Errorf("unable to retrieve handler: %w", err),
			}
		}
		e.log.Debug("attached to existing session")
	}
	return nil
}

// sessionError is returned by runSession. It carries the message for the
// status reporter separately from the error returned to the runner: the status
// message names the session and provider and spells out Windows error codes,
// while the returned error keeps the shorter form that ends up in the runner's
// log line. Run decides whether it means Failed or Degraded.
type sessionError struct {
	status string
	err    error
}

func (e *sessionError) Error() string { return e.err.Error() }
func (e *sessionError) Unwrap() error { return e.err }

// statusMessage returns the text to report for err.
func statusMessage(err error) string {
	var se *sessionError
	if errors.As(err, &se) {
		return se.status
	}
	return errDetail(err)
}

// eventHealth turns the per-event success/failure stream from consumeEvent
// into Degraded/Running reports using the consecutive-count model of
// Kubernetes probes and the awss3 input: failureThreshold failures in a row
// mark the input Degraded, recoveryThreshold successes in a row clear it.
// The two are separate knobs because there is no reason to expect the run
// lengths of good and bad events to match in a failing provider.
type eventHealth struct {
	reporter          status.StatusReporter
	failureThreshold  uint // 0 means never report Degraded.
	recoveryThreshold uint // Validated to be at least 1 if thresholding is being used.

	// ETW delivers callbacks on a single thread today, so mu is rarely
	// contended; it is cheap insurance against that changing.
	mu       sync.Mutex
	bad      uint // Consecutive failures, reset by a success.
	good     uint // Consecutive successes, reset by a failure.
	degraded bool
}

// failure records an event that could not be read. err is only formatted
// when the threshold is reached, since most calls never report anything.
func (h *eventHealth) failure(err error) {
	if h.failureThreshold == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.good = 0
	h.bad++
	if h.degraded || h.bad < h.failureThreshold {
		return
	}
	h.degraded = true
	h.reporter.UpdateStatus(status.Degraded,
		fmt.Sprintf("%d consecutive events could not be read; last error: %s", h.bad, errDetail(err)))
}

// success records an event that was read and published.
func (h *eventHealth) success() {
	if h.failureThreshold == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.bad = 0
	if !h.degraded {
		return
	}
	h.good++
	if h.good < h.recoveryThreshold {
		return
	}
	h.degraded = false
	h.reporter.UpdateStatus(status.Running, "")
}

// reset forgets the run counts and any Degraded that was reported. It is for
// when the input has reported a lifecycle status itself, such as Running
// after a reconnect, so that the two views of the input's health agree.
func (h *eventHealth) reset() {
	h.mu.Lock()
	h.bad, h.good, h.degraded = 0, 0, false
	h.mu.Unlock()
}

var (
	// levelToSeverity maps ETW trace levels to names for use in ECS log.level.
	// Based on Microsoft ETW documentation and Windows Event Log conventions.
	levelToSeverity = map[uint8]string{
		0: "information", // Catch-all/unknown level, treated as information
		1: "critical",    // Abnormal exit or termination events
		2: "error",       // Severe error events
		3: "warning",     // Warning events such as allocation failures
		4: "information", // Non-error events such as entry or exit events
		5: "verbose",     // Detailed trace events
	}

	// zeroGUID is the zero-value for a windows.GUID.
	zeroGUID = windows.GUID{}
)

// isValidValue checks if a value should be included in the event (not empty).
func isValidValue(value any) bool {
	switch v := value.(type) {
	case string:
		return v != ""
	case []string:
		return len(v) > 0
	case []byte:
		return len(v) > 0
	case windows.GUID:
		return v != zeroGUID
	case nil:
		return false
	default:
		return true
	}
}

// setIfValid sets a field in the map only if the value is valid (not empty).
func setIfValid(m map[string]any, key string, value any) {
	if isValidValue(value) {
		m[key] = value
	}
}

// buildEvent builds the final beat.Event emitted by this input.
func buildEvent(etwEvent etw.RenderedEtwEvent, h etw.EventHeader, session *etw.Session, cfg config) beat.Event {
	winlog := map[string]any{
		"flags_raw":    fmt.Sprintf("0x%X", h.Flags),
		"keywords_raw": fmt.Sprintf("0x%X", etwEvent.KeywordsRaw),
		"opcode_raw":   etwEvent.OpcodeRaw,
		"process_id":   strconv.FormatUint(uint64(etwEvent.ProcessID), 10),
		"session":      session.Name,
		"task_raw":     etwEvent.TaskRaw,
		"level_raw":    etwEvent.LevelRaw,
		"thread_id":    strconv.FormatUint(uint64(etwEvent.ThreadID), 10),
		"version":      etwEvent.Version,
	}

	// Only set fields that have valid (non-empty) values
	if h.ActivityId != zeroGUID {
		winlog["activity_id"] = h.ActivityId.String()
	}
	setIfValid(winlog, "activity_id_name", etwEvent.ActivityIDName)
	setIfValid(winlog, "related_activity_id_name", etwEvent.RelatedActivityIDName)
	setIfValid(winlog, "channel", etwEvent.Channel)
	setIfValid(winlog, "keywords", etwEvent.Keywords)
	setIfValid(winlog, "opcode", etwEvent.Opcode)
	setIfValid(winlog, "task", etwEvent.Task)
	setIfValid(winlog, "level", etwEvent.Level)
	setIfValid(winlog, "flags", h.FlagsAsStrings())
	setIfValid(winlog, "provider_message", etwEvent.ProviderMessage)

	// Handle provider GUID with fallback to session GUID
	if etwEvent.ProviderGUID != zeroGUID {
		winlog["provider_guid"] = etwEvent.ProviderGUID.String()
	}

	eventData := mapstr.M{}
	for _, prop := range etwEvent.Properties {
		if !isValidValue(prop.Value) {
			continue
		}
		switch v := prop.Value.(type) {
		case []byte:
			eventData.Put(prop.Name, fmt.Sprintf("0x%X", v))
		default:
			eventData.Put(prop.Name, v)
		}
	}

	extended := mapstr.M{}
	for _, ext := range etwEvent.ExtendedData {
		if !isValidValue(ext.Data) {
			continue
		}
		switch v := ext.Data.(type) {
		case []byte:
			extended.Put(ext.ExtType, fmt.Sprintf("0x%X", v))
		default:
			extended.Put(ext.ExtType, v)
		}
	}

	if len(extended) > 0 {
		eventData.Put("extended_data", extended)
	}

	if len(eventData) > 0 {
		winlog["event_data"] = eventData
	}

	event := mapstr.M{
		"code":     strconv.FormatUint(uint64(h.EventDescriptor.Id), 10),
		"created":  time.Now().UTC(),
		"kind":     "event",
		"severity": h.EventDescriptor.Level,
	}
	if etwEvent.ProviderName != "" {
		event["provider"] = etwEvent.ProviderName
	} else if cfg.ProviderName != "" {
		event["provider"] = cfg.ProviderName
	}

	fields := mapstr.M{
		"event":  event,
		"winlog": winlog,
	}

	// Only set message if it's not empty
	if etwEvent.EventMessage != "" {
		fields["message"] = etwEvent.EventMessage
	}

	if level, found := levelToSeverity[etwEvent.LevelRaw]; found {
		fields.Put("log.level", level)
	}
	if cfg.Logfile != "" {
		fields.Put("log.file.path", cfg.Logfile)
	}

	return beat.Event{
		Timestamp: etwEvent.Timestamp,
		Fields:    fields,
	}
}

// errNullRecord is reported to eventHealth when ETW hands us a nil record.
var errNullRecord = errors.New("received null event record from ETW session")

func (e *etwInput) consumeEvent(record *etw.EventRecord) uintptr {
	if record == nil {
		e.log.Error("received null event record")
		e.metrics.errors.Inc()
		e.health.failure(errNullRecord)
		return 1
	}

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		e.metrics.processingTime.Update(elapsed.Nanoseconds())
	}()

	etwEvent, err := e.etwSession.RenderEvent(record)
	if err != nil {
		// Unprocessable events are expected noise from some providers and
		// are dropped quietly; anything else means we are losing data.
		if !errors.Is(err, etw.ErrUnprocessableEvent) {
			e.log.Errorw("failed to read event properties", "error", err)
			e.metrics.errors.Inc()
			e.metrics.dropped.Inc()
			e.health.failure(err)
		}
		return 1
	}

	evt := buildEvent(etwEvent, record.EventHeader, e.etwSession, e.config)
	e.publisher.Publish(evt)

	e.health.success()
	e.metrics.events.Inc()
	e.metrics.sourceLag.Update(start.Sub(evt.Timestamp).Nanoseconds())
	if !e.metrics.lastCallback.IsZero() {
		e.metrics.arrivalPeriod.Update(start.Sub(e.metrics.lastCallback).Nanoseconds())
	}
	e.metrics.lastCallback = start

	return 0
}

// errDetail formats err for a status message. When a Windows error is wrapped
// it appends the raw code, since the text alone ("Access is denied.") is often
// not enough to search for, and for access denied it spells out what
// privileges ETW needs.
func errDetail(err error) string {
	var errno windows.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}
	msg := fmt.Sprintf("%v (windows error %d)", err, uint32(errno))
	if errno == etw.ERROR_ACCESS_DENIED {
		msg += "; ETW sessions require running as Administrator or membership in the Performance Log Users group"
	}
	return msg
}

// Close stops the ETW session and logs the outcome. It runs both at shutdown
// and after a session ends on its own, to release the trace handle before the
// session is reused.
func (e *etwInput) Close() {
	if err := e.operator.stopSession(e.etwSession); err != nil {
		e.log.Errorw("failed to stop ETW session", "error", err)
		e.metrics.errors.Inc()
		return
	}
	e.log.Info("ETW session stopped")
}

// inputMetrics handles event log metric reporting.
type inputMetrics struct {
	lastCallback time.Time

	name           *monitoring.String // name of the etw session being read
	events         *monitoring.Uint   // total number of events received
	dropped        *monitoring.Uint   // total number of discarded events
	errors         *monitoring.Uint   // total number of errors
	reconnects     *monitoring.Uint   // total number of attempts to reconnect after the session ended
	sourceLag      metrics.Sample     // histogram of the difference between timestamped event's creation and reading
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between callbacks.
	processingTime metrics.Sample     // histogram of the elapsed time between event callback receipt and publication.
}

// newInputMetrics returns an input metric for windows ETW.
// If id is empty, a nil inputMetric is returned.
func newInputMetrics(session string, reg *monitoring.Registry, logger *logp.Logger) *inputMetrics {
	out := &inputMetrics{
		name:           monitoring.NewString(reg, "session"),
		events:         monitoring.NewUint(reg, "received_events_total"),
		dropped:        monitoring.NewUint(reg, "discarded_events_total"),
		errors:         monitoring.NewUint(reg, "errors_total"),
		reconnects:     monitoring.NewUint(reg, "reconnects_total"),
		sourceLag:      metrics.NewUniformSample(1024),
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	out.name.Set(session)
	_ = adapter.NewGoMetrics(reg, "source_lag_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.sourceLag))
	_ = adapter.NewGoMetrics(reg, "arrival_period", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	return out
}
