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
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// ScheduledProfileSink is a lease on one policy's scheduled-profile stream.
// A retired lease rejects publication, including after its namespace is reused.
type ScheduledProfileSink interface {
	Valid() bool
	// Publish returns whether the lease was active for this publication attempt.
	// Publication is best effort: a busy publisher drops new profiles immediately.
	Publish(queryName, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any) bool
}

type scheduledProfileSink struct {
	mu        sync.Mutex
	client    beat.Client
	publisher *scheduledProfilePublisher
	namespace string
	retired   atomic.Bool
}

type scheduledProfilePublication struct {
	sink  *scheduledProfileSink
	event beat.Event
}

// scheduledProfilePublisher belongs to the Publisher, not an osqueryd run or
// policy. At most one profile is pending or in flight across all daemon restarts.
// A stuck beat.Client.Publish cannot be cancelled; it is released when the
// output recovers or the shared pipeline closes, without holding up collection.
type scheduledProfilePublisher struct {
	mu      sync.Mutex
	jobs    chan scheduledProfilePublication
	closing chan struct{}
	done    chan struct{}
	busy    bool
	closed  bool
	dropped *monitoring.Uint
}

func newScheduledProfilePublisher(dropped *monitoring.Uint) *scheduledProfilePublisher {
	p := &scheduledProfilePublisher{
		jobs:    make(chan scheduledProfilePublication, 1),
		closing: make(chan struct{}),
		done:    make(chan struct{}),
		dropped: dropped,
	}
	go p.run()
	return p
}

func (p *scheduledProfilePublisher) publish(job scheduledProfilePublication) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || !job.sink.Valid() {
		return false
	}
	if p.busy {
		p.dropped.Inc()
		return true
	}
	// busy stays set until the worker finishes, so this send always fits and
	// no second profile can queue behind a blocked publication.
	p.busy = true
	p.jobs <- job
	return true
}

func (p *scheduledProfilePublisher) run() {
	defer close(p.done)
	for {
		select {
		case <-p.closing:
			return
		case job := <-p.jobs:
			p.mu.Lock()
			closed := p.closed
			p.mu.Unlock()
			if closed {
				return
			}
			job.sink.publish(job.event)
			p.mu.Lock()
			p.busy = false
			p.mu.Unlock()
		}
	}
}

func (p *scheduledProfilePublisher) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.closed {
		p.closed = true
		close(p.closing)
		// Discard an offer that the worker has not picked up yet.
		select {
		case <-p.jobs:
		default:
		}
	}
	// Do not join a worker blocked in Publish: the shared pipeline only closes
	// after the beater returns. The worker owns closing its retired client.
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
	defer p.scheduledProfileMu.Unlock()
	if p.scheduledProfileClosed {
		_ = client.Close()
		return errors.New("scheduled profile publisher is closed")
	}
	if p.scheduledProfileWorker == nil {
		reg := p.b.Monitoring.StatsRegistry()
		if reg == nil {
			reg = monitoring.NewRegistry()
		}
		p.scheduledProfileWorker = newScheduledProfilePublisher(monitoring.NewUint(
			reg.GetOrCreateRegistry("osquerybeat.scheduled_query_profiles"), "dropped.publisher_busy"))
	}
	p.scheduledProfile = &scheduledProfileSink{client: client, publisher: p.scheduledProfileWorker, namespace: namespace}
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

func (s *scheduledProfileSink) Publish(queryName, scheduleID, responseID, spaceID string, scheduleExecutionCount int64, profile map[string]any) bool {
	return s.publisher.publish(scheduledProfilePublication{sink: s, event: beat.Event{
		Timestamp: time.Now(),
		Fields:    queryProfileToEvent(queryName, "", responseID, spaceID, profile, nil, scheduleID, scheduleExecutionCount),
		Meta:      mapstr.M{events.FieldMetaRawIndex: config.QueryProfileDatastream(s.namespace)},
	}})
}

func (s *scheduledProfileSink) publish(event beat.Event) {
	s.mu.Lock()
	defer func() {
		if s.retired.Load() {
			s.closeClient()
		}
		s.mu.Unlock()
	}()
	if s.retired.Load() {
		return
	}
	s.client.Publish(event)
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
