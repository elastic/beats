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
	"time"

	"github.com/joeshaw/multierror"
	"go.uber.org/zap/zapcore"

	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// unitKey is used to identify a unique unit in a map
// the `ID` of a unit in itself is not unique without its type, only `Type` + `ID` is unique
type unitKey struct {
	Type client.UnitType
	ID   string
}

// BeatV2Manager is the main type for tracing V2-related config updates
type BeatV2Manager struct {
	config   *Config
	registry *reload.Registry
	client   client.V2

	logger *logp.Logger

	// track individual units given to us by the V2 API
	mx      sync.Mutex
	units   map[unitKey]*client.Unit
	actions []client.Action

	// status is reported as a whole for every unit sent to this component
	// hopefully this can be improved in the future to be seperated per unit
	status  lbmanagement.Status
	message string
	payload map[string]interface{}

	// stop callback must be registered by libbeat, as with the V1 callback
	stopFunc           func()
	stopOnOutputReload bool
	stopMut            sync.Mutex
	beatStop           sync.Once

	// sync channel for shutting down the manager after we get a stop from
	// either the agent or the beat
	stopChan chan struct{}

	isRunning bool

	// is set on first instance of a config reload,
	// allowing us to restart the beat if stopOnOutputReload is set
	outputIsConfigured bool

	// used for the debug callback to report as-running config
	lastOutputCfg *reload.ConfigWithMeta
	lastInputCfg  []*reload.ConfigWithMeta
}

// ================================
// Init Functions
// ================================

