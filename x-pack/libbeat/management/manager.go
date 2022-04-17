// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/menderesk/elastic-agent-client/v7/pkg/client"
	"github.com/menderesk/elastic-agent-client/v7/pkg/proto"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/cfgwarn"
	"github.com/menderesk/beats/v7/libbeat/common/reload"
	"github.com/menderesk/beats/v7/libbeat/logp"
	lbmanagement "github.com/menderesk/beats/v7/libbeat/management"
)

var notReportedErrors = []error{
	context.Canceled,
}

// Manager handles internal config updates. By retrieving
// new configs from Kibana and applying them to the Beat.
type Manager struct {
	config    *Config
	logger    *logp.Logger
	beatUUID  uuid.UUID
	registry  *reload.Registry
	blacklist *ConfigBlacklist
	client    client.Client
	lock      sync.Mutex
	status    lbmanagement.Status
	msg       string
	payload   map[string]interface{}

	stopFunc  func()
	isRunning bool
}

// NewFleetManager returns a X-Pack Beats Fleet Management manager.
func NewFleetManager(config *common.Config, registry *reload.Registry, beatUUID uuid.UUID) (lbmanagement.Manager, error) {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, errors.Wrap(err, "parsing fleet management settings")
		}
	}
	return NewFleetManagerWithConfig(c, registry, beatUUID)
}

// NewFleetManagerWithConfig returns a X-Pack Beats Fleet Management manager.
func NewFleetManagerWithConfig(c *Config, registry *reload.Registry, beatUUID uuid.UUID) (lbmanagement.Manager, error) {
	log := logp.NewLogger(lbmanagement.DebugK)

	m := &Manager{
		config:   c,
		logger:   log.Named("fleet"),
		beatUUID: beatUUID,
		registry: registry,
	}

	var err error
	var blacklist *ConfigBlacklist
	var eac client.Client
	if c.Enabled {
		// Initialize configs blacklist
		blacklist, err = NewConfigBlacklist(c.Blacklist)
		if err != nil {
			return nil, errors.Wrap(err, "wrong settings for configurations blacklist")
		}

		// Initialize the client
		eac, err = client.NewFromReader(os.Stdin, m)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create elastic-agent-client")
		}
	}

	m.blacklist = blacklist
	m.client = eac
	return m, nil
}

// Enabled returns true if config management is enabled.
func (cm *Manager) Enabled() bool {
	return cm.config.Enabled
}

// SetStopCallback sets the callback to run when the manager want to shutdown the beats gracefully.
func (cm *Manager) SetStopCallback(stopFunc func()) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	cm.stopFunc = stopFunc
}

// Start the config manager.
func (cm *Manager) Start() error {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if !cm.Enabled() {
		return nil
	}

	cfgwarn.Beta("Fleet management is enabled")
	cm.logger.Info("Starting fleet management service")

	cm.isRunning = true
	err := cm.client.Start(context.Background())
	if err != nil {
		cm.logger.Errorf("failed to start elastic-agent-client: %s", err)
		return err
	}
	cm.logger.Info("Ready to receive configuration")
	return nil
}

// Stop stops the current Manager and close the connection to Elastic Agent.
func (cm *Manager) Stop() {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if !cm.Enabled() {
		return
	}

	cm.logger.Info("Stopping fleet management service")
	cm.isRunning = false
	cm.client.Stop()
}

// CheckRawConfig check settings are correct to start the beat. This method
// checks there are no collision between the existing configuration and what
// fleet management can configure.
//
// NOTE: This is currently not implemented for fleet.
func (cm *Manager) CheckRawConfig(cfg *common.Config) error {
	// TODO implement this method
	return nil
}

// UpdateStatus updates the manager with the current status for the beat.
func (cm *Manager) UpdateStatus(status lbmanagement.Status, msg string) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if cm.status != status || cm.msg != msg {
		cm.status = status
		cm.msg = msg
		cm.client.Status(statusToProtoStatus(status), msg, nil)
		cm.logger.Infof("Status change to %s: %s", status, msg)
	}
}

// updateStatusWithError updates the manager with the current status for the beat with error.
func (cm *Manager) updateStatusWithError(err error) {
	if err == nil {
		return
	}

	for _, e := range notReportedErrors {
		if errors.Is(err, e) {
			return
		}
	}

	cm.logger.Error(err)
	cm.UpdateStatus(lbmanagement.Failed, err.Error())
}

func (cm *Manager) OnConfig(s string) {
	cm.UpdateStatus(lbmanagement.Configuring, "Updating configuration")

	var configMap common.MapStr
	uconfig, err := common.NewConfigFrom(s)
	if err != nil {
		err = errors.Wrap(err, "config blocks unsuccessfully generated")
		cm.updateStatusWithError(err)
		return
	}

	err = uconfig.Unpack(&configMap)
	if err != nil {
		err = errors.Wrap(err, "config blocks unsuccessfully generated")
		cm.updateStatusWithError(err)
		return
	}

	blocks, err := cm.toConfigBlocks(configMap)
	if err != nil {
		err = errors.Wrap(err, "failed to parse configuration")
		cm.updateStatusWithError(err)
		return
	}

	if errs := cm.apply(blocks); errs != nil {
		// `cm.apply` already logs the errors; currently allow beat to run degraded
		cm.updateStatusWithError(err)
		cm.logger.Errorf("failed applying config blocks: %v", err)
		return
	}

	cm.client.Status(proto.StateObserved_HEALTHY, "Running", cm.payload)
}

