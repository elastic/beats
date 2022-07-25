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
	"gopkg.in/yaml.v2"
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

type InputMetadata struct {
	Package PackageData
}

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

type BeatV2Manager struct {
	config   *Config
	registry *reload.Registry
	client   client.V2

	logger *logp.Logger

	// Track individual units given to us by the V2 API
	unitsMut sync.Mutex
	units    map[string]*client.Unit

	// This satisfies the SetPayload() function, and will pass along this value to the UpdateStatus()
	// call whenever a config is re-registered
	payload map[string]interface{}
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
	}

	if config.Enabled {
		m.client = agentClient
	}

	return m, nil
}

// Manager implementation

// UpdateStatus updates the manager with the current status for the beat.
func (cm *BeatV2Manager) UpdateStatus(status lbmanagement.Status, msg string) {
	// TODO: Updates on V2 are handled by individual units. What unit do we fetch for status updates here?
}

// Enabled returns true if config management is enabled.
func (cm *BeatV2Manager) Enabled() bool {
	return cm.config.Enabled
}

// SetStopCallback sets the callback to run when the manager want to shutdown the beats gracefully.
func (cm *BeatV2Manager) SetStopCallback(stopFunc func()) {
	//TODO: figure out how to use this
}

// Start the config manager.
func (cm *BeatV2Manager) Start() error {
	// TODO: this will spin up a UnitChanges() listener in another thread?
	err := cm.client.Start(context.Background())
	if err != nil {
		return fmt.Errorf("error starting connection to client")
	}

	return nil
}

// Stop stops the current Manager and close the connection to Elastic Agent.
func (cm *BeatV2Manager) Stop() {
	// TODO: This will run a callback to shut down the agent client
}

// NOTE: This is currently not implemented for fleet.
func (cm *BeatV2Manager) CheckRawConfig(cfg *conf.C) error {
	// This does not do anything on V1 or V2, but here we are
	return nil
}

func (cm *BeatV2Manager) RegisterAction(action client.Action) {
	// like the UpdateStatus() method, RegisterAction() in V2 is attached to a unit, we need to figure out which
}

func (cm *BeatV2Manager) UnregisterAction(action client.Action) {
	// like the UpdateStatus() method, RegisterAction() in V2 is attached to a unit, we need to figure out which
}

func (cm *BeatV2Manager) SetPayload(payload map[string]interface{}) {
	// similar problem as before, although since this is supposed to be attached to a StatusUpdate() in OnConfig,
	// this can maybe go on any Unit update
	cm.payload = payload
}

// Wrappers for the unit map

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

// private V3 implementation

func (cm *BeatV2Manager) unitListen() {
	for {
		select {

		case change := <-cm.client.UnitChanges():

			switch change.Type {
			case client.UnitChangedAdded:
				go cm.handleUnitReload(change.Unit)
			case client.UnitChangedModified:
				go cm.handleUnitReload(change.Unit)
			case client.UnitChangedRemoved:
				// the Reloadable interface doesn't have any kind of stop/shutdown command
				cm.deleteUnit(change.Unit)
			}
		}
	}
}

func (cm *BeatV2Manager) handleUnitReload(unit *client.Unit) {
	cm.addUnit(unit)
	unitType := unit.Type()
	reloadConfig, err := createConfigForReloader(unit)
	if err != nil {
		errString := fmt.Errorf("Failed to generate config for output: %w", err)
		unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
	}

	if unitType == client.UnitTypeOutput { // unit for the beat's configured output
		// Assuming that the output reloadable isn't a list, see createBeater() in cmd/instance/beat.go
		output := cm.registry.GetReloadable("output")

		unit.UpdateState(client.UnitStateConfiguring, "reloading component", nil)
		err = output.Reload(reloadConfig)
		if err != nil {
			errString := fmt.Errorf("Failed to reload component: %w", err)
			unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
		}
		unit.UpdateState(client.UnitStateHealthy, "reloaded component", nil)
	} else if unitType == client.UnitTypeInput {

		// Find the V2 inputs we need to reload
		// I'm not 100% sure that we'll "get" a unit that is specific to the beat we're currently running,
		// and we may need some kind of filter here to make sure we route unit configs to the correct reloader

		// The reloader provides list and non-list types, but all the beats register as lists,
		// so just go with that for V2
		obj := cm.registry.GetReloadableList("input")
		if obj == nil {
			unit.UpdateState(client.UnitStateFailed, "failed to find beat reloadable type 'input'", nil)
			return
		}
		var unitCfg UnitsConfig
		_, unitRaw := unit.Expected()
		err := yaml.Unmarshal([]byte(unitRaw), &unitCfg)
		if err != nil {
			errString := fmt.Errorf("Failed to create Unit config: %w", err)
			unit.UpdateState(client.UnitStateFailed, errString.Error(), nil)
		}
		beatCfg, err := generateBeatConfig(unitCfg)

	}
}

func (cm *BeatV2Manager) handleUnitUpdated(unit *client.Unit) {

}

// helpers

// createConfigForReloader is a little helper that takes the raw config from the unit
// and converts it to the weird config type used by the reloaders
func createConfigForReloader(unit *client.Unit) (*reload.ConfigWithMeta, error) {

	uconfig, err := conf.NewConfigFrom(unitRaw)
	if err != nil {
		return nil, err
	}
	return &reload.ConfigWithMeta{Config: uconfig}, nil
}