// NewV2AgentManager returns a remote config manager for the agent V2 protocol.
// This is meant to be used by the management plugin system, which will register this as a callback.
func NewV2AgentManager(config *conf.C, registry *reload.Registry, _ uuid.UUID) (lbmanagement.Manager, error) {
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
	if config.OutputRestart {
		log.Infof("Output reload is enabled, the beat will restart as needed on change of output config")
	}
	m := &BeatV2Manager{
		stopOnOutputReload: config.OutputRestart,
		config:             config,
		logger:             log.Named("V2-manager"),
		registry:           registry,
		units:              make(map[unitKey]*client.Unit),
		status:             lbmanagement.Running,
		message:            "Healthy",
		stopChan:           make(chan struct{}, 1),
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
	cm.mx.Lock()
	defer cm.mx.Unlock()

	cm.status = status
	cm.message = msg
	cm.updateStatuses()
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

	cm.client.RegisterDiagnosticHook("beat-rendered-config", "the rendered config used by the beat", "beat-rendered-config.yml", "application/yaml", cm.handleDebugYaml)
	go cm.unitListen()
	cm.isRunning = true
	return nil
}

// Stop stops the current Manager and close the connection to Elastic Agent.
func (cm *BeatV2Manager) Stop() {
	cm.stopChan <- struct{}{}
}

// CheckRawConfig is currently not implemented for V1.
func (cm *BeatV2Manager) CheckRawConfig(_ *conf.C) error {
	// This does not do anything on V1 or V2, but here we are
	return nil
}

// RegisterAction adds a V2 client action
func (cm *BeatV2Manager) RegisterAction(action client.Action) {
	cm.mx.Lock()
	defer cm.mx.Unlock()

	cm.actions = append(cm.actions, action)
	for _, unit := range cm.units {
		// actions are only registered on input units (not a requirement by Agent but
		// don't see a need in beats to support actions on an output at the moment)
		if unit.Type() == client.UnitTypeInput {
			unit.RegisterAction(action)
		}
	}
}

// UnregisterAction removes a V2 client action
func (cm *BeatV2Manager) UnregisterAction(action client.Action) {
	cm.mx.Lock()
	defer cm.mx.Unlock()

	// remove the registered action
	i := func() int {
		for i, a := range cm.actions {
			if a.Name() == action.Name() {
				return i
			}
		}
		return -1
	}()
	if i == -1 {
		// not registered
		return
	}
	cm.actions = append(cm.actions[:i], cm.actions[i+1:]...)

	for _, unit := range cm.units {
		// actions are only registered on input units (not a requirement by Agent but
		// don't see a need in beats to support actions on an output at the moment)
		if unit.Type() == client.UnitTypeInput {
			unit.UnregisterAction(action)
		}
	}
}

// SetPayload sets the global payload for the V2 client
func (cm *BeatV2Manager) SetPayload(payload map[string]interface{}) {
	cm.mx.Lock()
	defer cm.mx.Unlock()

	cm.payload = payload
	cm.updateStatuses()
}

// updateStatuses updates the status for all units to match the status of the entire manager.
//
// This is done because beats at the moment cannot manage different status per unit, something
// that is new in the V2 control protocol but not supported in beats itself.
func (cm *BeatV2Manager) updateStatuses() {
	status := getUnitState(cm.status)
	message := cm.message
	payload := cm.payload

	for _, unit := range cm.units {
		state, _, _ := unit.Expected()
		if state == client.UnitStateStopped {
			// unit is expected to be stopping (don't adjust the state as the state is now managed by the
			// `reload` method and will be marked stopped in that code path)
			continue
		}
		err := unit.UpdateState(status, message, payload)
		if err != nil {
			cm.logger.Errorf("Failed to update unit %s status: %s", unit.ID(), err)
		}
	}
}

// ================================
// Unit manager
// ================================

func (cm *BeatV2Manager) addUnit(unit *client.Unit) {
	cm.mx.Lock()
	defer cm.mx.Unlock()
	cm.units[unitKey{unit.Type(), unit.ID()}] = unit

	// update specific unit to starting
	_ = unit.UpdateState(client.UnitStateStarting, "Starting", nil)

	// register the already registered actions (only on input units)
	for _, action := range cm.actions {
		unit.RegisterAction(action)
	}
}

func (cm *BeatV2Manager) modifyUnit(unit *client.Unit) {
	// `unit` is already in `cm.units` no need to add it to the map again
	// but the lock still needs to be help so reload can be triggered
	cm.mx.Lock()
	defer cm.mx.Unlock()

	state, _, _ := unit.Expected()
	if state == client.UnitStateStopped {
		// expected to be stopped; needs to stop this unit
		_ = unit.UpdateState(client.UnitStateStopping, "Stopping", nil)
	} else {
		// update specific unit to configuring
		_ = unit.UpdateState(client.UnitStateConfiguring, "Configuring", nil)
	}
}

func (cm *BeatV2Manager) deleteUnit(unit *client.Unit) {
	// a unit will only be deleted once it has reported stopped so nothing
	// more needs to be done other than cleaning up the reference to the unit
	cm.mx.Lock()
	defer cm.mx.Unlock()
	delete(cm.units, unitKey{unit.Type(), unit.ID()})
}

// ================================
// Private V2 implementation
// ================================

func (cm *BeatV2Manager) unitListen() {
	const changeDebounce = 100 * time.Millisecond

	// register signal handler
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// timer is used to provide debounce on unit changes
	// this allows multiple changes to come in and only a single reload be performed
	t := time.NewTimer(changeDebounce)
	t.Stop() // starts stopped, until a change occurs

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
			cm.UpdateStatus(lbmanagement.Stopping, "Stopping")
			return
		case change := <-cm.client.UnitChanges():
			switch change.Type {
			// Within the context of how we send config to beats, I'm not sure there is a difference between
			// A unit add and a unit change, since either way we can't do much more than call the reloader
			case client.UnitChangedAdded:
				cm.addUnit(change.Unit)
				t.Reset(changeDebounce)
			case client.UnitChangedModified:
				cm.modifyUnit(change.Unit)
				t.Reset(changeDebounce)
			case client.UnitChangedRemoved:
				cm.deleteUnit(change.Unit)
			}
		case <-t.C:
			// a copy of the units is used for reload to prevent the holding of the `cm.mx`.
			// it could be possible that sending the configuration to reload could cause the `UpdateStatus`
			// to be called on the manager causing it to try and grab the `cm.mx` lock, causing a deadlock.
			cm.mx.Lock()
			units := make(map[unitKey]*client.Unit, len(cm.units))
			for k, u := range cm.units {
				units[k] = u
			}
			cm.mx.Unlock()
			cm.reload(units)
		}

	}
}

