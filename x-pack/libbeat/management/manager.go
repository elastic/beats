// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/reload"

	"github.com/gofrs/uuid"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/libbeat/management/api"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/management"
)

var errEmptyAccessToken = errors.New("access_token is empty, you must reenroll your Beat")

func init() {
	management.Register("x-pack", NewConfigManager, feature.Beta)
}

// ConfigManager handles internal config updates. By retrieving
// new configs from Kibana and applying them to the Beat
type ConfigManager struct {
	config   *Config
	cache    *Cache
	logger   *logp.Logger
	client   *api.Client
	beatUUID uuid.UUID
	done     chan struct{}
	registry *reload.Registry
	wg       sync.WaitGroup
}

// NewConfigManager returns a X-Pack Beats Central Management manager
func NewConfigManager(config *common.Config, registry *reload.Registry, beatUUID uuid.UUID) (management.ConfigManager, error) {
	c := defaultConfig()
	if config.Enabled() {
		if err := config.Unpack(&c); err != nil {
			return nil, errors.Wrap(err, "parsing central management settings")
		}
	}
	return NewConfigManagerWithConfig(c, registry, beatUUID)
}

// NewConfigManagerWithConfig returns a X-Pack Beats Central Management manager
func NewConfigManagerWithConfig(c *Config, registry *reload.Registry, beatUUID uuid.UUID) (management.ConfigManager, error) {
	var client *api.Client
	var cache *Cache
	if c.Enabled {
		var err error

		if err = validateConfig(c); err != nil {
			return nil, errors.Wrap(err, "wrong settings for configurations")
		}

		// Initialize central management settings cache
		cache = &Cache{
			ConfigOK: true,
		}
		if err := cache.Load(); err != nil {
			return nil, errors.Wrap(err, "reading central management internal cache")
		}

		// Ignore kibana version to avoid permission errors
		c.Kibana.IgnoreVersion = true

		client, err = api.NewClient(c.Kibana)
		if err != nil {
			return nil, errors.Wrap(err, "initializing kibana client")
		}
	}

	return &ConfigManager{
		config:   c,
		cache:    cache,
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
			if err := cm.cache.Save(); err != nil {
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
	configs, err := cm.client.Configuration(cm.config.AccessToken, cm.beatUUID, cm.cache.ConfigOK)

	if api.IsConfigurationNotFound(err) {
		if cm.cache.HasConfig() {
			cm.logger.Error("Disabling all running configuration because no configurations were found for this Beat, the endpoint returned a 404 or the beat is not enrolled with central management")
			cm.cache.Configs = api.ConfigBlocks{}
		}
		return true
	}

	if err != nil {
		cm.logger.Errorf("error retrieving new configurations, will use cached ones: %s", err)
		return false
	}

	if api.ConfigBlocksEqual(configs, cm.cache.Configs) {
		cm.logger.Debug("configuration didn't change, sleeping")
		return false
	}

	cm.logger.Info("New configurations retrieved")
	cm.cache.Configs = configs

	return true
}

func (cm *ConfigManager) apply() {
	configOK := true

	missing := map[string]bool{}
	for _, name := range cm.registry.GetRegisteredNames() {
		missing[name] = true
	}

	// Reload configs
	for _, b := range cm.cache.Configs {
		err := cm.reload(b.Type, b.Blocks)
		configOK = configOK && err == nil
		missing[b.Type] = false
	}

	// Unset missing configs
	for name := range missing {
		if missing[name] {
			cm.reload(name, []*api.ConfigBlock{})
		}
	}

	if !configOK {
		logp.Info("Failed to apply settings, reporting error on next fetch")
	}

	// Update configOK flag with the result of this apply
	cm.cache.ConfigOK = configOK
}

func (cm *ConfigManager) reload(t string, blocks []*api.ConfigBlock) error {
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
				continue
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

func validateConfig(config *Config) error {
	if len(config.AccessToken) == 0 {
		return errEmptyAccessToken
	}
	return nil
}
