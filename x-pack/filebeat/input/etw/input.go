// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw_input

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

type etw_input struct {
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

	return newETW(conf)
}

func newETW(config config) (*etw_input, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &etw_input{
		config: config,
	}, nil
}

func (e *etw_input) Name() string { return inputName }

func (e *etw_input) Test(_ input.TestContext) error {
	// ToDo
	return nil
}

func (e *etw_input) Run(ctx input.Context, publisher stateless.Publisher) error {
	var err error
	e.etwSession, err = etw.NewSession(convertConfig(e.config))
	if err != nil {
		return fmt.Errorf("error when inicializing '%s' session: %v", e.etwSession.Name, err)
	}

	e.log := ctx.Logger.With("session", e.etwSession.Name)
	e.log.Info("Starting " + inputName + " input")

	var wg sync.WaitGroup

	if e.etwSession.Realtime {
		err = e.etwSession.CreateRealtimeSession()
		if err != nil {
			return fmt.Errorf("realtime session could not be created: %v", e.etwSession.Name, err)
		}
		e.log.Debug("created session")
	}
	defer func() {
		wg.Wait() // Wait for the goroutine to finish
		e.close()
		e.log.Info(inputName + " input stopped")
	}()

	// Define callback that will process ETW events
	// Callback which receives every ETW event from the reading source
	eventReceivedCallback := func(er *etw.EventRecord) uintptr {
		if er == nil {
			e.log.Error("received null event record")
			return 1
		}

		e.log.Debugf("received event %d with length %d", er.EventHeader.EventDescriptor.Id, er.UserDataLength)

		event := make(map[string]interface{})
		event["Header"] = er.EventHeader

		if data, err := etw.GetEventProperties(er); err == nil {
			event["EventProperties"] = data
		} else {
			e.log.Errorf("failed to read event properties: %s", err)
			return 1
		}

		evt := beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"header": event["Header"],
				"winlog": event["EventProperties"],
			},
		}
		publisher.Publish(evt)

		return 0
	}

	e.etwSession.Callback = syscall.NewCallback(eventReceivedCallback)

	wg.Add(1)
	go func() {
		defer wg.Done()
		e.log.Debug("starting to listen ETW events")
		if err = e.etwSession.StartConsumer(); err != nil {
			e.log.Warnf("events could not be read: %v", err)
		}
		e.log.Debug("stopped to read ETW events")
	}()

	return nil
}

// Closes all the opened handlers and resources
func (e *etw_input) close() {
	if err := e.etwSession.StopSession(); err != nil {
		e.log.Error("failed to shutdown ETW session")
	}
	e.log.Info("successfully shutdown")
}