func (cm *BeatV2Manager) stopBeat() {
	if !cm.isRunning {
		return
	}
	cm.logger.Debugf("Stopping beat")
	cm.UpdateStatus(lbmanagement.Stopping, "Stopping")

	cm.isRunning = false
	cm.stopMut.Lock()
	defer cm.stopMut.Unlock()
	if cm.stopFunc != nil {
		// I'm not 100% sure the once here is needed,
		// but various beats tend to handle this in a not-quite-safe way
		cm.beatStop.Do(cm.stopFunc)
	}
	cm.client.Stop()
	cm.UpdateStatus(lbmanagement.Stopped, "Stopped")
}

func (cm *BeatV2Manager) reload(units map[unitKey]*client.Unit) {
	lowestLevel := client.UnitLogLevelError
	var outputUnit *client.Unit
	var inputUnits []*client.Unit
	var stoppingUnits []*client.Unit
	for _, unit := range units {
		state, ll, _ := unit.Expected()
		if state == client.UnitStateStopped {
			// unit is being stopped
			//
			// we keep the unit so after reload is performed
			// these units can be marked as stopped
			stoppingUnits = append(stoppingUnits, unit)
			continue
		} else if state != client.UnitStateHealthy {
			// only stopped or healthy are known (and expected) state
			// for a unit
			cm.logger.Errorf("unit %s has an unknown state %+v", unit.ID(), state)
		}
		if ll > lowestLevel {
			lowestLevel = ll
		}
		if unit.Type() == client.UnitTypeOutput {
			outputUnit = unit
		} else if unit.Type() == client.UnitTypeInput {
			inputUnits = append(inputUnits, unit)
		} else {
			cm.logger.Errorf("unit %s as an unknown type %+v", unit.ID(), unit.Type())
		}
	}

	// set the new log level (if nothing has changed is a noop)
	logp.SetLevel(getZapcoreLevel(lowestLevel))

	// reload the output configuration
	var errs multierror.Errors
	if err := cm.reloadOutput(outputUnit); err != nil {
		errs = append(errs, err)
	}

	// compute the input configuration
	//
	// in v2 only a single input type will be started per component, so we don't need to
	// worry about getting multiple re-loaders (we just need the one for the type)
	if err := cm.reloadInputs(inputUnits); err != nil {
		errs = append(errs, err)
	}

	// report the stopping units as stopped
	for _, unit := range stoppingUnits {
		_ = unit.UpdateState(client.UnitStateStopped, "Stopped", nil)
	}

	// any error during reload changes the whole state of the beat to failed
	if len(errs) > 0 {
		cm.status = lbmanagement.Failed
		cm.message = fmt.Sprintf("%s", errs)
	}

	// now update the statuses of all units
	cm.mx.Lock()
	status := getUnitState(cm.status)
	message := cm.message
	payload := cm.payload
	cm.mx.Unlock()
	for _, unit := range units {
		state, _, _ := unit.Expected()
		if state == client.UnitStateStopped {
			// unit is expected to be stopping (don't adjust the state as the state is now managed by the
			// `reload` method and will be marked stopped in that code path)
			continue
		}
		err := unit.UpdateState(status, message, payload)
		if err != nil {
			cm.logger.Errorf("Failed to update unit %s status: %s", unit.ID(), err)
		}
	}
}

func (cm *BeatV2Manager) reloadOutput(unit *client.Unit) error {
	// Assuming that the output reloadable isn't a list, see createBeater() in cmd/instance/beat.go
	output := cm.registry.GetReloadableOutput()
	if output == nil {
		return fmt.Errorf("failed to find beat reloadable type 'output'")
	}

	if unit == nil {
		// output is being stopped
		err := output.Reload(nil)
		if err != nil {
			return fmt.Errorf("failed to reload output: %w", err)
		}
		cm.lastOutputCfg = nil
		return nil
	}

	if cm.stopOnOutputReload && cm.outputIsConfigured {
		cm.logger.Info("beat is restarting because output changed")
		_ = unit.UpdateState(client.UnitStateStopping, "Restarting", nil)
		cm.Stop()
		return nil
	}

	_, _, rawConfig := unit.Expected()
	if rawConfig == nil {
		return fmt.Errorf("output unit has no config")
	}
	cm.logger.Debugf("Got output unit config '%s'", rawConfig.GetId())

	reloadConfig, err := groupByOutputs(rawConfig)
	if err != nil {
		return fmt.Errorf("failed to generate config for output: %w", err)
	}

	err = output.Reload(reloadConfig)
	if err != nil {
		return fmt.Errorf("failed to reload output: %w", err)
	}
	cm.lastOutputCfg = reloadConfig
	// set to true, we'll reload the output if we need to re-configure
	cm.outputIsConfigured = true
	return nil
}

