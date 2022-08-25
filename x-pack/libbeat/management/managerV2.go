// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// BeatV2Manager is the main type for tracing V2-related config updates
type BeatV2Manager struct {
	config   *Config
	registry *reload.Registry
	client   client.V2

	logger *logp.Logger

	// Track individual units given to us by the V2 API
	unitsMut sync.Mutex
	units    map[string]*client.Unit
	mainUnit string

	// This satisfies the SetPayload() function, and will pass along this value to the UpdateStatus()
	// call whenever a config is re-registered
	payload map[string]interface{}

	// stop callback must be registered by libbeat, as with the V1 callback
	stopFunc func()
	stopMut  sync.Mutex
	beatStop sync.Once

	// sync channel for shutting down the manager after we get a stop from
	// either the agent or the beat
	stopChan chan struct{}

	isRunning bool
}

// NewV2AgentManager returns a remote config manager for the agent V2 protocol.
// This is meant to be used by the management plugin system, which will register this as a callback.
func NewV2AgentManager(config *conf.C, registry *reload.Registry, beatUUID uuid.UUID) (lbmanagement.Manager, error) {
	c := DefaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, fmt.Errorf("parsing fleet management settings: %w", err)
		}
	}
	agentClient, _, err := client.NewV2FromReader(os.Stdin, client.VersionInfo{
		Name:    "beat-v2-client",
		Version: version.GetDefaultVersion(),
		Meta: map[string]string{
			"commit":     version.Commit(),
			"build_time": version.BuildTime().String(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error reading control config from agent: %w", err)
	}

	return NewV2AgentManagerWithClient(c, registry, agentClient)
}

// NewV2AgentManagerWithClient actually creates the manager instance used by the rest of the beats.
func NewV2AgentManagerWithClient(config *Config, registry *reload.Registry, agentClient client.V2) (lbmanagement.Manager, error) {
	log := logp.NewLogger(lbmanagement.DebugK)
	m := &BeatV2Manager{
		config:   config,
		logger:   log.Named("V2-manager"),
		registry: registry,
		units:    make(map[string]*client.Unit),
		stopChan: make(chan struct{}, 1),
	}

	if config.Enabled {
		m.client = agentClient
	}
	return m, nil
}

// ================================
// Beats central management interface implementation
// ================================

// UpdateStatus updates the manager with the current status for the beat.
func (cm *BeatV2Manager) UpdateStatus(status lbmanagement.Status, msg string) {
	updateState := client.UnitState(status)
	stateUnit, exists := cm.getMainUnit()
	cm.logger.Debugf("Updating beat status: %s", msg)
	if exists {
		_ = stateUnit.UpdateState(updateState, msg, cm.payload)
	} else {
		cm.logger.Warnf("Cannot update state to %s, no main unit is set. Msg: %s", status, msg)
	}
}

// Enabled returns true if config management is enabled.
func (cm *BeatV2Manager) Enabled() bool {
	return cm.config.Enabled
}

// SetStopCallback sets the callback to run when the manager want to shutdown the beats gracefully.
func (cm *BeatV2Manager) SetStopCallback(stopFunc func()) {
	cm.stopMut.Lock()
	defer cm.stopMut.Unlock()
	cm.stopFunc = stopFunc
}

// Start the config manager.
func (cm *BeatV2Manager) Start() error {
	if !cm.Enabled() {
		return fmt.Errorf("V2 Manager is disabled")
	}
	err := cm.client.Start(context.Background())
	if err != nil {
		return fmt.Errorf("error starting connection to client")
	}

	go cm.unitListen()
	cm.isRunning = true
	return nil
}

// Stop stops the current Manager and close the connection to Elastic Agent.
func (cm *BeatV2Manager) Stop() {
	cm.stopChan <- struct{}{}
}

// CheckRawConfig is currently not implemented for V1.
func (cm *BeatV2Manager) CheckRawConfig(cfg *conf.C) error {
	// This does not do anything on V1 or V2, but here we are
	return nil
}

func (cm *BeatV2Manager) RegisterAction(action client.Action) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	stateUnit, exists := cm.units[cm.mainUnit]
	if exists {
		_ = stateUnit.UpdateState(client.UnitStateHealthy, fmt.Sprintf("Registering action %s for main unit with ID %s", cm.mainUnit, action.Name()), nil)
		cm.units[cm.mainUnit].RegisterAction(action)
	} else {
		cm.logger.Warnf("Cannot register action %s, no main unit found", action.Name())
	}
}

func (cm *BeatV2Manager) UnregisterAction(action client.Action) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	stateUnit, exists := cm.units[cm.mainUnit]
	if exists {
		_ = stateUnit.UpdateState(client.UnitStateHealthy, fmt.Sprintf("Unregistering action %s for main unit with ID %s", cm.mainUnit, action.Name()), nil)
		cm.units[cm.mainUnit].UnregisterAction(action)
	} else {
		cm.logger.Warnf("Cannot Unregister action %s, no main unit found", action.Name())
	}
}

