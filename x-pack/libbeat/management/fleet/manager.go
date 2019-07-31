// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"fmt"
	"sort"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/management"
	"github.com/elastic/beats/x-pack/libbeat/management/api"

	xmanagement "github.com/elastic/beats/x-pack/libbeat/management"
)

var errEmptyAccessToken = errors.New("access_token is empty, you must re-enroll your Beat")

// ConfigManager provides a functionality to retrieve config channel
// using which manager is informed about config changes
type ConfigManager interface {
	ConfigChan() chan<- map[string]interface{}
}

func init() {
	management.Register("x-pack-fleet", NewFleetManager, feature.Beta)
}

// Manager handles internal config updates. By retrieving
// new configs from Kibana and applying them to the Beat
type Manager struct {
	config    *Config
	cache     *xmanagement.Cache
	logger    *logp.Logger
	beatUUID  uuid.UUID
	done      chan struct{}
	registry  *reload.Registry
	wg        sync.WaitGroup
	blacklist *xmanagement.ConfigBlacklist

	configChan chan map[string]interface{}
}

// NewFleetManager returns a X-Pack Beats Central Management manager
func NewFleetManager(config *common.Config, registry *reload.Registry, beatUUID uuid.UUID) (management.ConfigManager, error) {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, errors.Wrap(err, "parsing central management settings")
		}
	}
	return NewFleetManagerWithConfig(c, registry, beatUUID)
}

// NewFleetManagerWithConfig returns a X-Pack Beats Central Management manager
func NewFleetManagerWithConfig(c *Config, registry *reload.Registry, beatUUID uuid.UUID) (management.ConfigManager, error) {
	var cache *xmanagement.Cache
	var blacklist *xmanagement.ConfigBlacklist

	if c.Fleet.Enabled {
		var err error

		// Initialize configs blacklist
		blacklist, err = xmanagement.NewConfigBlacklist(c.Blacklist)
		if err != nil {
			return nil, errors.Wrap(err, "wrong settings for configurations blacklist")
		}

		// Initialize central management settings cache
		cache = &xmanagement.Cache{}
		if err := cache.Load(); err != nil {
			return nil, errors.Wrap(err, "reading central management internal cache")
		}
	}

	log := logp.NewLogger(management.DebugK)

	return &Manager{
		config:     c,
		cache:      cache,
		blacklist:  blacklist,
		logger:     log,
		done:       make(chan struct{}),
		beatUUID:   beatUUID,
		registry:   registry,
		configChan: make(chan map[string]interface{}),
	}, nil
}

// Enabled returns true if config management is enabled
func (cm *Manager) Enabled() bool {
	return cm.config.Fleet.Enabled
}

// ConfigChan returns a channel used to communicate configuration changes.
func (cm *Manager) ConfigChan() chan<- map[string]interface{} {
	return cm.configChan
}

// Start the config manager
func (cm *Manager) Start() {
	if !cm.Enabled() {
		return
	}

	cfgwarn.Beta("Central management is enabled")
	cm.logger.Info("Starting central management service")

	cm.wg.Add(1)
	go cm.worker()
}

// Stop the config manager
func (cm *Manager) Stop() {
	if !cm.Enabled() {
		return
	}

	// stop collecting configuration
	cm.logger.Info("Stopping central management service")
	close(cm.done)
	cm.wg.Wait()
}

// CheckRawConfig check settings are correct to start the beat. This method
// checks there are no collision between the existing configuration and what
// central management can configure.
func (cm *Manager) CheckRawConfig(cfg *common.Config) error {
	// TODO implement this method
	return nil
}

func (cm *Manager) worker() {
	defer cm.wg.Done()

	// Start worker loop: fetch + apply + cache new settings
WORKERLOOP:
	for {
		select {
		case cfg := <-cm.configChan:
			// store in cache
			blocks, err := cm.toConfigBlocks(cfg)
			if err != nil {
				cm.logger.Errorf("Could not apply the configuration, error: %+v", err)
				continue WORKERLOOP
			}

			cm.cache.Configs = blocks
			if errs := cm.apply(); !errs.IsEmpty() {
				cm.logger.Errorf("Could not apply the configuration, error: %+v", errs)
				continue WORKERLOOP
			}

			// store new configs (already applied)
			cm.logger.Info("Storing new state")
			if err := cm.cache.Save(); err != nil {
				cm.logger.Errorf("error storing central management state: %s", err)
			}
		case <-cm.done:
			return
		}
	}
}

func (cm *Manager) apply() xmanagement.Errors {
	var errors xmanagement.Errors
	missing := map[string]bool{}
	for _, name := range cm.registry.GetRegisteredNames() {
		missing[name] = true
	}

	// Detect unwanted configs from the list
	if errs := cm.blacklist.Detect(cm.cache.Configs); !errs.IsEmpty() {
		errors = append(errors, errs...)
		return errors
	}

	// Reload configs
	for _, b := range cm.cache.Configs {
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

		rawBlock := map[string]interface{}{
			regName: iBlock,
		}

		blocks[regName] = append(blocks[regName], &api.ConfigBlock{Raw: rawBlock})
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

var _ ConfigManager = &Manager{}
