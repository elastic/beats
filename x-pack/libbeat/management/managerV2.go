// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joeshaw/multierror"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	gproto "google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/features"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// diagnosticHandler is a wrapper type that's a bit of a hack, the compiler won't let us send the raw unit struct,
// since there's a type disagreement with the `client.DiagnosticHook` argument, and due to licensing issues we can't import the agent client types into the reloader
type diagnosticHandler struct {
	log    *logp.Logger
	client *client.Unit
}

func (handler diagnosticHandler) Register(name string, description string, filename string, contentType string, callback func() []byte) {
	handler.log.Infof("registering callback with %s", name)
	// paranoid checking
	if handler.client != nil {
		handler.client.RegisterDiagnosticHook(name, description, filename, contentType, callback)
	} else {
		handler.log.Warnf("client handler for diag callback %s is nil", name)
	}
}

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

	// handles client errors
	errCanceller context.CancelFunc

	// track individual units given to us by the V2 API
	mx          sync.Mutex
	units       map[unitKey]*client.Unit
	actions     []client.Action
	forceReload bool

	// status is reported as a whole for every unit sent to this component
	// hopefully this can be improved in the future to be separated per unit
	status  lbmanagement.Status
	message string
	payload map[string]interface{}

	// stop callback must be registered by libbeat, as with the V1 callback
	stopFunc           func()
	stopOnOutputReload bool
	stopOnEmptyUnits   bool
	stopMut            sync.Mutex
	beatStop           sync.Once

	// sync channel for shutting down the manager after we get a stop from
	// either the agent or the beat
	stopChan chan struct{}

	isRunning bool

	// set with the last applied output config
	// allows tracking if the configuration actually changed and if the
	// beat needs to restart if stopOnOutputReload is set
	lastOutputCfg *proto.UnitExpectedConfig

	// set with the last applied input configs
	lastInputCfgs map[string]*proto.UnitExpectedConfig

	// used for the debug callback to report as-running config
	lastBeatOutputCfg   *reload.ConfigWithMeta
	lastBeatInputCfgs   []*reload.ConfigWithMeta
	lastBeatFeaturesCfg *conf.C

	// changeDebounce is the debounce time for a configuration change
	changeDebounce time.Duration
	// forceReloadDebounce is the time the manager will wait before
	// trying to reload the configuration after an input not finished error
	// happens
	forceReloadDebounce time.Duration
}

// ================================
// Optionals
// ================================

// WithStopOnEmptyUnits enables stopping the beat when agent sends no units.
func WithStopOnEmptyUnits(m *BeatV2Manager) {
	m.stopOnEmptyUnits = true
}

// WithChangeDebounce sets the changeDeboung value
func WithChangeDebounce(d time.Duration) func(b *BeatV2Manager) {
	return func(b *BeatV2Manager) {
		b.changeDebounce = d
	}
}

// WithForceReloadDebounce sets the forceReloadDebounce value
func WithForceReloadDebounce(d time.Duration) func(b *BeatV2Manager) {
	return func(b *BeatV2Manager) {
		b.forceReloadDebounce = d
	}
}

// ================================
// Init Functions
// ================================

// Register the agent manager, so that calls to lbmanagement.NewManager will
// invoke NewV2AgentManager when linked with x-pack.
func init() {
	lbmanagement.SetManagerFactory(NewV2AgentManager)
}

