package management

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/reload"
	lbmanagement "github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// BeatInput represents the "root" of a single input config
type BeatInput struct {
	Enabled   bool
	Id        ComponentID
	Name      string
	UseOutput string
	Streams   []MetricbeatStream
	Meta      InputMetadata
}

// InputMetadata wraps the metadata for an input
type InputMetadata struct {
	Package PackageData
}

// PackageData carries metadata for an assoicated package
type PackageData struct {
	Name    string
	Version string
}

// ComponentID covers the `id` field found in streams and inputs,
// which currently requires regex and string format statements to manipulate
type ComponentID struct {
	Type      string
	Namespace string
	Dataset   string
}

// MetricbeatStream covers the config for an indvidual stream found in an input config,
// which maps to a metricbeat module/metricset within metricbeat.
// Normally metricbeat has a one-to-many relationship between modules and metricsets,
// But fleet will give us one stream per metricset, translating to one module per metricset.
type MetricbeatStream struct {
	Hosts   []string
	Period  time.Duration
	Module  string
	Dataset string
	Enabled bool
	// The one remaining "quarantined" string blob.
	DatasetConfig string
}

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

	isRunning bool
}

// NewV2AgentManager returns a remote config manager for the agent V2 protocol.
// This is mainly meant to be used by the management plugin system, which will register this as a callback.
func NewV2AgentManager(config *conf.C, registry *reload.Registry, beatUUID uuid.UUID) (lbmanagement.Manager, error) {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, errors.Wrap(err, "parsing fleet management settings")
		}
	}
	agentClient, _, err := client.NewV2FromReader(os.Stdin, client.VersionInfo{Name: "elastic-agent-shipper", Version: "v2"})
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
		logger:   log.Named("fleet"),
		registry: registry,
		units:    make(map[string]*client.Unit),
	}

	if config.Enabled {
		m.client = agentClient
	}

	return m, nil
}

// ================================
// Manager interface implementation
// ================================

// UpdateStatus updates the manager with the current status for the beat.
func (cm *BeatV2Manager) UpdateStatus(status lbmanagement.Status, msg string) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	updateState := client.UnitState(status)
	cm.units[cm.mainUnit].UpdateState(updateState, msg, cm.payload)

}

// Enabled returns true if config management is enabled.
func (cm *BeatV2Manager) Enabled() bool {
	return cm.config.Enabled
}

// SetStopCallback sets the callback to run when the manager want to shutdown the beats gracefully.
func (cm *BeatV2Manager) SetStopCallback(stopFunc func()) {
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
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	main, ok := cm.units[cm.mainUnit]
	if ok {
		cm.stopBeat(main)
	}
}

// CheckRawConfig is currently not implemented for V1.
func (cm *BeatV2Manager) CheckRawConfig(cfg *conf.C) error {
	// This does not do anything on V1 or V2, but here we are
	return nil
}

func (cm *BeatV2Manager) RegisterAction(action client.Action) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	cm.units[cm.mainUnit].RegisterAction(action)
}

func (cm *BeatV2Manager) UnregisterAction(action client.Action) {
	cm.unitsMut.Lock()
	defer cm.unitsMut.Unlock()
	cm.units[cm.mainUnit].UnregisterAction(action)
}

func (cm *BeatV2Manager) SetPayload(payload map[string]interface{}) {
	cm.payload = payload
}

// ================================
// Unit manager
// ================================

func (c *BeatV2Manager) addUnit(unit *client.Unit) {
	c.unitsMut.Lock()
	c.units[unit.ID()] = unit
	c.unitsMut.Unlock()
}

func (c *BeatV2Manager) getUnit(ID string) *client.Unit {
	c.unitsMut.Lock()
	defer c.unitsMut.Unlock()
	return c.units[ID]

}

func (c *BeatV2Manager) deleteUnit(unit *client.Unit) {
	c.unitsMut.Lock()
	delete(c.units, unit.ID())
	c.unitsMut.Unlock()
}

// ================================
// Private V3 implementation
// ================================

