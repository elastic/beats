// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package cloudfoundry

import (
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	cfcommon "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
	"github.com/elastic/elastic-agent-libs/logp"
)

type ModuleV1 struct {
	mb.BaseModule

	log *logp.Logger

	running  atomic.Bool
	consumer DopplerConsumer

	events        chan cfcommon.Event
	subscriptions chan subscription
}

func newModuleV1(base mb.BaseModule, hub CloudfoundryHub, log *logp.Logger) (*ModuleV1, error) {
	m := ModuleV1{
		BaseModule: base,
		log:        log,
		running:    atomic.MakeBool(false),
	}
	consumer, err := hub.DopplerConsumer(cfcommon.DopplerCallbacks{
		Metric: m.callback,
	})
	if err != nil {
		return nil, err
	}
	m.consumer = consumer
	m.events = make(chan cfcommon.Event)
	m.subscriptions = make(chan subscription)

	return &m, nil
}

func (m *ModuleV1) RunCounterReporter(reporter mb.PushReporterV2) {
	m.subscribe(cfcommon.EventTypeCounter, reporter)
	defer m.unsubscribe(cfcommon.EventTypeCounter, reporter)
	<-reporter.Done()
}

func (m *ModuleV1) RunValueReporter(reporter mb.PushReporterV2) {
	m.subscribe(cfcommon.EventTypeValueMetric, reporter)
	defer m.unsubscribe(cfcommon.EventTypeValueMetric, reporter)
	<-reporter.Done()
}

func (m *ModuleV1) RunContainerReporter(reporter mb.PushReporterV2) {
	m.subscribe(cfcommon.EventTypeContainerMetric, reporter)
	defer m.unsubscribe(cfcommon.EventTypeContainerMetric, reporter)
	<-reporter.Done()
}

func (m *ModuleV1) subscribe(eventType cfcommon.EventType, reporter mb.PushReporterV2) {
	go m.run(subscription{
		eventType: eventType,
		reporter:  reporter,
	})
}

func (m *ModuleV1) unsubscribe(eventType cfcommon.EventType, reporter mb.PushReporterV2) {
	m.subscriptions <- subscription{
		eventType:   eventType,
		reporter:    reporter,
		unsubscribe: true,
	}
}

func (m *ModuleV1) callback(event cfcommon.Event) {
	m.events <- event
}

// run ensures that the module is running with the passed subscription
func (m *ModuleV1) run(s subscription) {
	if !m.running.CAS(false, true) {
		// Module is already running, queue subscription for current dispatcher.
		m.subscriptions <- s
		return
	}
	defer func() { m.running.Store(false) }()

	m.consumer.Run()
	defer m.consumer.Stop()

	dispatcher := newEventDispatcher(m.log)

	// Ensure that the initial subscription is configured before starting the loop,
	// this is specially relevant to make tests more deterministic.
	dispatcher.handleSubscription(s)

	for {
		// Handle subscriptions and events dispatching on the same
		// goroutine so locking is not needed.
		select {
		case e := <-m.events:
			dispatcher.dispatch(e)
		case s := <-m.subscriptions:
			dispatcher.handleSubscription(s)
			if dispatcher.empty() {
				return
			}
		}
	}
}

type subscription struct {
	eventType cfcommon.EventType
	reporter  mb.PushReporterV2

	unsubscribe bool
}

// eventDispatcher keeps track on the reporters that are subscribed to each event type
// and dispatches events to them when received.
type eventDispatcher struct {
	log       *logp.Logger
	reporters map[cfcommon.EventType]mb.PushReporterV2
}

func newEventDispatcher(log *logp.Logger) *eventDispatcher {
	return &eventDispatcher{
		log:       log,
		reporters: make(map[cfcommon.EventType]mb.PushReporterV2),
	}
}

func (d *eventDispatcher) handleSubscription(s subscription) {
	current, subscribed := d.reporters[s.eventType]
	if s.unsubscribe {
		if !subscribed || current != s.reporter {
			// This can happen if same metricset is used twice
			d.log.Warnf("Ignoring unsubscription of not subscribed reporter for %s", s.eventType)
			return
		}
		delete(d.reporters, s.eventType)
	} else {
		if subscribed {
			if s.reporter != current {
				// This can happen if same metricset is used twice
				d.log.Warnf("Ignoring subscription of multiple reporters for %s", s.eventType)
			}
			return
		}
		d.reporters[s.eventType] = s.reporter
	}
}

func (d *eventDispatcher) dispatch(e cfcommon.Event) {
	reporter, found := d.reporters[e.EventType()]
	if !found {
		return
	}
	reporter.Event(mb.Event{
		Timestamp:  e.Timestamp(),
		RootFields: e.ToFields(),
	})
}

func (d *eventDispatcher) empty() bool {
	return len(d.reporters) == 0
}