// NewV2AgentManager returns a remote config manager for the agent V2 protocol.
// This is registered as the manager factory in init() so that calls to
// lbmanagement.NewManager will be forwarded here.
func NewV2AgentManager(config *conf.C, registry *reload.Registry) (lbmanagement.Manager, error) {
	logger := logp.NewLogger(lbmanagement.DebugK).Named("V2-manager")
	c := DefaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, fmt.Errorf("parsing fleet management settings: %w", err)
		}
	}

	versionInfo := client.VersionInfo{
		Name:      "beat-v2-client",
		BuildHash: version.Commit(),
		Meta: map[string]string{
			"commit":     version.Commit(),
			"build_time": version.BuildTime().String(),
		}}
	var agentClient client.V2
	var err error
	if c.InsecureGRPCURLForTesting != "" && c.Enabled {
		// Insecure for testing Elastic-Agent-Client initialisation
		logger.Info("Using INSECURE GRPC connection, this should be only used for testing!")
		agentClient = client.NewV2(c.InsecureGRPCURLForTesting,
			"", // Insecure connection for test, no token needed
			versionInfo,
			client.WithGRPCDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))
	} else {
		// Normal Elastic-Agent-Client initialisation
		agentClient, _, err = client.NewV2FromReader(os.Stdin, versionInfo)
		if err != nil {
			return nil, fmt.Errorf("error reading control config from agent: %w", err)
		}
	}

	// officially running under the elastic-agent; we set the publisher pipeline
	// to inform it that we are running under elastic-agent (used to ensure "Publish event: "
	// debug log messages are only outputted when running in trace mode
	lbmanagement.SetUnderAgent(true)

	return NewV2AgentManagerWithClient(c, registry, agentClient)
}

// NewV2AgentManagerWithClient actually creates the manager instance used by the rest of the beats.
func NewV2AgentManagerWithClient(config *Config, registry *reload.Registry, agentClient client.V2, opts ...func(*BeatV2Manager)) (lbmanagement.Manager, error) {
	log := logp.NewLogger(lbmanagement.DebugK)
	if config.RestartOnOutputChange {
		log.Infof("Output reload is enabled, the beat will restart as needed on change of output config")
	}
	m := &BeatV2Manager{
		stopOnOutputReload: config.RestartOnOutputChange,
		config:             config,
		logger:             log.Named("V2-manager"),
		registry:           registry,
		units:              make(map[unitKey]*client.Unit),
		status:             lbmanagement.Running,
		message:            "Healthy",
		stopChan:           make(chan struct{}, 1),
		changeDebounce:     time.Second,
		// forceReloadDebounce is greater than changeDebounce because it is only
		// used when an input has not reached its finished state, this means some events
		// still need to be acked by the acker, hence the longer we wait the more likely
		// for the input to have reached its finished state.
		forceReloadDebounce: time.Second * 10,
	}

	if config.Enabled {
		m.client = agentClient
	}
	for _, o := range opts {
		o(m)
	}
	return m, nil
}

// ================================
// Beats central management interface implementation
// ================================

func (cm *BeatV2Manager) AgentInfo() client.AgentInfo {
	if cm.client.AgentInfo() == nil {
		return client.AgentInfo{}
	}

	return *cm.client.AgentInfo()
}

// RegisterDiagnosticHook will register a diagnostic callback function when elastic-agent asks for a diagnostics dump
func (cm *BeatV2Manager) RegisterDiagnosticHook(name string, description string, filename string, contentType string, hook client.DiagnosticHook) {
	cm.client.RegisterDiagnosticHook(name, description, filename, contentType, hook)
}

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