func (cm *BeatV2Manager) unitListen() {
	cm.logger.Debugf("Listening for agent unit changes")
	fmt.Printf("ARGV: %s\n", os.Args[0])
	for {
		select {

		case change := <-cm.client.UnitChanges():

			switch change.Type {
			// Within the context of how we send config to beats, I'm not sure there is a difference between
			// A unit add and a unit change, since either way we need to fetch the reloader and hand over the config
			case client.UnitChangedAdded:
				go cm.handleUnitReload(change.Unit)
			case client.UnitChangedModified:
				go cm.handleUnitReload(change.Unit)
			case client.UnitChangedRemoved:
				cm.stopBeat(change.Unit)
			}
		}
	}
}

// We need a "main" unit that we can send updates to for the StatusReporter interface
// the purpose of this is to just grab the first input-type unit we get and set it as the "main" unit
func (cm *BeatV2Manager) setMainUnitValue(unit *client.Unit) {
	if cm.mainUnit == "" {
		cm.mainUnit = unit.ID()
	}
}

func (cm *BeatV2Manager) stopBeat(unit *client.Unit) {

	// will we ever get a Unit removed for anything other than the main beat?
	// Individual reloaders don't have a "stop" function, so the most we can do
	// is just shut down a beat, I think.
	if !cm.isRunning {
		return
	}

	cm.isRunning = false
	unit.UpdateState(client.UnitStateStopping, "stopping beat", nil)
	if cm.stopFunc != nil {
		cm.stopFunc()
	}
	cm.client.Stop()
	unit.UpdateState(client.UnitStateStopped, "stopped beat", nil)
	cm.deleteUnit(unit)
}

func (cm *BeatV2Manager) handleUnitReload(unit *client.Unit) {
	cm.logger.Debugf("Starting unit reload for ID %s", unit.ID())
	cm.addUnit(unit)
	unitType := unit.Type()

	if unitType == client.UnitTypeOutput { // unit for the beat's configured output
		cm.handleOutputReload(unit)
	} else if unitType == client.UnitTypeInput {
		cm.handleInputReload(unit)
	}
}

func (cm *BeatV2Manager) handleOutputReload(unit *client.Unit) {
	_, rawConfig := unit.Expected()
	reloadConfig, err := groupByOutputs(rawConfig)
	if err != nil {
		errString := fmt.Errorf("Failed to generate config for output: %w", err)
		unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
	}
	// Assuming that the output reloadable isn't a list, see createBeater() in cmd/instance/beat.go
	output := cm.registry.GetReloadable("output")
	if output == nil {
		unit.UpdateState(client.UnitStateFailed, "failed to find beat reloadable type 'output'", nil)
		return
	}

	unit.UpdateState(client.UnitStateConfiguring, "reloading output component", nil)
	err = output.Reload(reloadConfig)
	if err != nil {
		errString := fmt.Errorf("Failed to reload component: %w", err)
		unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
	}
	unit.UpdateState(client.UnitStateHealthy, "reloaded output component", nil)
}

func (cm *BeatV2Manager) handleInputReload(unit *client.Unit) {
	_, rawConfig := unit.Expected()
	cm.setMainUnitValue(unit)

	// Find the V2 inputs we need to reload

	// The reloader provides list and non-list types, but all the beats register as lists,
	// so just go with that for V2
	obj := cm.registry.GetReloadableList("input")
	if obj == nil {
		unit.UpdateState(client.UnitStateFailed, "failed to find beat reloadable type 'input'", nil)
		return
	}
	unit.UpdateState(client.UnitStateConfiguring, "found reloader for 'input'", nil)

	beatCfg, err := generateBeatConfig(rawConfig)
	if err != nil {
		errString := fmt.Errorf("Failed to create Unit config: %w", err)
		unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
	}

	fmt.Printf("raw config: %s\n", printConfigDebug(beatCfg))

	err = obj.Reload(beatCfg)
	if err != nil {
		errString := fmt.Errorf("Error reloading input: %w", err)
		unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
	}
	unit.UpdateState(client.UnitStateHealthy, "beat reloaded", nil)
}
