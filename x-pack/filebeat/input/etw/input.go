// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/libbeat/reader/etw"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	inputName = "etw"
)

// etwInput struct holds the configuration and state for the ETW input
type etwInput struct {
	log        *logp.Logger
	config     config
	etwSession etw.Session
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
		return fmt.Errorf("error when initializing '%s' session: %w", e.etwSession.Name, err)
	}

	// Set up logger with session information.
	e.log = ctx.Logger.With("session", e.etwSession.Name)
	e.log.Info("Starting " + inputName + " input")

	var wg sync.WaitGroup
	var once sync.Once

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
	defer func() {
		wg.Wait() // Ensure all goroutines have finished before closing.
		once.Do(e.Close)
		e.log.Info(inputName + " input stopped")
	}()

	// eventReceivedCallback processes each ETW event.
	eventReceivedCallback := func(er *etw.EventRecord) uintptr {
		if er == nil {
			e.log.Error("received null event record")
			return 1
		}

		e.log.Debugf("received event %d with length %d", er.EventHeader.EventDescriptor.Id, er.UserDataLength)

		var event map[string]interface{}

		if data, err := etw.GetEventProperties(er); err == nil {
			event = map[string]interface{}{
				"Header":          er.EventHeader,
				"EventProperties": data,
				"Metadata":        fillEventMetadata(er, e.etwSession, e.config),
			}
		} else {
			e.log.Errorf("failed to read event properties: %w", err)
			return 1
		}

		evt := beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"metadata": event["Metadata"],
				"header":   event["Header"],
				"winlog":   event["EventProperties"],
			},
		}
		publisher.Publish(evt)

		return 0
	}

	// Set the callback function for the ETW session.
	e.etwSession.Callback = syscall.NewCallback(eventReceivedCallback)

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

// fillEventMetadata constructs a metadata map for an event record.
func fillEventMetadata(er *etw.EventRecord, session etw.Session, cfg config) map[string]interface{} {
	// Mapping from Level to Severity
	levelToSeverity := map[uint8]string{
		1: "critical",
		2: "error",
		3: "warning",
		4: "information",
		5: "verbose",
	}

	metadata := make(map[string]interface{})

	// Get the severity level, with a default value if not found
	severity, ok := levelToSeverity[er.EventHeader.EventDescriptor.Level]
	if !ok {
		severity = "unknown" // Default severity level
	}
	metadata["Severity"] = severity

	// Include provider name and GUID in metadata if available
	if cfg.ProviderName != "" {
		metadata["ProviderName"] = cfg.ProviderName
	}
	if cfg.ProviderGUID != "" {
		metadata["ProviderGUID"] = cfg.ProviderGUID
	} else if etw.IsGUIDValid(session.GUID) {
		metadata["ProviderGUID"] = etw.GUIDToString(session.GUID)
	}

	// Include logfile path if available
	if cfg.Logfile != "" {
		metadata["Logfile"] = cfg.Logfile
	}

	// Include session name if available
	if cfg.Session != "" {
		metadata["Session"] = cfg.Session
	} else if cfg.SessionName != "" {
		metadata["Session"] = cfg.SessionName
	} else if cfg.ProviderGUID != "" || cfg.ProviderName != "" {
		metadata["Session"] = session.Name
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