// SetStopCallback sets the callback to run when the manager want to shut down the beats gracefully.
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
	if cm.errCanceller != nil {
		cm.errCanceller()
		cm.errCanceller = nil
	}

	ctx := context.Background()
	err := cm.client.Start(ctx)
	if err != nil {
		return fmt.Errorf("error starting connection to client")
	}
	ctx, canceller := context.WithCancel(ctx)
	cm.errCanceller = canceller
	go cm.watchErrChan(ctx)
	cm.client.RegisterDiagnosticHook(
		"beat-rendered-config",
		"the rendered config used by the beat",
		"beat-rendered-config.yml",
		"application/yaml",
		cm.handleDebugYaml)

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
// This is done because beats at the moment cannot fully manage different status per unit, something
// that is new in the V2 control protocol but not supported in beats itself.
//
// Errors while starting/reloading inputs are already reported by unit, but
// the shutdown process is still not being handled by unit.
func (cm *BeatV2Manager) updateStatuses() {
	status := getUnitState(cm.status)
	message := cm.message
	payload := cm.payload

	for _, unit := range cm.units {
		expected := unit.Expected()
		if expected.State == client.UnitStateStopped {
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
	// but the lock still needs to be held so reload can be triggered
	cm.mx.Lock()
	defer cm.mx.Unlock()

	// no need to update cm.units because the elastic-agent-client and the beats share
	// the pointer to each unit, so when the client updates a unit on its side, it
	// is reflected here. As this deals with modifications, they're already present.
	// Only the state needs to be updated.

	expected := unit.Expected()
	if expected.State == client.UnitStateStopped {
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
	delete(cm.units, unitKey{unit.Type(), unit.ID()})
	empty := len(cm.units) == 0
	cm.mx.Unlock()

	// stop the entire beat when all units removed
	if empty && cm.stopOnEmptyUnits {
		cm.stopBeat()
	}
}

// ================================
// Private V2 implementation
// ================================

func (cm *BeatV2Manager) watchErrChan(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-cm.client.Errors():
			// Don't print the context canceled errors that happen normally during shutdown, restart, etc
			if !errors.Is(context.Canceled, err) {
				cm.logger.Errorf("elastic-agent-client error: %s", err)
			}

		}
	}
}

func (cm *BeatV2Manager) unitListen() {
	// register signal handler
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// timer is used to provide debounce on unit changes
	// this allows multiple changes to come in and only a single reload be performed
	t := time.NewTimer(cm.changeDebounce)
	t.Stop() // starts stopped, until a change occurs

	cm.logger.Debug("Listening for agent unit changes")
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
			cm.logger.Infof(
				"BeatV2Manager.unitListen UnitChanged.ID(%s), UnitChanged.Type(%s), UnitChanged.Trigger(%d): %s/%s",
				change.Unit.ID(),
				change.Type, int64(change.Triggers), change.Type, change.Triggers)

			switch change.Type {
			// Within the context of how we send config to beats, I'm not sure if there is a difference between
			// A unit add and a unit change, since either way we can't do much more than call the reloader
			case client.UnitChangedAdded:
				cm.addUnit(change.Unit)
				// reset can be called here because `<-t.C` is handled in the same select
				t.Reset(cm.changeDebounce)
			case client.UnitChangedModified:
				cm.modifyUnit(change.Unit)
				// reset can be called here because `<-t.C` is handled in the same select
				t.Reset(cm.changeDebounce)
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
			if cm.forceReload {
				// Restart the debounce timer so we try to reload the inputs.
				t.Reset(cm.forceReloadDebounce)
			}
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
	if cm.errCanceller != nil {
		cm.errCanceller()
		cm.errCanceller = nil
	}
}

func (cm *BeatV2Manager) reload(units map[unitKey]*client.Unit) {
	lowestLevel := client.UnitLogLevelError
	var outputUnit *client.Unit
	var inputUnits []*client.Unit
	var stoppingUnits []*client.Unit
	healthyInputs := map[string]*client.Unit{}
	unitErrors := map[string][]error{}

	// as the very last action, set the state of the failed units
	defer func() {
		for _, unit := range units {
			errs := unitErrors[unit.ID()]
			if len(errs) != 0 {
				_ = unit.UpdateState(client.UnitStateFailed, errors.Join(errs...).Error(), nil)
			}
		}
	}()

	for _, unit := range units {
		expected := unit.Expected()
		if expected.LogLevel > lowestLevel {
			// log level is still used from an expected stopped unit until
			// the unit is completely removed (aka. fully stopped)
			lowestLevel = expected.LogLevel
		}
		if expected.Features != nil {
			// unit is expected to update its feature flags
			featuresCfg, err := features.NewConfigFromProto(expected.Features)
			if err != nil {
				unitErrors[unit.ID()] = append(unitErrors[unit.ID()], err)
			}

			if err := features.UpdateFromConfig(featuresCfg); err != nil {
				unitErrors[unit.ID()] = append(unitErrors[unit.ID()], err)
			}

			cm.lastBeatFeaturesCfg = featuresCfg
		}
		if expected.State == client.UnitStateStopped {
			// unit is being stopped
			//
			// we keep the unit so after reload is performed
			// these units can be marked as stopped
			stoppingUnits = append(stoppingUnits, unit)
			continue
		} else if expected.State != client.UnitStateHealthy {
			// only stopped or healthy are known (and expected) state
			// for a unit
			cm.logger.Errorf("unit %s has an unknown state %+v",
				unit.ID(), expected.State)
		}
		if unit.Type() == client.UnitTypeOutput {
			outputUnit = unit
		} else if unit.Type() == client.UnitTypeInput {
			inputUnits = append(inputUnits, unit)
			healthyInputs[unit.ID()] = unit
		} else {
			cm.logger.Errorf("unit %s as an unknown type %+v", unit.ID(), unit.Type())
		}
	}

	// set the new log level (if nothing has changed is a noop)
	ll, trace := getZapcoreLevel(lowestLevel)
	logp.SetLevel(ll)
	lbmanagement.SetUnderAgentTrace(trace)

	// reload the output configuration
	restartBeat, err := cm.reloadOutput(outputUnit)
	// The manager has already signalled the Beat to stop,
	// there is nothing else to do. Trying to reload inputs
	// will only lead to invalid state updates and possible
	// race conditions.
	if restartBeat {
		return
	}
	if err != nil {
		// Output creation failed, there is no point in going any further
		// because there is no output to read events.
		//
		// Trying to start inputs will eventually lead them to deadlock
		// waiting for the output. Log input will deadlock when starting,
		// effectively blocking this manager.
		cm.logger.Errorw("could not start output", "error", err)

		msg := fmt.Sprintf("could not start output: %s", err)
		if err := outputUnit.UpdateState(client.UnitStateFailed, msg, nil); err != nil {
			cm.logger.Errorw("setting output state", "error", err)
		}

		return
	}

	if err := outputUnit.UpdateState(client.UnitStateHealthy, "Healthy", nil); err != nil {
		cm.logger.Errorw("setting output state", "error", err)
	}

	// compute the input configuration
	//
	// in v2 only a single input type will be started per component, so we don't need to
	// worry about getting multiple re-loaders (we just need the one for the type)
	if err := cm.reloadInputs(inputUnits); err != nil {
		merror := &multierror.MultiError{}
		if errors.As(err, &merror) {
			for _, err := range merror.Errors {
				unitErr := cfgfile.UnitError{}
				if errors.As(err, &unitErr) {
					unitErrors[unitErr.UnitID] = append(unitErrors[unitErr.UnitID], unitErr.Err)
					delete(healthyInputs, unitErr.UnitID)
				}
			}
		}
	}

	// report the stopping units as stopped
	for _, unit := range stoppingUnits {
		_ = unit.UpdateState(client.UnitStateStopped, "Stopped", nil)
	}

	// now update the statuses of all units that contain only healthy
	// inputs. If there isn't an error with the inputs, we set the unit as
	// healthy because there is no way to know more information about its inputs.
	for _, unit := range healthyInputs {
		expected := unit.Expected()
		if expected.State == client.UnitStateStopped {
			// unit is expected to be stopping (don't adjust the state as the state is now managed by the
			// `reload` method and will be marked stopped in that code path)
			continue
		}

		err := unit.UpdateState(client.UnitStateHealthy, "Healthy", nil)
		if err != nil {
			cm.logger.Errorf("Failed to update unit %s status: %s", unit.ID(), err)
		}
	}
}

// reloadOutput reload outputs, it returns a bool and an error.
// The bool, if set, indicates that the output reload requires an restart,
// in that case the error is always `nil`.
//
// In any other case, the bool is always false and the error will be non nil
// if any error has occurred.
func (cm *BeatV2Manager) reloadOutput(unit *client.Unit) (bool, error) {
	// Assuming that the output reloadable isn't a list, see createBeater() in cmd/instance/beat.go
	output := cm.registry.GetReloadableOutput()
	if output == nil {
		return false, fmt.Errorf("failed to find beat reloadable type 'output'")
	}

	if unit == nil {
		// output is being stopped
		err := output.Reload(nil)
		if err != nil {
			return false, fmt.Errorf("failed to reload output: %w", err)
		}
		cm.lastOutputCfg = nil
		cm.lastBeatOutputCfg = nil
		return false, nil
	}

	expected := unit.Expected()
	if expected.Config == nil {
		// should not happen; hard stop
		return false, fmt.Errorf("output unit has no config")
	}

	if cm.lastOutputCfg != nil && gproto.Equal(cm.lastOutputCfg, expected.Config) {
		// configuration for the output did not change; do nothing
		cm.logger.Debug("Skipped reloading output; configuration didn't change")
		return false, nil
	}

	cm.logger.Debugf("Got output unit config '%s'", expected.Config.GetId())

	if cm.stopOnOutputReload && cm.lastOutputCfg != nil {
		cm.logger.Info("beat is restarting because output changed")
		_ = unit.UpdateState(client.UnitStateStopping, "Restarting", nil)
		cm.Stop()
		return true, nil
	}

	reloadConfig, err := groupByOutputs(expected.Config)
	if err != nil {
		return false, fmt.Errorf("failed to generate config for output: %w", err)
	}

	// Set those variables regardless of the outcome of output.Reload
	// this ensures that if we're on a failed output state and a new
	// output configuration is sent, the Beat will gracefully exit
	cm.lastOutputCfg = expected.Config
	cm.lastBeatOutputCfg = reloadConfig

	err = output.Reload(reloadConfig)
	if err != nil {
		return false, fmt.Errorf("failed to reload output: %w", err)
	}
	return false, nil
}

func (cm *BeatV2Manager) reloadInputs(inputUnits []*client.Unit) error {
	obj := cm.registry.GetInputList()
	if obj == nil {
		return fmt.Errorf("failed to find beat reloadable type 'input'")
	}

	inputCfgs := make(map[string]*proto.UnitExpectedConfig, len(inputUnits))
	inputBeatCfgs := make([]*reload.ConfigWithMeta, 0, len(inputUnits))
	agentInfo := cm.client.AgentInfo()

	for _, unit := range inputUnits {
		expected := unit.Expected()
		if expected.Config == nil {
			// should not happen; hard stop
			return fmt.Errorf("input unit %s has no config", unit.ID())
		}

		inputCfg, err := generateBeatConfig(expected.Config, agentInfo)
		if err != nil {
			return fmt.Errorf("failed to generate configuration for unit %s: %w", unit.ID(), err)
		}
		// add diag callbacks for unit
		// we want to add the diagnostic handler that's specific to the unit, and not the gobal diagnostic handler
		for _, in := range inputCfg {
			in.DiagCallback = diagnosticHandler{client: unit, log: cm.logger.Named("diagnostic-manager")}
			in.InputUnitID = unit.ID()
		}
		inputCfgs[unit.ID()] = expected.Config
		inputBeatCfgs = append(inputBeatCfgs, inputCfg...)
	}

	if !didChange(cm.lastInputCfgs, inputCfgs) && !cm.forceReload {
		cm.logger.Debug("Skipped reloading input units; configuration didn't change")
		return nil
	}

	if cm.forceReload {
		cm.logger.Info("Reloading Beats inputs because forceReload is true. " +
			"Set log level to debug to get more information about which " +
			"inputs are causing this.")
	}

	if err := obj.Reload(inputBeatCfgs); err != nil {
		merror := &multierror.MultiError{}
		realErrors := multierror.Errors{}

		// At the moment this logic is tightly bound to the current RunnerList
		// implementation from libbeat/cfgfile/list.go and Input.loadStates from
		// filebeat/input/log/input.go.
		// If they change the way they report errors, this will break.
		// TODO (Tiago): update all layers to use the most recent features from
		// the standard library errors package.
		if errors.As(err, &merror) {
			for _, err := range merror.Errors {
				causeErr := errors.Unwrap(err)
				// A Log input is only marked as finished when all events it
				// produced are acked by the acker so when we see this error,
				// we just retry until the new input can be started.
				// This is the same logic used by the standalone configuration file
				// reloader implemented on libbeat/cfgfile/reload.go
				inputNotFinishedErr := &common.ErrInputNotFinished{}
				if ok := errors.As(causeErr, &inputNotFinishedErr); ok {
					cm.logger.Debugf("file '%s' is not finished, will retry starting the input soon", inputNotFinishedErr.File)
					cm.forceReload = true
					cm.logger.Debug("ForceReload set to TRUE")
					continue
				}

				// This is an error that cannot be ignored, so we report it
				realErrors = append(realErrors, err)
			}
		}

		if len(realErrors) != 0 {
			return fmt.Errorf("failed to reload inputs: %w", realErrors.Err())
		}
	} else {
		// If there was no error reloading input and forceReload was
		// true, then set it to false. This prevents unnecessary logging
		// and makes it clear this was the moment when the input reload
		// finally worked.
		if cm.forceReload {
			cm.forceReload = false
			cm.logger.Debug("ForceReload set to FALSE")
		}
	}

	cm.lastInputCfgs = inputCfgs
	cm.lastBeatInputCfgs = inputBeatCfgs
	return nil
}

// this function is registered as a debug hook
// it prints the last known configuration generated by the beat
func (cm *BeatV2Manager) handleDebugYaml() []byte {
	// generate input
	inputList := []map[string]interface{}{}
	for _, module := range cm.lastBeatInputCfgs {
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
	if cm.lastBeatOutputCfg != nil {
		err := cm.lastBeatOutputCfg.Config.Unpack(&outputCfg)
		if err != nil {
			cm.logger.Errorf("error unpacking output config for debug callback: %s", err)
			return nil
		}
	}

	// generate features
	var featuresCfg map[string]interface{}
	if cm.lastBeatFeaturesCfg != nil {
		if err := cm.lastBeatFeaturesCfg.Unpack(&featuresCfg); err != nil {
			cm.logger.Errorf("error unpacking feature flags config for debug callback: %s", err)
			return nil
		}
	}

	// combine all of the above in a somewhat coherent way
	// This isn't perfect, but generating a config that can actually be fed back into the beat
	// would require
	beatCfg := struct {
		Inputs   []map[string]interface{}
		Outputs  map[string]interface{}
		Features map[string]interface{}
	}{
		Inputs:   inputList,
		Outputs:  outputCfg,
		Features: featuresCfg,
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

func getZapcoreLevel(ll client.UnitLogLevel) (zapcore.Level, bool) {
	switch ll {
	case client.UnitLogLevelError:
		return zapcore.ErrorLevel, false
	case client.UnitLogLevelWarn:
		return zapcore.WarnLevel, false
	case client.UnitLogLevelInfo:
		return zapcore.InfoLevel, false
	case client.UnitLogLevelDebug:
		return zapcore.DebugLevel, false
	case client.UnitLogLevelTrace:
		// beats doesn't support trace
		// but we do allow the "Publish event:" debug logs
		// when trace mode is enabled
		return zapcore.DebugLevel, true
	}
	// info level for fallback
	return zapcore.InfoLevel, false
}

func didChange(previous map[string]*proto.UnitExpectedConfig, latest map[string]*proto.UnitExpectedConfig) bool {
	if (previous == nil && latest != nil) || (previous != nil && latest == nil) {
		return true
	}
	if len(previous) != len(latest) {
		return true
	}
	for k, v := range latest {
		p, ok := previous[k]
		if !ok {
			return true
		}
		if !gproto.Equal(p, v) {
			return true
		}
	}
	return false
}
