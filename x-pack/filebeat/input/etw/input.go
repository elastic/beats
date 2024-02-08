// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"math"
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
			e.log.Errorf("failed to read event properties: %w", err)
			return 1
		}

		evt := beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"winlog": buildEvent(data, record.EventHeader, e.etwSession, e.config),
			},
		}

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

	// We ensure resources are closed when receiving a cancelation signal
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

// buildEvent builds the winlog object.
func buildEvent(data map[string]any, h etw.EventHeader, session *etw.Session, cfg config) map[string]any {
	// Mapping from Level to Severity
	levelToSeverity := map[uint8]string{
		1: "critical",
		2: "error",
		3: "warning",
		4: "information",
		5: "verbose",
	}

	// Get the severity level, with a default value if not found
	severity, ok := levelToSeverity[h.EventDescriptor.Level]
	if !ok {
		severity = "unknown" // Default severity level
	}

	winlog := map[string]any{
		"activity_guid": h.ActivityId.String(),
		"channel":       h.EventDescriptor.Channel,
		"event_data":    data,
		"event_id":      h.EventDescriptor.Id,
		"flags":         h.Flags,
		"keywords":      h.EventDescriptor.Keyword,
		"level":         h.EventDescriptor.Level,
		"opcode":        h.EventDescriptor.Opcode,
		"process_id":    h.ProcessId,
		"provider_guid": h.ProviderId.String(),
		"session":       session.Name,
		"severity":      severity,
		"task":          h.EventDescriptor.Task,
		"thread_id":     h.ThreadId,
		"timestamp":     convertFileTimeToGoTime(uint64(h.TimeStamp)),
		"version":       h.EventDescriptor.Version,
	}

	// Include provider name and GUID if available
	if cfg.ProviderName != "" {
		winlog["provider_name"] = cfg.ProviderName
	}

	zeroGUID := "{00000000-0000-0000-0000-000000000000}"
	if winlog["provider_guid"] == zeroGUID {
		winlog["provider_guid"] = session.GUID.String()
	}

	// Include logfile path if available
	if cfg.Logfile != "" {
		winlog["logfile"] = cfg.Logfile
	}

	return winlog
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

	return time.Unix(0, fileTime.Nanoseconds())
}

// close stops the ETW session and logs the outcome.
func (e *etwInput) Close() {
	if err := e.operator.stopSession(e.etwSession); err != nil {
		e.log.Error("failed to shutdown ETW session")
	}
	e.log.Info("successfully shutdown")
}
