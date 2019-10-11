// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/libbeat/feature"

	"github.com/gofrs/uuid"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
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
	config    *Config
	cache     *Cache
	logger    *logp.Logger
	client    api.AuthClienter
	beatUUID  uuid.UUID
	done      chan struct{}
	registry  *reload.Registry
	wg        sync.WaitGroup
	blacklist *ConfigBlacklist
	reporter  *api.EventReporter
	state     *State
	mux       sync.RWMutex
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
	var blacklist *ConfigBlacklist

	log := logp.NewLogger(management.DebugK)

	if c.Enabled {
		log.Warn("DEPRECATED: Central Management is deprecated and will be removed in 8.0")

		var err error

		if err = validateConfig(c); err != nil {
			return nil, errors.Wrap(err, "wrong settings for configurations")
		}

		// Initialize configs blacklist
		blacklist, err = NewConfigBlacklist(c.Blacklist)
		if err != nil {
			return nil, errors.Wrap(err, "wrong settings for configurations blacklist")
		}

		// Initialize central management settings cache
		cache = &Cache{}
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

	authClient := &api.AuthClient{Client: client, AccessToken: c.AccessToken, BeatUUID: beatUUID}

	return &ConfigManager{
		config:    c,
		cache:     cache,
		blacklist: blacklist,
		logger:    log,
		client:    authClient,
		done:      make(chan struct{}),
		beatUUID:  beatUUID,
		registry:  registry,
		reporter: api.NewEventReporter(
			log,
			authClient,
			c.EventsReporter.Period,
			c.EventsReporter.MaxBatchSize,
		),
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

	cm.reporter.Start()
	cm.wg.Add(1)
	go cm.worker()
}

// Stop the config manager
func (cm *ConfigManager) Stop() {
	if !cm.Enabled() {
		return
	}

	// stop collecting configuration
	cm.logger.Info("Stopping central management service")
	close(cm.done)
	cm.wg.Wait()

	// report last state and stop reporting.
	cm.updateState(Stopped)
	cm.reporter.Stop()
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

	cm.updateState(Starting)

	// Start worker loop: fetch + apply + cache new settings
	for {
		select {
		case <-cm.done:
			return
		case <-time.After(period):
		}

		changed := cm.fetch()
		if changed || firstRun {
			cm.updateState(InProgress)
			// configs changed, apply changes
			// TODO only reload the blocks that changed
			if errs := cm.apply(); !errs.IsEmpty() {
				cm.reportErrors(errs)
				cm.updateState(Failed)
				cm.logger.Errorf("Could not apply the configuration, error: %+v", errs)
			} else {
				cm.updateState(Running)
			}
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

func (cm *ConfigManager) reportErrors(errs Errors) {
	for _, err := range errs {
		cm.reporter.AddEvent(err)
	}
}

// fetch configurations from kibana, return true if they changed
func (cm *ConfigManager) fetch() bool {
	cm.logger.Debug("Retrieving new configurations from Kibana")
	configs, err := cm.client.Configuration()

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

	equal, err := api.ConfigBlocksEqual(configs, cm.cache.Configs)
	if err != nil {
		cm.logger.Errorf("error comparing the configurations, will use cached ones: %s", err)
		return false
	}

	if equal {
		cm.logger.Debug("configuration didn't change, sleeping")
		return false
	}

	cm.logger.Info("New configurations retrieved")
	cm.cache.Configs = configs

	return true
}

func (cm *ConfigManager) apply() Errors {
	var errors Errors
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

func (cm *ConfigManager) reload(t string, blocks []*api.ConfigBlock) *Error {
	cm.logger.Infof("Applying settings for %s", t)
	if obj := cm.registry.GetReloadable(t); obj != nil {
		// Single object
		if len(blocks) > 1 {
			err := fmt.Errorf("got an invalid number of configs for %s: %d, expected: 1", t, len(blocks))
			cm.logger.Error(err)
			return newConfigError(err)
		}

		var config *reload.ConfigWithMeta
		var err error
		if len(blocks) == 1 {
			config, err = blocks[0].ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				return newConfigError(err)
			}
		}

		if err := obj.Reload(config); err != nil {
			cm.logger.Error(err)
			return newConfigError(err)
		}
	} else if obj := cm.registry.GetReloadableList(t); obj != nil {
		// List
		var configs []*reload.ConfigWithMeta
		for _, block := range blocks {
			config, err := block.ConfigWithMeta()
			if err != nil {
				cm.logger.Error(err)
				return newConfigError(err)
			}
			configs = append(configs, config)
		}

		if err := obj.Reload(configs); err != nil {
			cm.logger.Error(err)
			return newConfigError(err)
		}
	}

	return nil
}

func (cm *ConfigManager) updateState(state State) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.state = &state
	cm.reporter.AddEvent(&state)
	cm.logger.Infof("Updating state to '%s'", state)
}

func validateConfig(config *Config) error {
	if len(config.AccessToken) == 0 {
		return errEmptyAccessToken
	}
	return nil
}
