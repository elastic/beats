// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	cfcommon "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
)

// ModuleName is the name of this module.
const ModuleName = "cloudfoundry"

type Module struct {
	mb.BaseModule

	log *logp.Logger

	hub          *cfcommon.Hub
	listener     *cfcommon.RlpListener
	listenerLock sync.Mutex

	counterReporter   mb.PushReporterV2
	valueReporter     mb.PushReporterV2
	containerReporter mb.PushReporterV2
}

func init() {
	if err := mb.Registry.AddModule(ModuleName, newModule); err != nil {
		panic(err)
	}
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	var cfg cfcommon.Config
	if err := base.UnpackConfig(&cfg); err != nil {
		return nil, err
	}

	log := logp.NewLogger("cloudfoundry")
	hub := cfcommon.NewHub(&cfg, "metricbeat", log)

	// early check that listener can be created
	_, err := hub.RlpListener(cfcommon.RlpListenerCallbacks{})
	if err != nil {
		return nil, err

	}

	return &Module{
		BaseModule: base,
		log:        log,
		hub:        hub,
	}, nil
}

func (m *Module) RunCounterReporter(reporter mb.PushReporterV2) {
	m.listenerLock.Lock()
	m.runReporters(reporter, m.valueReporter, m.containerReporter)
	m.listenerLock.Unlock()

	<-reporter.Done()

	m.listenerLock.Lock()
	m.runReporters(nil, m.valueReporter, m.containerReporter)
	m.listenerLock.Unlock()
}

func (m *Module) RunValueReporter(reporter mb.PushReporterV2) {
	m.listenerLock.Lock()
	m.runReporters(m.counterReporter, reporter, m.containerReporter)
	m.listenerLock.Unlock()

	<-reporter.Done()

	m.listenerLock.Lock()
	m.runReporters(m.counterReporter, nil, m.containerReporter)
	m.listenerLock.Unlock()
}

func (m *Module) RunContainerReporter(reporter mb.PushReporterV2) {
	m.listenerLock.Lock()
	m.runReporters(m.counterReporter, m.valueReporter, reporter)
	m.listenerLock.Unlock()

	<-reporter.Done()

	m.listenerLock.Lock()
	m.runReporters(m.counterReporter, m.valueReporter, nil)
	m.listenerLock.Unlock()
}

func (m *Module) runReporters(counterReporter, valueReporter, containerReporter mb.PushReporterV2) {
	if m.listener != nil {
		m.listener.Stop()
		m.listener = nil
	}
	m.counterReporter = counterReporter
	m.valueReporter = valueReporter
	m.containerReporter = containerReporter

	start := false
	callbacks := cfcommon.RlpListenerCallbacks{}
	if m.counterReporter != nil {
		start = true
		callbacks.Counter = func(evt *cfcommon.EventCounter) {
			m.counterReporter.Event(mb.Event{
				Timestamp:  evt.Timestamp(),
				RootFields: evt.ToFields(),
			})
		}
	}
	if m.valueReporter != nil {
		start = true
		callbacks.ValueMetric = func(evt *cfcommon.EventValueMetric) {
			m.valueReporter.Event(mb.Event{
				Timestamp:  evt.Timestamp(),
				RootFields: evt.ToFields(),
			})
		}
	}
	if m.containerReporter != nil {
		start = true
		callbacks.ContainerMetric = func(evt *cfcommon.EventContainerMetric) {
			m.containerReporter.Event(mb.Event{
				Timestamp:  evt.Timestamp(),
				RootFields: evt.ToFields(),
			})
		}
	}
	if start {
		l, err := m.hub.RlpListener(callbacks)
		if err != nil {
			m.log.Errorf("failed to create RlpListener: %v", err)
			return
		}
		l.Start(context.Background())
		m.listener = l
	}
}