func (cm *Manager) RegisterAction(action client.Action) {
	cm.client.RegisterAction(action)
}

func (cm *Manager) UnregisterAction(action client.Action) {
	cm.client.UnregisterAction(action)
}

func (cm *Manager) SetPayload(payload map[string]interface{}) {
	cm.lock.Lock()
	cm.payload = payload
	cm.lock.Unlock()
}

func (cm *Manager) OnStop() {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if cm.stopFunc != nil {
		cm.client.Status(proto.StateObserved_STOPPING, "Stopping", nil)
		cm.stopFunc()
	}
}

func (cm *Manager) OnError(err error) {
	isStopped := false
	cm.lock.Lock()
	isStopped = !cm.isRunning
	cm.lock.Unlock()

	if isStopped && errors.Is(err, context.Canceled) {
		// don't report context cancelled on shutdown
		return
	}
	cm.logger.Errorf("elastic-agent-client got error: %s", err)
}

func (cm *Manager) apply(blocks ConfigBlocks) error {
	missing := map[string]bool{}
	for _, name := range cm.registry.GetRegisteredNames() {
		missing[name] = true
	}

	// Detect unwanted configs from the list
	if err := cm.blacklist.Detect(blocks); err != nil {
		return err
	}

	var errors *multierror.Error
	// Reload configs
	for _, b := range blocks {
		if err := cm.reload(b.Type, b.Blocks); err != nil {
			errors = multierror.Append(errors, err)
		}
		missing[b.Type] = false
	}

	// Unset missing configs
	for name, isMissing := range missing {
		if isMissing {
			if err := cm.reload(name, []*ConfigBlock{}); err != nil {
				errors = multierror.Append(errors, err)
			}
		}
	}

	return errors.ErrorOrNil()
}

func (cm *Manager) reload(t string, blocks []*ConfigBlock) error {
	cm.logger.Infof("Applying settings for %s", t)
	if obj := cm.registry.GetReloadable(t); obj != nil {
		// Single object
		if len(blocks) > 1 {
			err := fmt.Errorf("got an invalid number of configs for %s: %d, expected: 1", t, len(blocks))
			cm.logger.Error(err)
			return err
		}

		var config *reload.ConfigWithMeta
		var err error
		if len(blocks) == 1 {
			config, err = blocks[0].ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				return err
			}
		}

		if err := obj.Reload(config); err != nil {
			cm.logger.Error(err)
			return err
		}
	} else if obj := cm.registry.GetReloadableList(t); obj != nil {
		// List
		var configs []*reload.ConfigWithMeta
		for _, block := range blocks {
			config, err := block.ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				return err
			}
			configs = append(configs, config)
		}

		if err := obj.Reload(configs); err != nil {
			cm.logger.Error(err)
			return err
		}
	}

	return nil
}

func (cm *Manager) toConfigBlocks(cfg common.MapStr) (ConfigBlocks, error) {
	blocks := map[string][]*ConfigBlock{}

	// Extract all registered values beat can respond to
	for _, regName := range cm.registry.GetRegisteredNames() {
		iBlock, err := cfg.GetValue(regName)
		if err != nil {
			cm.logger.Warnf("failed to get '%s' from config: %v. Continuing to next one", regName, err)
			continue
		}

		if mapBlock, ok := iBlock.(map[string]interface{}); ok {
			blocks[regName] = append(blocks[regName], &ConfigBlock{Raw: mapBlock})
		} else if arrayBlock, ok := iBlock.([]interface{}); ok {
			for _, item := range arrayBlock {
				if mapBlock, ok := item.(map[string]interface{}); ok {
					blocks[regName] = append(blocks[regName], &ConfigBlock{Raw: mapBlock})
				}
			}
		}
	}

	// keep the ordering consistent while grouping the items.
	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	res := ConfigBlocks{}
	for _, t := range keys {
		b := blocks[t]
		res = append(res, ConfigBlocksWithType{Type: t, Blocks: b})
	}

	return res, nil
}

func statusToProtoStatus(status lbmanagement.Status) proto.StateObserved_Status {
	switch status {
	case lbmanagement.Unknown:
		// unknown is reported as healthy, as the status is unknown
		return proto.StateObserved_HEALTHY
	case lbmanagement.Starting:
		return proto.StateObserved_STARTING
	case lbmanagement.Configuring:
		return proto.StateObserved_CONFIGURING
	case lbmanagement.Running:
		return proto.StateObserved_HEALTHY
	case lbmanagement.Degraded:
		return proto.StateObserved_DEGRADED
	case lbmanagement.Failed:
		return proto.StateObserved_FAILED
	case lbmanagement.Stopping:
		return proto.StateObserved_STOPPING
	}
	// unknown status, still reported as healthy
	return proto.StateObserved_HEALTHY
}
