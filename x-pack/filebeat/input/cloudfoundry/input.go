// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func init() {
	err := input.Register("cloudfoundry", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input defines a udp input to receive event on a specific host:port.
type Input struct {
	sync.Mutex
	listener *cloudfoundry.RlpListener
	started  bool
	log      *logp.Logger
	outlet   channel.Outleter
}

// NewInput creates a new udp input
func NewInput(
	cfg *common.Config,
	outlet channel.Connector,
	context input.Context,
) (input.Input, error) {
	log := logp.NewLogger("cloudfoundry")

	out, err := outlet.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: context.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
	}

	var conf cloudfoundry.Config
	if err = cfg.Unpack(&conf); err != nil {
		return nil, err
	}

	hub := cloudfoundry.NewHub(&conf, "filebeat", log)
	forwarder := harvester.NewForwarder(out)
	callbacks := cloudfoundry.RlpListenerCallbacks{
		HttpAccess: func(evt *cloudfoundry.EventHttpAccess) {
			forwarder.Send(beat.Event{
				Timestamp: evt.Timestamp(),
				Fields:    evt.ToFields(),
			})
		},
		Log: func(evt *cloudfoundry.EventLog) {
			forwarder.Send(beat.Event{
				Timestamp: evt.Timestamp(),
				Fields:    evt.ToFields(),
			})
		},
		Error: func(evt *cloudfoundry.EventError) {
			forwarder.Send(beat.Event{
				Timestamp: evt.Timestamp(),
				Fields:    evt.ToFields(),
			})
		},
	}

	listener, err := hub.RlpListener(callbacks)
	if err != nil {
		return nil, err
	}
	return &Input{
		outlet:   out,
		listener: listener,
		started:  false,
		log:      log,
	}, nil
}

// Run starts and start the UDP server and read events from the socket
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		p.log.Info("starting cloudfoundry input")
		p.listener.Start(context.TODO())
		p.started = true
	}
}

// Stop stops the UDP input
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	p.log.Info("stopping cloudfoundry input")
	p.listener.Stop()
	p.started = false
}

// Wait suspends the UDP input
func (p *Input) Wait() {
	p.Stop()
}
