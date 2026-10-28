// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelmanager

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	gproto "google.golang.org/protobuf/proto"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"
	xpmanagement "github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/logp"
)

// unitsDebounce is how long the manager waits for more unit changes before
// applying them, as the process manager does.
const unitsDebounce = time.Second

// UnitTransform turns an input unit's config into beat input configs.
type UnitTransform func(*proto.UnitExpectedConfig, *client.AgentInfo) ([]*reload.ConfigWithMeta, error)

// UnitsManager is the manager for a beat receiver that gets its inputs from
// Elastic Agent as units over the control protocol. The control protocol only
// delivers configuration: everything else works as with OtelManager, so the
// receiver reports status through the collector, and diagnostics and actions go
// through the collector extensions.
//
// Unlike the process manager, it doesn't report unit state, or apply log
// levels, features or APM configuration from the units.
type UnitsManager struct {
	*OtelManager

	client    client.V2
	registry  *reload.Registry
	transform UnitTransform
	logger    *logp.Logger

	ready     chan struct{}
	readyOnce sync.Once
	stop      chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup

	// owned by the run loop
	units      map[string]*client.Unit
	lastOutput *proto.UnitExpectedConfig
	lastInputs map[string]*proto.UnitExpectedConfig

	// the configs last applied, for the beat-rendered-config diagnostic
	renderedMx     sync.Mutex
	renderedOutput *reload.ConfigWithMeta
	renderedInputs []*reload.ConfigWithMeta
	outputReady    bool
}

var _ management.Manager = (*UnitsManager)(nil)

// NewUnitsManager returns a manager that applies the units it gets from
// agentClient to the beat's reloaders in registry.
func NewUnitsManager(agentClient client.V2, registry *reload.Registry, transform UnitTransform, logger *logp.Logger) *UnitsManager {
	management.SetUnderAgent(true)
	return &UnitsManager{
		OtelManager: &OtelManager{logger: logger},
		client:      agentClient,
		registry:    registry,
		transform:   transform,
		logger:      logger.Named("units-manager"),
		ready:       make(chan struct{}),
		stop:        make(chan struct{}),
		units:       make(map[string]*client.Unit),
	}
}

// ConfigFromControlProtocol returns true: the inputs come from Elastic Agent
// over the control protocol.
func (m *UnitsManager) ConfigFromControlProtocol() bool { return true }

// AgentInfo returns the agent info Elastic Agent sent.
func (m *UnitsManager) AgentInfo() management.AgentInfo {
	info := m.client.AgentInfo()
	if info == nil {
		return management.AgentInfo{}
	}
	return management.AgentInfo{
		ID:           info.ID,
		Version:      info.Version,
		Snapshot:     info.Snapshot,
		ManagedMode:  management.AgentManagedMode(info.ManagedMode),
		Unprivileged: info.Unprivileged,
	}
}

// PreInit connects to Elastic Agent. Units are applied after PostInit.
func (m *UnitsManager) PreInit() error {
	if err := m.client.Start(context.Background()); err != nil {
		return err
	}
	// The receiver sets the diagnostics extension before running the beat, so
	// the hook reaches it. This is the same file a beat process provides.
	m.RegisterDiagnosticHook("beat-rendered-config", "the rendered config used by the beat",
		"beat-rendered-config.yml", "application/yaml", m.renderedConfig)
	m.wg.Go(m.run)
	return nil
}

// PostInit lets the manager apply units, once the beat has registered its reloaders.
func (m *UnitsManager) PostInit() {
	m.readyOnce.Do(func() { close(m.ready) })
}

// Start calls PreInit and PostInit.
func (m *UnitsManager) Start() error {
	if err := m.PreInit(); err != nil {
		return err
	}
	m.PostInit()
	return nil
}

// Stop disconnects from Elastic Agent, stops the inputs it started, and stops
// the beat. The receiver can outlive the beat in the collector, so the inputs
// must not keep running.
func (m *UnitsManager) Stop() {
	m.stopOnce.Do(func() {
		// The client blocks until its channels are read, so stop it while the
		// run loop still reads them.
		m.client.Stop()
		close(m.stop)
		m.wg.Wait()
		if list := m.registry.GetInputList(); list != nil && m.lastInputs != nil {
			if err := list.Reload(nil); err != nil {
				m.logger.Errorf("failed to stop inputs: %v", err)
			}
		}
	})
	m.OtelManager.Stop()
}