func (cm *BeatV2Manager) SetPayload(payload map[string]interface{}) {
	cm.payload = payload
}

// ================================
// Unit manager
// ================================

func (cm *BeatV2Manager) addUnit(unit *client.Unit) {
	cm.unitsMut.Lock()
	cm.units[unit.ID()] = unit
	cm.unitsMut.Unlock()
}

func (cm *BeatV2Manager) getMainUnit() (*client.Unit, bool) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	if cm.mainUnit == "" {
		return nil, false
	}
	return cm.units[cm.mainUnit], true
}

// We need a "main" unit that we can send updates to for the StatusReporter interface
// the purpose of this is to just grab the first input-type unit we get and set it as the "main" unit
func (cm *BeatV2Manager) setMainUnitValue(unit *client.Unit) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	if cm.mainUnit == "" {
		cm.logger.Debugf("Set main input unit to ID %s", unit.ID)
		cm.mainUnit = unit.ID()
	}
}

func (cm *BeatV2Manager) deleteUnit(unit *client.Unit) {
	cm.unitsMut.Lock()
	delete(cm.units, unit.ID())
	cm.unitsMut.Unlock()
}

// ================================
// Private V2 implementation
// ================================

func (cm *BeatV2Manager) unitListen() {

	// register signal handler
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	cm.logger.Debugf("Listening for agent unit changes")
	for {
		select {
		// The stopChan channel comes from the Manager interface Stop() method
		case <-cm.stopChan:
			cm.stopBeat()
		case sig := <-sigc:
			// we can't duplicate the same logic used by stopChan here.
			// A beat will also watch for sigint and shut down, if we call the stopFunc
			// callback, either the V2 client or the beat will get a panic,
			// as the stopFunc sent by the beats is usually unsafe.
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				cm.logger.Debug("Received sigterm/sigint, stopping")
			case syscall.SIGHUP:
				cm.logger.Debug("Received sighup, stopping")
			}
			cm.isRunning = false
			unit, mainExists := cm.getMainUnit()
			if mainExists {
				_ = unit.UpdateState(client.UnitStateStopping, "stopping beat", nil)
			}
			cm.client.Stop()
			return
		case change := <-cm.client.UnitChanges():
			switch change.Type {
			// Within the context of how we send config to beats, I'm not sure there is a difference between
			// A unit add and a unit change, since either way we can't do much more than call the reloader
			case client.UnitChangedAdded:
				// At this point we also get a log level, however I'm not sure the beats core logger provides a
				// clean way to "just" change the log level, without resetting the whole log config
				state, _, _ := change.Unit.Expected()
				cm.logger.Debugf("Got unit added: %s, type: %s expected state: %s", change.Unit.ID(), change.Unit.Type(), state.String())
				go cm.handleUnitReload(change.Unit)

			case client.UnitChangedModified:
				state, _, _ := change.Unit.Expected()
				cm.logger.Debugf("Got unit modified: %s, type: %s expected state: %s", change.Unit.ID(), change.Unit.Type(), state.String())
				// I'm assuming that a state STOPPED just tells us to shut down the entire beat,
				// as such we don't really care about updating via a particular unit
				if state == client.UnitStateStopped {
					cm.stopBeat()
				} else {
					go cm.handleUnitReload(change.Unit)
				}

			case client.UnitChangedRemoved:
				cm.logger.Debugf("Got unit removed: %s", change.Unit.ID())
				cm.deleteUnit(change.Unit)
			}
		}

	}
}

