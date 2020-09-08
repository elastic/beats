// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/management/api"

	xmanagement "github.com/elastic/beats/v7/x-pack/libbeat/management"
)

// Manager handles internal config updates. By retrieving
// new configs from Kibana and applying them to the Beat.
type Manager struct {
	config    *Config
	logger    *logp.Logger
	beatUUID  uuid.UUID
	registry  *reload.Registry
	blacklist *xmanagement.ConfigBlacklist
	client    client.Client
	lock      sync.Mutex
	status    management.Status
	msg       string

	stopFunc func()
}

// NewFleetManager returns a X-Pack Beats Fleet Management manager.
func NewFleetManager(config *common.Config, registry *reload.Registry, beatUUID uuid.UUID) (management.Manager, error) {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, errors.Wrap(err, "parsing fleet management settings")
		}
	}
	return NewFleetManagerWithConfig(c, registry, beatUUID)
}

// NewFleetManagerWithConfig returns a X-Pack Beats Fleet Management manager.
func NewFleetManagerWithConfig(c *Config, registry *reload.Registry, beatUUID uuid.UUID) (management.Manager, error) {
	log := logp.NewLogger(management.DebugK)

	m := &Manager{
		config:   c,
		logger:   log.Named("fleet"),
		beatUUID: beatUUID,
		registry: registry,
	}

	var err error
	var blacklist *xmanagement.ConfigBlacklist
	var eac client.Client
	if c.Enabled && c.Mode == xmanagement.ModeFleet {
		// Initialize configs blacklist
		blacklist, err = xmanagement.NewConfigBlacklist(c.Blacklist)
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
	return cm.config.Enabled && cm.config.Mode == xmanagement.ModeFleet
}

// Start the config manager
func (cm *Manager) Start(stopFunc func()) {
	if !cm.Enabled() {
		return
	}

	cfgwarn.Beta("Fleet management is enabled")
	cm.logger.Info("Starting fleet management service")

	cm.stopFunc = stopFunc
	err := cm.client.Start(context.Background())
	if err != nil {
		cm.logger.Errorf("failed to start elastic-agent-client: %s", err)
	}
}

// Stop the config manager
func (cm *Manager) Stop() {
	if !cm.Enabled() {
		return
	}

	cm.logger.Info("Stopping fleet management service")
	cm.client.Stop()
}

// CheckRawConfig check settings are correct to start the beat. This method
// checks there are no collision between the existing configuration and what
// fleet management can configure.
func (cm *Manager) CheckRawConfig(cfg *common.Config) error {
	// TODO implement this method
	return nil
}

// UpdateStatus updates the manager with the current status for the beat.
func (cm *Manager) UpdateStatus(status management.Status, msg string) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if cm.status != status || cm.msg != msg {
		cm.status = status
		cm.msg = msg
		cm.client.Status(statusToProtoStatus(status), msg, nil)
		cm.logger.Infof("Status change to %s: %s", status, msg)
	}
}

func (cm *Manager) OnConfig(s string) {
	cm.UpdateStatus(management.Configuring, "Updating configuration")

	var configMap common.MapStr
	uconfig, err := common.NewConfigFrom(s)
	if err != nil {
		err = errors.Wrap(err, "config blocks unsuccessfully generated")
		cm.logger.Error(err)
		cm.UpdateStatus(management.Failed, err.Error())
		return
	}

	err = uconfig.Unpack(&configMap)
	if err != nil {
		err = errors.Wrap(err, "config blocks unsuccessfully generated")
		cm.logger.Error(err)
		cm.UpdateStatus(management.Failed, err.Error())
		return
	}

	blocks, err := cm.toConfigBlocks(configMap)
	if err != nil {
		err = errors.Wrap(err, "failed to parse configuration")
		cm.logger.Error(err)
		cm.UpdateStatus(management.Failed, err.Error())
		return
	}

	if errs := cm.apply(blocks); !errs.IsEmpty() {
		// `cm.apply` already logs the errors; currently allow beat to run degraded
		cm.UpdateStatus(management.Degraded, errs.Error())
		return
	}

	cm.client.Status(proto.StateObserved_HEALTHY, "Running", nil)
}

