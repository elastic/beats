// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

// InputV2 defines a Cloudfoundry input that uses the consumer V2 API
type InputV2 struct {
	sync.Mutex
	listener *cloudfoundry.RlpListener
	started  bool
	log      *logp.Logger
	outlet   channel.Outleter
}

func newInputV2(log *logp.Logger, conf cloudfoundry.Config, out channel.Outleter, context input.Context) (*InputV2, error) {
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
	return &InputV2{
		outlet:   out,
		listener: listener,
		started:  false,
		log:      log,
	}, nil
}

// Run starts the listener of cloudfoundry events
func (p *InputV2) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		p.log.Info("starting cloudfoundry input")
		p.listener.Start(context.TODO())
		p.started = true
	}
}

// Stop stops cloudfoundry listener
func (p *InputV2) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	p.log.Info("stopping cloudfoundry input")
	p.listener.Stop()
	p.started = false
}

// Wait waits for the input to finalize, and stops it
func (p *InputV2) Wait() {
	p.Stop()
}
