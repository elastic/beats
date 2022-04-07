// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package cloudfoundry

import (
	"context"

	cfcommon "github.com/elastic/beats/v8/x-pack/libbeat/common/cloudfoundry"
)

// DopplerConsumer is the interface that a Doppler Consumer must implement for the Cloud Foundry module.
type DopplerConsumer interface {
	Run()
	Stop()
}

// RlpListener is the interface that a RLP listener must implement for the Cloud Foundry module.
type RlpListener interface {
	Start(context.Context)
	Stop()
}

// CloudfoundryHub is the interface that a Hub must implement for the Cloud Foundry module.
type CloudfoundryHub interface {
	DopplerConsumer(cfcommon.DopplerCallbacks) (DopplerConsumer, error)
	RlpListener(cfcommon.RlpListenerCallbacks) (RlpListener, error)
}

// HubAdapter adapt a cloudfoundry Hub to the hub expected by the metricbeat module.
// This adaptation is needed to return different but compatible types, so the Hub can be mocked.
type HubAdapter struct {
	hub *cfcommon.Hub
}

func (h *HubAdapter) DopplerConsumer(cbs cfcommon.DopplerCallbacks) (DopplerConsumer, error) {
	return h.hub.DopplerConsumer(cbs)
}

func (h *HubAdapter) RlpListener(cbs cfcommon.RlpListenerCallbacks) (RlpListener, error) {
	return h.hub.RlpListener(cbs)
}