func (cm *BeatV2Manager) reloadInputs(inputUnits []*client.Unit) error {
	obj := cm.registry.GetInputList()
	if obj == nil {
		return fmt.Errorf("failed to find beat reloadable type 'input'")
	}

	var inputCfgs []*reload.ConfigWithMeta
	agentInfo := cm.client.AgentInfo()
	for _, unit := range inputUnits {
		_, _, rawConfig := unit.Expected()
		inputCfg, err := generateBeatConfig(rawConfig, agentInfo)
		if err != nil {
			return fmt.Errorf("failed to generate configuration for unit %s: %w", unit.ID(), err)
		}
		inputCfgs = append(inputCfgs, inputCfg...)
	}

	err := obj.Reload(inputCfgs)
	if err != nil {
		return fmt.Errorf("failed to reloading inputs: %w", err)
	}
	cm.lastInputCfg = inputCfgs
	return nil
}

// this function is registered as a debug hook
// it prints the last known configuration genreated by the beat
func (cm *BeatV2Manager) handleDebugYaml() []byte {
	// generate input
	inputList := []map[string]interface{}{}
	for _, module := range cm.lastInputCfg {
		var inputMap map[string]interface{}
		err := module.Config.Unpack(&inputMap)
		if err != nil {
			cm.logger.Errorf("error unpacking input config for debug callback: %s", err)
			return nil
		}
		inputList = append(inputList, inputMap)
	}

	// generate output
	outputCfg := map[string]interface{}{}
	if cm.lastOutputCfg != nil {
		err := cm.lastOutputCfg.Config.Unpack(&outputCfg)
		if err != nil {
			cm.logger.Errorf("error unpacking output config for debug callback: %s", err)
			return nil
		}
	}
	// combine the two in a somewhat coherent way
	// This isn't perfect, but generating a config that can actually be fed back into the beat
	// would require
	beatCfg := struct {
		Inputs  []map[string]interface{}
		Outputs map[string]interface{}
	}{
		Inputs:  inputList,
		Outputs: outputCfg,
	}

	data, err := yaml.Marshal(beatCfg)
	if err != nil {
		cm.logger.Errorf("error generating YAML for input debug callback: %w", err)
		return nil
	}
	return data
}

func getUnitState(status lbmanagement.Status) client.UnitState {
	switch status {
	case lbmanagement.Unknown:
		// must be started if its unknown
		return client.UnitStateStarting
	case lbmanagement.Starting:
		return client.UnitStateStarting
	case lbmanagement.Configuring:
		return client.UnitStateConfiguring
	case lbmanagement.Running:
		return client.UnitStateHealthy
	case lbmanagement.Degraded:
		return client.UnitStateDegraded
	case lbmanagement.Failed:
		return client.UnitStateFailed
	case lbmanagement.Stopping:
		return client.UnitStateStopping
	case lbmanagement.Stopped:
		return client.UnitStateStopped
	}
	// unknown again?
	return client.UnitStateStarting
}

func getZapcoreLevel(ll client.UnitLogLevel) zapcore.Level {
	switch ll {
	case client.UnitLogLevelError:
		return zapcore.ErrorLevel
	case client.UnitLogLevelWarn:
		return zapcore.WarnLevel
	case client.UnitLogLevelInfo:
		return zapcore.InfoLevel
	case client.UnitLogLevelDebug:
		return zapcore.DebugLevel
	case client.UnitLogLevelTrace:
		// beats doesn't support trace
		return zapcore.DebugLevel
	}
	// info level for fallback
	return zapcore.InfoLevel
}
