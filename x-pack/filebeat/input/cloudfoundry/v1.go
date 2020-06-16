// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

// InputV1 defines a udp input to receive event on a specific host:port.
type InputV1 struct {
	sync.Mutex
	consumer *cloudfoundry.DopplerConsumer
	started  bool
	log      *logp.Logger
	outlet   channel.Outleter
}

func newInputV1(log *logp.Logger, conf cloudfoundry.Config, out channel.Outleter, context input.Context) (*InputV1, error) {
	hub := cloudfoundry.NewHub(&conf, "filebeat", log)
	forwarder := harvester.NewForwarder(out)

	callbacks := cloudfoundry.DopplerCallbacks{
		Log: func(evt cloudfoundry.Event) {
			forwarder.Send(beat.Event{
				Timestamp: evt.Timestamp(),
				Fields:    evt.ToFields(),
			})
		},
		Error: func(evt cloudfoundry.EventError) {
			forwarder.Send(beat.Event{
				Timestamp: evt.Timestamp(),
				Fields:    evt.ToFields(),
			})
		},
	}

	consumer, err := hub.DopplerConsumer(callbacks)
	if err != nil {
		return nil, errors.Wrapf(err, "initializing doppler consumer")
	}
	return &InputV1{
		outlet:   out,
		consumer: consumer,
		started:  false,
		log:      log,
	}, nil
}

// Run starts the consumer of cloudfoundry events
func (p *InputV1) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		p.log.Info("starting cloudfoundry input")
		p.consumer.Run()
		p.started = true
	}
}

// Stop stops cloudfoundry doppler consumer
func (p *InputV1) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	p.log.Info("stopping cloudfoundry input")
	p.consumer.Stop()
	p.started = false
}

// Wait waits for the input to finalize, and stops it
func (p *InputV1) Wait() {
	p.Stop()
	p.consumer.Wait()
}
