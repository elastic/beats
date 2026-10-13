// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
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
	// Publish returns whether the lease accepted this publication attempt.
	// Queue admission remains best effort; dropped.output_full records losses.
	Publish(queryName, responseID, spaceID string, profile map[string]any) bool
}

type scheduledProfileSink struct {
	mu        sync.Mutex
	client    beat.Client
	namespace string
	retired   atomic.Bool
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
	client, err := p.b.Publisher.ConnectWith(beat.ClientConfig{
		PublishMode: beat.DropIfFull,
		Processing:  beat.ProcessingConfig{Processor: procs},
		ClientListener: &scheduledProfileListener{dropped: func() {
			p.scheduledProfileDropped.Inc()
			now := time.Now().UnixNano()
			previous := p.scheduledProfileWarning.Load()
			if now-previous >= int64(time.Minute) && p.scheduledProfileWarning.CompareAndSwap(previous, now) {
				p.log.Warn("Scheduled query profile dropped because the publisher queue is full; see osquerybeat.scheduled_query_profiles.dropped.output_full")
			}
		}},
	})
	if err != nil {
		return err
	}
	namespace := result.Datastream.Namespace
	if namespace == "" {
		namespace = config.DefaultNamespace
	}
	p.scheduledProfileMu.Lock()
	p.scheduledProfile = &scheduledProfileSink{client: client, namespace: namespace}
	p.scheduledProfileMu.Unlock()
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

func (s *scheduledProfileSink) retire() {
	s.retired.Store(true)
	// Do not wait behind publication to invalidate this policy. The publisher
	// owns closing its client if retirement overlaps an active call.
	if s.mu.TryLock() {
		defer s.mu.Unlock()
		s.closeClient()
	}
}

func (s *scheduledProfileSink) closeClient() {
	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
}

func (s *scheduledProfileSink) Publish(queryName, responseID, spaceID string, profile map[string]any) bool {
	s.mu.Lock()
	defer func() {
		if s.retired.Load() {
			s.closeClient()
		}
		s.mu.Unlock()
	}()
	if s.retired.Load() {
		return false
	}
	s.client.Publish(beat.Event{
		Timestamp: time.Now(),
		Fields:    queryProfileToEvent(queryName, "", responseID, spaceID, profile, nil),
		Meta:      mapstr.M{events.FieldMetaRawIndex: config.QueryProfileDatastream(s.namespace)},
	})
	return true
}

type scheduledProfileListener struct{ dropped func() }

func (*scheduledProfileListener) Closing()   {}
func (*scheduledProfileListener) Closed()    {}
func (*scheduledProfileListener) NewEvent()  {}
func (*scheduledProfileListener) Filtered()  {}
func (*scheduledProfileListener) Published() {}
func (l *scheduledProfileListener) DroppedOnPublish(beat.Event) {
	l.dropped()
}
