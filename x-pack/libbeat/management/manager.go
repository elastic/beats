// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"sync"
	"time"

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
	wg       sync.WaitGroup
}

// NewConfigManager returns a X-Pack Beats Central Management manager
func NewConfigManager(beatUUID uuid.UUID) (management.ConfigManager, error) {
	c := defaultConfig()
	if err := c.Load(); err != nil {
		return nil, errors.Wrap(err, "reading central management internal settings")
	}

	client, err := api.NewClient(c.Kibana)
	if err != nil {
		return nil, errors.Wrap(err, "initializing kibana client")
	}

	return &ConfigManager{
		config:   c,
		logger:   logp.NewLogger("centralmgmt"),
		client:   client,
		done:     make(chan struct{}),
		beatUUID: beatUUID,
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

// CheckRawConfig check settings are correct to start the beat
func (cm *ConfigManager) CheckRawConfig(cfg *common.Config) error {
	return nil
}

func (cm *ConfigManager) worker() {
	defer cm.wg.Done()
	sleep := 0 * time.Second
	for {
		select {
		case <-cm.done:
			return
		case <-time.After(sleep):
			sleep = cm.config.Period
		}
		cm.logger.Debug("Retrieving new configurations from Kibana")
		configs, err := cm.client.Configuration(cm.config.AccessToken, cm.beatUUID)
		if err != nil {
			cm.logger.Errorf("error retriving new configurations: %s", err)
			continue
		}

		if api.ConfigBlocksEqual(configs, cm.config.Configs) {
			cm.logger.Debug("configuration didn't change, sleeping")
			continue
		}

		cm.logger.Info("New configuration retrieved from central management, applying changes...")

		// configs changed, apply changes
		// TODO only reload the blocks that changed
		for _, config := range configs {
			cm.logger.Infof("%+v", config)
		}

		// store new configs (already applied)
		cm.config.Configs = configs
		if err := cm.config.Save(); err != nil {
			cm.logger.Errorf("error storing central management state: %s", err)
		}
	}
}
