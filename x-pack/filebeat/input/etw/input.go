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

// etwInput struct holds the configuration and state for the ETW input
type etwInput struct {
	log        *logp.Logger
	config     config
	etwSession *etw.Session
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
		config: conf,
	}, nil
}

func (e *etwInput) Name() string { return inputName }

func (e *etwInput) Test(_ input.TestContext) error {
	// ToDo
	return nil
}

// Run starts the ETW session and processes incoming events.
func (e *etwInput) Run(ctx input.Context, publisher stateless.Publisher) error {
	var err error
	// Initialize a new ETW session with the provided configuration.
	e.etwSession, err = etw.NewSession(convertConfig(e.config))
	if err != nil {
		return fmt.Errorf("error initializing ETW session: %w", err)
	}

	// Set up logger with session information.
	e.log = ctx.Logger.With("session", e.etwSession.Name)
	e.log.Info("Starting " + inputName + " input")

	// Handle realtime session creation or attachment.
	if e.etwSession.Realtime {
		if !e.etwSession.NewSession {
			// Attach to an existing session.
			err = e.etwSession.GetHandler()
			if err != nil {
				return fmt.Errorf("unable to retrieve handler: %w", err)
			}
			e.log.Debug("attached to existing session")
		} else {
			// Create a new realtime session.
			err = e.etwSession.CreateRealtimeSession()
			if err != nil {
				return fmt.Errorf("realtime session could not be created: %w", err)
			}
			e.log.Debug("created session")
		}
	}
	// Defer the cleanup and closing of resources.
	var wg sync.WaitGroup
	var once sync.Once

	defer func() {
		wg.Wait() // Ensure all goroutines have finished before closing.
		once.Do(e.Close)
		e.log.Info(inputName + " input stopped")
	}()

	// eventReceivedCallback processes each ETW event.
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
				"metadata": fillEventMetadata(record, e.etwSession, e.config),
				"header":   fillEventHeader(record.EventHeader),
				"winlog":   data,
			},
		}

		publisher.Publish(evt)

		return 0
	}

	// Set the callback function for the ETW session.
	e.etwSession.Callback = eventReceivedCallback

	// Start a goroutine to consume ETW events.
	wg.Add(1)
	go func() {
		defer wg.Done()
		e.log.Debug("starting to listen ETW events")
		if err = e.etwSession.StartConsumer(); err != nil {
			e.log.Warnf("events could not be read from session: %w", err)
		}
		e.log.Debug("stopped to read ETW events from session")
	}()

	// We ensure resources are closed when receiving a cancelation signal
	go func() {
		<-ctx.Cancelation.Done()
		once.Do(e.Close)
	}()

	return nil
}

// fillEventHeader constructs a header map for an event record header.
func fillEventHeader(h etw.EventHeader) map[string]interface{} {
	// Mapping from Level to Severity
	levelToSeverity := map[uint8]string{
		1: "critical",
		2: "error",
		3: "warning",
		4: "information",
		5: "verbose",
	}

	header := make(map[string]interface{})

	header["size"] = h.Size
	header["type"] = h.HeaderType
	header["flags"] = h.Flags
	header["event_property"] = h.EventProperty
	header["thread_id"] = h.ThreadId
	header["process_id"] = h.ProcessId
	header["timestamp"] = convertFileTimeToGoTime(uint64(h.TimeStamp))
	header["provider_guid"] = h.ProviderId.String()
	header["event_id"] = h.EventDescriptor.Id
	header["event_version"] = h.EventDescriptor.Version
	header["channel"] = h.EventDescriptor.Channel
	header["level"] = h.EventDescriptor.Level
	// Get the severity level, with a default value if not found
	severity, ok := levelToSeverity[h.EventDescriptor.Level]
	if !ok {
		severity = "unknown" // Default severity level
	}
	header["severity"] = severity
	header["opcode"] = h.EventDescriptor.Opcode
	header["task"] = h.EventDescriptor.Task
	header["keyword"] = h.EventDescriptor.Keyword
	header["time"] = h.Time
	header["activity_guid"] = h.ActivityId.String()

	return header
}

// convertFileTimeToGoTime converts a Windows FileTime to a Go time.Time structure.
func convertFileTimeToGoTime(fileTime64 uint64) time.Time {
	fileTime := windows.Filetime{
		HighDateTime: uint32(fileTime64 >> 32),
		LowDateTime:  uint32(fileTime64 & math.MaxUint32),
	}

	return time.Unix(0, fileTime.Nanoseconds())
}

// fillEventMetadata constructs a metadata map for an event record.
func fillEventMetadata(record *etw.EventRecord, session *etw.Session, cfg config) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Include provider name and GUID in metadata if available
	if cfg.ProviderName != "" {
		metadata["provider_name"] = cfg.ProviderName
	}
	if cfg.ProviderGUID != "" {
		metadata["provider_guid"] = cfg.ProviderGUID
	} else if etw.IsGUIDValid(session.GUID) {
		metadata["provider_guid"] = session.GUID.String()
	}

	// Include logfile path if available
	if cfg.Logfile != "" {
		metadata["logfile"] = cfg.Logfile
	}

	// Include session name if available
	if cfg.Session != "" {
		metadata["session"] = cfg.Session
	} else if cfg.SessionName != "" {
		metadata["session"] = cfg.SessionName
	} else if cfg.ProviderGUID != "" || cfg.ProviderName != "" {
		metadata["session"] = session.Name
	}

	return metadata
}

// close stops the ETW session and logs the outcome.
func (e *etwInput) Close() {
	if err := e.etwSession.StopSession(); err != nil {
		e.log.Error("failed to shutdown ETW session")
	}
	e.log.Info("successfully shutdown")
}
