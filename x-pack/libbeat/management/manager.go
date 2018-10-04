// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/reload"

	"github.com/satori/go.uuid"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/libbeat/management/api"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/management"
)

func init() {
	management.Register("x-pack", NewConfigManager, feature.Beta)
}

// ConfigManager handles internal config updates. By retrieving
// new configs from Kibana and applying them to the Beat
type ConfigManager struct {
	config   *Config
	logger   *logp.Logger
	client   *api.Client
	beatUUID uuid.UUID
	done     chan struct{}
	registry *reload.Registry
	wg       sync.WaitGroup
}

// NewConfigManager returns a X-Pack Beats Central Management manager
func NewConfigManager(registry *reload.Registry, beatUUID uuid.UUID) (management.ConfigManager, error) {
	c := defaultConfig()
	if err := c.Load(); err != nil {
		return nil, errors.Wrap(err, "reading central management internal settings")
	}
	return NewConfigManagerWithConfig(c, registry, beatUUID)
}

// NewConfigManagerWithConfig returns a X-Pack Beats Central Management manager
func NewConfigManagerWithConfig(c *Config, registry *reload.Registry, beatUUID uuid.UUID) (management.ConfigManager, error) {
	var client *api.Client
	if c.Enabled {
		var err error

		// Ignore kibana version to avoid permission errors
		c.Kibana.IgnoreVersion = true

		client, err = api.NewClient(c.Kibana)
		if err != nil {
			return nil, errors.Wrap(err, "initializing kibana client")
		}
	}

	return &ConfigManager{
		config:   c,
		logger:   logp.NewLogger(management.DebugK),
		client:   client,
		done:     make(chan struct{}),
		beatUUID: beatUUID,
		registry: registry,
	}, nil
}

// Enabled returns true if config management is enabled
func (cm *ConfigManager) Enabled() bool {
	return cm.config.Enabled
}

// Start the config manager
func (cm *ConfigManager) Start() {
	if !cm.Enabled() {
		return
	}
	cfgwarn.Beta("Central management is enabled")
	cm.logger.Info("Starting central management service")

	cm.wg.Add(1)
	go cm.worker()
}

// Stop the config manager
func (cm *ConfigManager) Stop() {
	if !cm.Enabled() {
		return
	}
	cm.logger.Info("Stopping central management service")
	close(cm.done)
	cm.wg.Wait()
}

// CheckRawConfig check settings are correct to start the beat. This method
// checks there are no collision between the existing configuration and what
// central management can configure.
func (cm *ConfigManager) CheckRawConfig(cfg *common.Config) error {
	// TODO implement this method
	return nil
}

func (cm *ConfigManager) worker() {
	defer cm.wg.Done()

	// Initial fetch && apply (even if errors happen while fetching)
	firstRun := true
	period := 0 * time.Second

	// Start worker loop: fetch + apply + cache new settings
	for {
		select {
		case <-cm.done:
			return
		case <-time.After(period):
		}

		changed := cm.fetch()
		if changed || firstRun {
			// configs changed, apply changes
			// TODO only reload the blocks that changed
			cm.apply()
		}

		if changed {
			// store new configs (already applied)
			cm.logger.Info("Storing new state")
			if err := cm.config.Save(); err != nil {
				cm.logger.Errorf("error storing central management state: %s", err)
			}
		}

		if firstRun {
			period = cm.config.Period
			firstRun = false
		}
	}
}

// fetch configurations from kibana, return true if they changed
func (cm *ConfigManager) fetch() bool {
	cm.logger.Debug("Retrieving new configurations from Kibana")
	configs, err := cm.client.Configuration(cm.config.AccessToken, cm.beatUUID)
	if err != nil {
		cm.logger.Errorf("error retriving new configurations, will use cached ones: %s", err)
		return false
	}

	if api.ConfigBlocksEqual(configs, cm.config.Configs) {
		cm.logger.Debug("configuration didn't change, sleeping")
		return false
	}

	cm.logger.Info("New configurations retrieved")
	cm.config.Configs = configs

	return true
}

func (cm *ConfigManager) apply() {
	for _, b := range cm.config.Configs {
		cm.reload(b.Type, b.Blocks)
	}
}

func (cm *ConfigManager) reload(t string, blocks []*api.ConfigBlock) {
	cm.logger.Infof("Applying settings for %s", t)

	if obj := cm.registry.GetReloadable(t); obj != nil {
		// Single object
		if len(blocks) != 1 {
			cm.logger.Errorf("got an invalid number of configs for %s: %d, expected: 1", t, len(blocks))
			return
		}
		config, err := blocks[0].ConfigWithMeta()
		if err != nil {
			cm.logger.Error(err)
			return
		}

		if err := obj.Reload(config); err != nil {
			cm.logger.Error(err)
		}
	} else if obj := cm.registry.GetReloadableList(t); obj != nil {
		// List
		var configs []*reload.ConfigWithMeta
		for _, block := range blocks {
			config, err := block.ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				continue
			}
			configs = append(configs, config)
		}

		if err := obj.Reload(configs); err != nil {
			cm.logger.Error(err)
		}
	}
}