func (m *UnitsManager) run() {
	timer := time.NewTimer(unitsDebounce)
	timer.Stop()
	ready := m.ready
	for {
		select {
		case <-m.stop:
			return
		case err := <-m.client.Errors():
			if !errors.Is(err, context.Canceled) {
				m.logger.Errorf("elastic-agent-client error: %v", err)
			}
		case change := <-m.client.UnitChanges():
			key := unitKey(change.Unit)
			if change.Type == client.UnitChangedRemoved {
				delete(m.units, key)
			} else {
				m.units[key] = change.Unit
			}
			timer.Reset(unitsDebounce)
		case <-ready:
			// apply the units that arrived before the beat was ready
			ready = nil
			timer.Reset(0)
		case <-timer.C:
			if ready == nil {
				m.apply()
			}
		}
	}
}

func unitKey(u *client.Unit) string {
	return u.Type().String() + "/" + u.ID()
}

// apply passes the current units to the beat's reloaders.
func (m *UnitsManager) apply() {
	var output *proto.UnitExpectedConfig
	inputs := make(map[string]*proto.UnitExpectedConfig)
	for _, u := range m.units {
		expected := u.Expected()
		if expected.State == client.UnitStateStopped || expected.Config == nil {
			continue
		}
		switch u.Type() {
		case client.UnitTypeOutput:
			output = expected.Config
		case client.UnitTypeInput:
			inputs[u.ID()] = expected.Config
		}
	}
	m.applyOutput(output)
	m.applyInputs(inputs)
}

// applyOutput passes the output unit's config to the beat's output reloader.
// The receiver doesn't publish through it, but the beat may want to load
// config from the output, as it can in a beat process.
func (m *UnitsManager) applyOutput(output *proto.UnitExpectedConfig) {
	reloader := m.registry.GetReloadableOutput()
	if reloader == nil || (m.outputReady && gproto.Equal(output, m.lastOutput)) {
		return
	}
	var cfg *reload.ConfigWithMeta
	if output != nil {
		var err error
		if cfg, err = xpmanagement.GroupByOutputs(output); err != nil {
			m.logger.Errorf("failed to generate output config: %v", err)
			return
		}
	}
	if err := reloader.Reload(cfg); err != nil {
		m.logger.Errorf("failed to reload output: %v", err)
		return
	}
	m.lastOutput, m.outputReady = output, true
	m.renderedMx.Lock()
	m.renderedOutput = cfg
	m.renderedMx.Unlock()
}

func (m *UnitsManager) applyInputs(inputs map[string]*proto.UnitExpectedConfig) {
	list := m.registry.GetInputList()
	if list == nil {
		m.logger.Error("beat has no reloadable input list")
		return
	}
	if m.lastInputs != nil && maps.EqualFunc(m.lastInputs, inputs, func(a, b *proto.UnitExpectedConfig) bool { return gproto.Equal(a, b) }) {
		return
	}

	var configs []*reload.ConfigWithMeta
	agentInfo := m.client.AgentInfo()
	for _, id := range slices.Sorted(maps.Keys(inputs)) {
		unitConfigs, err := m.transform(inputs[id], agentInfo)
		if err != nil {
			// Inputs report their status through the collector, but this input
			// never starts, so it can only be logged.
			m.logger.Errorf("failed to generate config for input unit %s: %v", id, err)
			continue
		}
		for _, c := range unitConfigs {
			c.InputUnitID = id
		}
		configs = append(configs, unitConfigs...)
	}
	// Input errors are reported through the collector status by the inputs.
	if err := list.Reload(configs); err != nil {
		m.logger.Errorf("failed to reload inputs: %v", err)
	}
	m.lastInputs = inputs
	m.renderedMx.Lock()
	m.renderedInputs = configs
	m.renderedMx.Unlock()
}

// renderedConfig returns the configs last applied, as the process manager does
// for beat-rendered-config.
func (m *UnitsManager) renderedConfig() []byte {
	m.renderedMx.Lock()
	defer m.renderedMx.Unlock()
	data, err := xpmanagement.RenderBeatConfig(m.renderedInputs, m.renderedOutput, nil, nil)
	if err != nil {
		m.logger.Errorf("failed to render beat config for diagnostics: %v", err)
		return nil
	}
	return data
}
