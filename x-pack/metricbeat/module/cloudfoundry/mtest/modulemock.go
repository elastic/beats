// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package mtest

import (
	"fmt"

	"github.com/cloudfoundry/sonde-go/events"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	cfcommon "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/cloudfoundry"
)

// ModuleMock is a Module with a mocked hub
type ModuleMock struct {
	cloudfoundry.Module

	Hub *HubMock
}

// NewModuleMock creates a mocked module. It contains a mocked hub that can be used to
// send envelopes for testing.
func NewModuleMock(base mb.BaseModule) (mb.Module, error) {
	module := ModuleMock{}
	factory := func(*cfcommon.Config, string, *logp.Logger) cloudfoundry.CloudfoundryHub {
		if module.Hub == nil {
			module.Hub = NewHubMock()
		}
		return module.Hub
	}
	m, err := cloudfoundry.NewModuleWithHubFactory(base, factory)
	if err != nil {
		return nil, err
	}

	module.Module = m.(cloudfoundry.Module)
	return &module, nil
}

// HubMock is a mocked hub, it can be used to send envelopes for testing.
type HubMock struct {
	envelopes chan *events.Envelope
}

// NewHubMock creates a mocked hub, it cannot be shared between metricsets.
func NewHubMock() *HubMock {
	return &HubMock{
		envelopes: make(chan *events.Envelope),
	}
}

// SendEnvelope is the main method to be used on testing, it sends an envelope through the hub.
func (h *HubMock) SendEnvelope(envelope *events.Envelope) {
	h.envelopes <- envelope
}

// DopplerConsumer creates a doppler consumer for testing, this consumer receives the events
// sent with `SendEnvelope()`.
func (h *HubMock) DopplerConsumer(cbs cfcommon.DopplerCallbacks) (cloudfoundry.DopplerConsumer, error) {
	return &MockedDopplerConsumer{h, cbs}, nil
}

// RlpListener creates a RLP listener for testing, this consumer receives the events
// sent with `SendEnvelope()`.
func (h *HubMock) RlpListener(cbs cfcommon.RlpListenerCallbacks) (cloudfoundry.RlpListener, error) {
	return nil, fmt.Errorf("mocked hub doesn't support RLP yet: not implemented")
}

// MokedDopplerConsumer is a mocked doppler consumer, it receives events sent through a mocked hub.
// It only supports the "Metrics" callback.
type MockedDopplerConsumer struct {
	hub *HubMock
	cbs cfcommon.DopplerCallbacks
}

// Run runs the doppler consumer.
// Only supports the metrics callback, what is enough for Metricbeat.
// To generalize it a dispatching mechanism should be implemented.
func (c *MockedDopplerConsumer) Run() {
	go func() {
		for envelope := range c.hub.envelopes {
			c.cbs.Metric(cfcommon.EnvelopeToEvent(envelope))
		}
	}()
}

// Stop stops the doppler consumer and the hub it uses.
func (c *MockedDopplerConsumer) Stop() {
	close(c.hub.envelopes)
}