func (cm *BeatV2Manager) stopBeat() {
	if !cm.isRunning {
		return
	}
	// will we ever get a Unit removed for anything other than the main beat?
	// Individual reloaders don't have a "stop" function, so the most we can do
	// is just shut down a beat, I think.
	cm.logger.Debugf("Stopping beat")
	// stop the "main" beat runtime
	unit, mainExists := cm.getMainUnit()
	if mainExists {
		_ = unit.UpdateState(client.UnitStateStopping, "stopping beat", nil)
	}

	cm.isRunning = false
	cm.stopMut.Lock()
	defer cm.stopMut.Unlock()
	if cm.stopFunc != nil {
		// I'm not 100% sure the once here is needed,
		// but various beats tend to handle this in a not-quite-safe way
		cm.beatStop.Do(cm.stopFunc)
	}
	cm.client.Stop()

	if mainExists {
		_ = unit.UpdateState(client.UnitStateStopped, "stopped beat", nil)
	}

}

func (cm *BeatV2Manager) handleUnitReload(unit *client.Unit) {
	cm.addUnit(unit)
	unitType := unit.Type()

	if unitType == client.UnitTypeOutput {
		cm.handleOutputReload(unit)
	} else if unitType == client.UnitTypeInput {
		cm.handleInputReload(unit)
	}
}

// Handle the updated config for an output unit
func (cm *BeatV2Manager) handleOutputReload(unit *client.Unit) {
	_, _, rawConfig := unit.Expected()
	cm.logger.Debugf("Got Output unit config: %s, ID: %s", rawConfig.Type, rawConfig.Id)

	reloadConfig, err := groupByOutputs(rawConfig)
	if err != nil {
		errString := fmt.Errorf("Failed to generate config for output: %w", err)
		_ = unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
		return
	}
	// Assuming that the output reloadable isn't a list, see createBeater() in cmd/instance/beat.go
	output := cm.registry.GetReloadableOutput()
	if output == nil {
		_ = unit.UpdateState(client.UnitStateFailed, "failed to find beat reloadable type 'output'", nil)
		return
	}

	_ = unit.UpdateState(client.UnitStateConfiguring, "reloading output component", nil)
	err = output.Reload(reloadConfig)
	if err != nil {
		errString := fmt.Errorf("Failed to reload component: %w", err)
		_ = unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
		return
	}
	_ = unit.UpdateState(client.UnitStateHealthy, "reloaded output component", nil)
}

// handle the updated config for an input unit
func (cm *BeatV2Manager) handleInputReload(unit *client.Unit) {
	_, _, rawConfig := unit.Expected()
	cm.setMainUnitValue(unit)
	cm.logger.Debugf("Got Input unit config: %s, ID: %s", rawConfig.Type, rawConfig.Id)

	// Find the V2 inputs we need to reload
	// The reloader provides list and non-list types, but all the beats register as lists,
	// so just go with that for V2
	obj := cm.registry.GetInputList()
	if obj == nil {
		_ = unit.UpdateState(client.UnitStateFailed, "failed to find beat reloadable type 'input'", nil)
		return
	}
	_ = unit.UpdateState(client.UnitStateConfiguring, "found reloader for 'input'", nil)

	beatCfg, err := generateBeatConfig(rawConfig, cm.client.AgentInfo())
	if err != nil {
		errString := fmt.Errorf("Failed to create Unit config: %w", err)
		_ = unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
		return
	}

	err = obj.Reload(beatCfg)
	if err != nil {
		errString := fmt.Errorf("Error reloading input: %w", err)
		_ = unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
		return
	}
	_ = unit.UpdateState(client.UnitStateHealthy, "beat reloaded", nil)
}
