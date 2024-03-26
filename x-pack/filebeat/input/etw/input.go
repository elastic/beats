// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"math"
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
	config     config
	etwSession *etw.Session
	operator   sessionOperator
}

func Plugin() input.Plugin {
	return input.Plugin{
		Name:      inputName,
		Stability: feature.Beta,
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

	// Set up logger with session information
	e.log = ctx.Logger.With("session", e.etwSession.Name)
	e.log.Info("Starting " + inputName + " input")

	// Handle realtime session creation or attachment
	if e.etwSession.Realtime {
		if !e.etwSession.NewSession {
			// Attach to an existing session
			err = e.operator.attachToExistingSession(e.etwSession)
			if err != nil {
				return fmt.Errorf("unable to retrieve handler: %w", err)
			}
			e.log.Debug("attached to existing session")
		} else {
			// Create a new realtime session
			err = e.operator.createRealtimeSession(e.etwSession)
			if err != nil {
				return fmt.Errorf("realtime session could not be created: %w", err)
			}
			e.log.Debug("created session")
		}
	}
	// Defer the cleanup and closing of resources
	var wg sync.WaitGroup
	var once sync.Once

	// Create an error channel to communicate errors from the goroutine
	errChan := make(chan error, 1)

	defer func() {
		once.Do(e.Close)
		e.log.Info(inputName + " input stopped")
	}()

	// eventReceivedCallback processes each ETW event
	eventReceivedCallback := func(record *etw.EventRecord) uintptr {
		if record == nil {
			e.log.Error("received null event record")
			return 1
		}

		e.log.Debugf("received event %d with length %d", record.EventHeader.EventDescriptor.Id, record.UserDataLength)

		data, err := etw.GetEventProperties(record)
		if err != nil {
			e.log.Errorw("failed to read event properties", "error", err)
			return 1
		}

		evt := buildEvent(data, record.EventHeader, e.etwSession, e.config)
		publisher.Publish(evt)

		return 0
	}

	// Set the callback function for the ETW session
	e.etwSession.Callback = eventReceivedCallback

	// Start a goroutine to consume ETW events
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.log.Debug("starting to listen ETW events")
		if err = e.operator.startConsumer(e.etwSession); err != nil {
			errChan <- fmt.Errorf("failed to start consumer: %w", err) // Send error to channel
			return
		}
		e.log.Debug("stopped to read ETW events from session")
		errChan <- nil
	}()

	// We ensure resources are closed when receiving a cancellation signal
	go func() {
		<-ctx.Cancelation.Done()
		once.Do(e.Close)
	}()

	wg.Wait() // Ensure all goroutines have finished before closing
	close(errChan)
	if err, ok := <-errChan; ok && err != nil {
		return err
	}

	return nil
}

var (
	// levelToSeverity maps ETW trace levels to names for use in ECS log.level.
	levelToSeverity = map[uint8]string{
		1: "critical",    // Abnormal exit or termination events
		2: "error",       // Severe error events
		3: "warning",     // Warning events such as allocation failures
		4: "information", // Non-error events such as entry or exit events
		5: "verbose",     // Detailed trace events
	}

	// zeroGUID is the zero-value for a windows.GUID.
	zeroGUID = windows.GUID{}
)

// buildEvent builds the final beat.Event emitted by this input.
func buildEvent(data map[string]any, h etw.EventHeader, session *etw.Session, cfg config) beat.Event {
	winlog := map[string]any{
		"activity_id":   h.ActivityId.String(),
		"channel":       strconv.FormatUint(uint64(h.EventDescriptor.Channel), 10),
		"event_data":    data,
		"flags":         strconv.FormatUint(uint64(h.Flags), 10),
		"keywords":      strconv.FormatUint(h.EventDescriptor.Keyword, 10),
		"opcode":        strconv.FormatUint(uint64(h.EventDescriptor.Opcode), 10),
		"process_id":    strconv.FormatUint(uint64(h.ProcessId), 10),
		"provider_guid": h.ProviderId.String(),
		"session":       session.Name,
		"task":          strconv.FormatUint(uint64(h.EventDescriptor.Task), 10),
		"thread_id":     strconv.FormatUint(uint64(h.ThreadId), 10),
		"version":       h.EventDescriptor.Version,
	}
	// Fallback to the session GUID if there is no provider GUID.
	if h.ProviderId == zeroGUID {
		winlog["provider_guid"] = session.GUID.String()
	}

	event := mapstr.M{
		"code":     strconv.FormatUint(uint64(h.EventDescriptor.Id), 10),
		"created":  time.Now().UTC(),
		"kind":     "event",
		"severity": h.EventDescriptor.Level,
	}
	if cfg.ProviderName != "" {
		event["provider"] = cfg.ProviderName
	}

	fields := mapstr.M{
		"event":  event,
		"winlog": winlog,
	}
	if level, found := levelToSeverity[h.EventDescriptor.Level]; found {
		fields.Put("log.level", level)
	}
	if cfg.Logfile != "" {
		fields.Put("log.file.path", cfg.Logfile)
	}

	return beat.Event{
		Timestamp: convertFileTimeToGoTime(uint64(h.TimeStamp)),
		Fields:    fields,
	}
}

// convertFileTimeToGoTime converts a Windows FileTime to a Go time.Time structure.
func convertFileTimeToGoTime(fileTime64 uint64) time.Time {
	// Define the offset between Windows epoch (1601) and Unix epoch (1970)
	const epochDifference = 116444736000000000
	if fileTime64 < epochDifference {
		// Time is before the Unix epoch, adjust accordingly
		return time.Time{}
	}

	fileTime := windows.Filetime{
		HighDateTime: uint32(fileTime64 >> 32),
		LowDateTime:  uint32(fileTime64 & math.MaxUint32),
	}

	return time.Unix(0, fileTime.Nanoseconds()).UTC()
}

// Close stops the ETW session and logs the outcome.
func (e *etwInput) Close() {
	if err := e.operator.stopSession(e.etwSession); err != nil {
		e.log.Error("failed to shutdown ETW session")
		return
	}
	e.log.Info("successfully shutdown")
}
