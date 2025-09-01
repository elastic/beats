// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/etw"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"

	"github.com/rcrowley/go-metrics"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/windows"
)

const (
	inputName = "etw"
)

// It abstracts the underlying operations needed to work with ETW, allowing for easier
// testing and decoupling from the Windows-specific ETW API.
type sessionOperator interface {
	newSession(config config) (*etw.Session, error)
	attachToExistingSession(session *etw.Session) error
	createRealtimeSession(session *etw.Session) error
	startConsumer(session *etw.Session) error
	stopSession(session *etw.Session) error
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

// etwInput struct holds the configuration and state for the ETW input
type etwInput struct {
	log        *logp.Logger
	metrics    *inputMetrics
	config     config
	etwSession *etw.Session
	publisher  stateless.Publisher
	operator   sessionOperator
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
func (e *etwInput) Run(ctx input.Context, publisher stateless.Publisher) error {
	var err error

	// Initialize a new ETW session with the provided configuration
	e.etwSession, err = e.operator.newSession(e.config)
	if err != nil {
		return fmt.Errorf("error initializing ETW session: %w", err)
	}
	e.etwSession.Callback = e.consumeEvent
	e.publisher = publisher
	e.metrics = newInputMetrics(e.etwSession.Name, ctx.MetricsRegistry, ctx.Logger)

	// Set up logger with session information
	e.log = ctx.Logger.With("session", e.etwSession.Name)
	e.log.Info("Starting " + inputName + " input")
	defer e.log.Info(inputName + " input stopped")

	// Handle realtime session creation or attachment
	if e.etwSession.Realtime {
		switch e.etwSession.NewSession {
		case true:
			// Create a new realtime session
			// If it fails with ERROR_ALREADY_EXISTS we try to attach to it
			createErr := e.operator.createRealtimeSession(e.etwSession)
			if createErr == nil {
				e.log.Debug("created new session")
				break
			}
			if !errors.Is(createErr, etw.ERROR_ALREADY_EXISTS) {
				return fmt.Errorf("realtime session could not be created: %w", createErr)
			}
			e.log.Debug("session already exists, trying to attach to it")
			fallthrough
		case false:
			// Attach to an existing session
			if err := e.operator.attachToExistingSession(e.etwSession); err != nil {
				return fmt.Errorf("unable to retrieve handler: %w", err)
			}
			e.log.Debug("attached to existing session")
		}
	}

	stopConsumer := sync.OnceFunc(e.Close)
	defer stopConsumer()

	// Stop the consumer upon input cancellation (shutdown).
	go func() {
		<-ctx.Cancelation.Done()
		stopConsumer()
	}()

	// Start a goroutine to consume ETW events
	g := new(errgroup.Group)
	g.Go(func() error {
		e.log.Debug("starting ETW consumer")
		defer e.log.Debug("stopped ETW consumer")
		if err = e.operator.startConsumer(e.etwSession); err != nil {
			e.metrics.errors.Inc()
			return fmt.Errorf("failed running ETW consumer: %w", err)
		}
		return nil
	})

	return g.Wait()
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

func (e *etwInput) consumeEvent(record *etw.EventRecord) uintptr {
	if record == nil {
		e.log.Error("received null event record")
		e.metrics.errors.Inc()
		return 1
	}

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		e.metrics.processingTime.Update(elapsed.Nanoseconds())
	}()

	etwEvent, err := e.etwSession.RenderEvent(record)
	if err != nil {
		if !errors.Is(err, etw.ErrUnprocessableEvent) {
			e.log.Errorw("failed to read event properties", "error", err)
			e.metrics.errors.Inc()
			e.metrics.dropped.Inc()
		}
		return 1
	}

	evt := buildEvent(etwEvent, record.EventHeader, e.etwSession, e.config)
	e.publisher.Publish(evt)

	e.metrics.events.Inc()
	e.metrics.sourceLag.Update(start.Sub(evt.Timestamp).Nanoseconds())
	if !e.metrics.lastCallback.IsZero() {
		e.metrics.arrivalPeriod.Update(start.Sub(e.metrics.lastCallback).Nanoseconds())
	}
	e.metrics.lastCallback = start

	return 0
}

// Close stops the ETW session and logs the outcome.
func (e *etwInput) Close() {
	if err := e.operator.stopSession(e.etwSession); err != nil {
		e.log.Error("failed to shutdown ETW session")
		e.metrics.errors.Inc()
		return
	}
	e.log.Info("successfully shutdown")
}

// inputMetrics handles event log metric reporting.
type inputMetrics struct {
	lastCallback time.Time

	name           *monitoring.String // name of the etw session being read
	events         *monitoring.Uint   // total number of events received
	dropped        *monitoring.Uint   // total number of discarded events
	errors         *monitoring.Uint   // total number of errors
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
