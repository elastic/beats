// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// ScheduledProfileSink is a lease on one policy's scheduled-profile stream.
// A retired lease rejects publication, including after its namespace is reused.
type ScheduledProfileSink interface {
	Valid() bool
	// Publish returns false when the lease is retired or the output queue is full.
	// Publication is best effort. On the OTel runtime's slab queue it never waits
	// for queue space; on the memory queue it can wait for a slot, as scheduled
	// result publishes already do.
	Publish(queryName, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any) bool
}

// scheduledProfileSink publishes through a DropIfFull client. The slab queue
// never makes that Publish wait; the memory queue can, so Publish may wait for
// a slot there. A waiting Publish holds the client's own mutex, so retire must
// not close the client then; that Publish closes it when it returns.
type scheduledProfileSink struct {
	mu     sync.Mutex
	client beat.Client
	// publishing is set while a Publish uses client; only one may at a time.
	publishing bool

	namespace string
	// countDrop records a profile dropped because the output queue is full.
	countDrop func()

	retired atomic.Bool
	// drops counts events the client rejected; Publish compares it across a call.
	drops atomic.Uint64
}

// CaptureScheduledProfileSink preserves the destination used by this result.
// It never acquires the mutex used by result or action publication.
func (p *Publisher) CaptureScheduledProfileSink(namespace string) ScheduledProfileSink {
	p.scheduledProfileMu.Lock()
	defer p.scheduledProfileMu.Unlock()
	if p.scheduledProfile == nil {
		if !p.scheduledProfileMissing {
			p.log.Info("Query profile stream is not configured. Scheduled query profile events will be dropped.")
			p.scheduledProfileMissing = true
		}
		return nil
	}
	if p.scheduledProfile.namespace != namespace {
		return nil
	}
	return p.scheduledProfile
}

func (p *Publisher) configureScheduledProfileSink(result, profile config.InputConfig) error {
	procs, err := p.processorsForInputConfig(profile, config.DefaultQueryProfileDataset)
	if err != nil {
		return err
	}
	namespace := result.Datastream.Namespace
	if namespace == "" {
		namespace = config.DefaultNamespace
	}
	sink := &scheduledProfileSink{namespace: namespace, countDrop: func() {
		p.scheduledProfileDropped.Inc()
		now := time.Now().UnixNano()
		previous := p.scheduledProfileWarning.Load()
		if now-previous >= int64(time.Minute) && p.scheduledProfileWarning.CompareAndSwap(previous, now) {
			p.log.Warn("Scheduled query profile dropped because the publisher queue is full; see osquerybeat.scheduled_query_profiles.dropped.output_full")
		}
	}}
	client, err := p.b.Publisher.ConnectWith(beat.ClientConfig{
		PublishMode: beat.DropIfFull,
		Processing:  beat.ProcessingConfig{Processor: procs},
		ClientListener: scheduledProfileListener(func() {
			sink.drops.Add(1)
			sink.countDrop()
		}),
	})
	if err != nil {
		return err
	}
	sink.client = client
	p.scheduledProfileMu.Lock()
	defer p.scheduledProfileMu.Unlock()
	if p.scheduledProfileClosed {
		_ = client.Close()
		return errors.New("scheduled profile publisher is closed")
	}
	p.scheduledProfile = sink
	return nil
}

func (p *Publisher) retireScheduledProfileSink() {
	p.scheduledProfileMu.Lock()
	sink := p.scheduledProfile
	p.scheduledProfile = nil
	p.scheduledProfileMissing = false
	p.scheduledProfileMu.Unlock()
	if sink != nil {
		sink.retire()
	}
}

func (s *scheduledProfileSink) Valid() bool {
	return !s.retired.Load()
}

// retire never waits for a Publish. If one is in flight, that Publish closes
// the client when it returns.
func (s *scheduledProfileSink) retire() {
	s.retired.Store(true)
	s.mu.Lock()
	client := s.client
	if s.publishing {
		client = nil
	} else {
		s.client = nil
	}
	s.mu.Unlock()
	if client != nil {
		_ = client.Close()
	}
}

// Publish hands the profile to the client and reports false when the lease is
// retired or the client dropped the event because its queue is full.
func (s *scheduledProfileSink) Publish(queryName, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any) bool {
	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    queryProfileToEvent(queryName, "", responseID, spaceID, profile, nil, scheduleID, scheduleExecutionCount),
		Meta:      mapstr.M{events.FieldMetaRawIndex: config.QueryProfileDatastream(s.namespace)},
	}
	s.mu.Lock()
	if s.retired.Load() || s.client == nil {
		s.mu.Unlock()
		return false
	}
	if s.publishing {
		// An earlier Publish is waiting for a queue slot. Waiting behind it
		// would strand one more collector per daemon restart.
		s.mu.Unlock()
		s.countDrop()
		return false
	}
	s.publishing = true
	client := s.client
	s.mu.Unlock()

	before := s.drops.Load()
	client.Publish(event)
	ok := s.drops.Load() == before

	s.mu.Lock()
	s.publishing = false
	if s.retired.Load() {
		// retire left the client to this call.
		s.client = nil
	} else {
		client = nil
	}
	s.mu.Unlock()
	if client != nil {
		_ = client.Close()
	}
	return ok
}

type scheduledProfileListener func()

func (scheduledProfileListener) Closing()   {}
func (scheduledProfileListener) Closed()    {}
func (scheduledProfileListener) NewEvent()  {}
func (scheduledProfileListener) Filtered()  {}
func (scheduledProfileListener) Published() {}
func (l scheduledProfileListener) DroppedOnPublish(beat.Event) {
	l()
}