func (cm *Manager) OnStop() {
	if cm.stopFunc != nil {
		cm.client.Status(proto.StateObserved_STOPPING, "Stopping", nil)
		cm.stopFunc()
	}
}

func (cm *Manager) OnError(err error) {
	cm.logger.Errorf("elastic-agent-client got error: %s", err)
}

func (cm *Manager) apply(blocks api.ConfigBlocks) xmanagement.Errors {
	var errors xmanagement.Errors
	missing := map[string]bool{}
	for _, name := range cm.registry.GetRegisteredNames() {
		missing[name] = true
	}

	// Detect unwanted configs from the list
	if errs := cm.blacklist.Detect(blocks); !errs.IsEmpty() {
		errors = append(errors, errs...)
		return errors
	}

	// Reload configs
	for _, b := range blocks {
		if err := cm.reload(b.Type, b.Blocks); err != nil {
			errors = append(errors, err)
		}
		missing[b.Type] = false
	}

	// Unset missing configs
	for name := range missing {
		if missing[name] {
			if err := cm.reload(name, []*api.ConfigBlock{}); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

func (cm *Manager) reload(t string, blocks []*api.ConfigBlock) *xmanagement.Error {
	cm.logger.Infof("Applying settings for %s", t)
	if obj := cm.registry.GetReloadable(t); obj != nil {
		// Single object
		if len(blocks) > 1 {
			err := fmt.Errorf("got an invalid number of configs for %s: %d, expected: 1", t, len(blocks))
			cm.logger.Error(err)
			return xmanagement.NewConfigError(err)
		}

		var config *reload.ConfigWithMeta
		var err error
		if len(blocks) == 1 {
			config, err = blocks[0].ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				return xmanagement.NewConfigError(err)
			}
		}

		if err := obj.Reload(config); err != nil {
			cm.logger.Error(err)
			return xmanagement.NewConfigError(err)
		}
	} else if obj := cm.registry.GetReloadableList(t); obj != nil {
		// List
		var configs []*reload.ConfigWithMeta
		for _, block := range blocks {
			config, err := block.ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				return xmanagement.NewConfigError(err)
			}
			configs = append(configs, config)
		}

		if err := obj.Reload(configs); err != nil {
			cm.logger.Error(err)
			return xmanagement.NewConfigError(err)
		}
	}

	return nil
}

func (cm *Manager) toConfigBlocks(cfg common.MapStr) (api.ConfigBlocks, error) {
	blocks := map[string][]*api.ConfigBlock{}

	// Extract all registered values beat can respond to
	for _, regName := range cm.registry.GetRegisteredNames() {
		iBlock, err := cfg.GetValue(regName)
		if err != nil {
			continue
		}

		if mapBlock, ok := iBlock.(map[string]interface{}); ok {
			blocks[regName] = append(blocks[regName], &api.ConfigBlock{Raw: mapBlock})
		} else if arrayBlock, ok := iBlock.([]interface{}); ok {
			for _, item := range arrayBlock {
				if mapBlock, ok := item.(map[string]interface{}); ok {
					blocks[regName] = append(blocks[regName], &api.ConfigBlock{Raw: mapBlock})
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

	res := api.ConfigBlocks{}
	for _, t := range keys {
		b := blocks[t]
		res = append(res, api.ConfigBlocksWithType{Type: t, Blocks: b})
	}

	return res, nil
}

func statusToProtoStatus(status management.Status) proto.StateObserved_Status {
	switch status {
	case management.Unknown:
		// unknown is reported as healthy, as the status is unknown
		return proto.StateObserved_HEALTHY
	case management.Starting:
		return proto.StateObserved_STARTING
	case management.Configuring:
		return proto.StateObserved_CONFIGURING
	case management.Running:
		return proto.StateObserved_HEALTHY
	case management.Degraded:
		return proto.StateObserved_DEGRADED
	case management.Failed:
		return proto.StateObserved_FAILED
	case management.Stopping:
		return proto.StateObserved_STOPPING
	}
	// unknown status, still reported as healthy
	return proto.StateObserved_HEALTHY
}
