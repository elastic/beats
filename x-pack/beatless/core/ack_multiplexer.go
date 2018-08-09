// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"errors"
	"sync"

	"github.com/elastic/beats/libbeat/logp"
)

// AckMultiplexer registers itself as a pipeline AckEvents Handler and allow to dynamic add or
// remove specific ack handlers. This is useful if you have client that pushes events to the pipeline
// and need to ack to another upstream systems. The AckMultiplexer uses the beat.Event.Private field
// to have a pointer to a struct implementing the SourceAcker interface.
//
// Caveats: Users of the the AckMultiplexer are responsable to add cleanup ackers, if the ackers
// is not found for an events we will just ignore it.

// Errors returned when adding or removing ackers.
var (
	ErrAlreadyExist = errors.New("acker already exist")
	ErrUnknown      = errors.New("unknown acker")
)

// sourceAcker is the callbacks interface when new events get ACK.
type sourceAcker interface {
	AckEvents([]interface{})
}

// SourceMetadata contains metadata about the clients who emitted the events, this struct will be
// saved in the Private field in the beat.Event.
type SourceMetadata struct {
	Acker sourceAcker
}

// AckMultiplexer needs to be registered as the ACKEvents on the publisher.
type AckMultiplexer struct {
	sync.RWMutex
	ackers map[sourceAcker]sourceAcker
	log    *logp.Logger
}

// NewAckMultiplexer creates a new ack multiplexer.
func NewAckMultiplexer() *AckMultiplexer {
	return &AckMultiplexer{
		ackers: make(map[sourceAcker]sourceAcker),
		log:    logp.NewLogger("ack-multiplexer"),
	}
}

// AddAcker adds a new acker.
func (am *AckMultiplexer) AddAcker(acker sourceAcker) error {
	am.Lock()
	defer am.Unlock()
	defer am.log.Debug("acker added")

	_, found := am.ackers[acker]
	if found {
		return ErrAlreadyExist
	}

	am.ackers[acker] = acker
	return nil
}

// RemoveAcker removes the specified acker.
func (am *AckMultiplexer) RemoveAcker(acker sourceAcker) error {
	am.Lock()
	defer am.Unlock()
	defer am.log.Debug("acker removed")

	_, found := am.ackers[acker]
	if !found {
		return ErrUnknown
	}

	delete(am.ackers, acker)
	return nil
}

// AckEvents receives the global ACK and send the events down to the right client handler.
func (am *AckMultiplexer) AckEvents(data []interface{}) {
	am.log.Debugw("ack events", "count", len(data))
	ordered := make(map[sourceAcker][]interface{}, len(am.ackers))

	for _, d := range data {
		if data == nil {
			am.log.Debug("received nil data from the ACK handler")
			continue
		}

		sm, ok := d.(SourceMetadata)
		if !ok {
			am.log.Debugf("incompatible type, expecting 'SourceMetadata' and received '%T'", d)
			continue
		}

		v, _ := ordered[sm.Acker]
		v = append(v, d)
		ordered[sm.Acker] = v
	}

	am.RLock()
	defer am.RUnlock()
	for a, d := range ordered {
		acker, ok := am.ackers[a]
		if !ok {
			am.log.Debug("acker not found, ignoring")
			continue
		}
		am.log.Debugw("sending batch of ACKs to a specific acker", "count", len(d))
		acker.AckEvents(d)
	}
}
