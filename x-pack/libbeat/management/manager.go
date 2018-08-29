// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/management"
)

func init() {
	management.Register("x-pack", NewConfigManager, feature.Beta)
}

// ConfigManager handles internal config updates. By retrieving
// new configs from Kibana and applying them to the Beat
type ConfigManager struct {
	Config *Config
	logger *logp.Logger
}

// NewConfigManager returns a X-Pack Beats Central Management manager
func NewConfigManager() (management.ConfigManager, error) {
	c := &Config{}
	if err := c.Load(); err != nil {
		return nil, errors.Wrap(err, "reading central management internal settings")
	}

	return &ConfigManager{
		Config: c,
		logger: logp.NewLogger("centralmgmt"),
	}, nil
}

// Enabled returns true if config management is enabled
func (cm *ConfigManager) Enabled() bool {
	return cm.Config.Enabled
}

// Start the config manager
func (cm *ConfigManager) Start() {
	if !cm.Enabled() {
		return
	}
	cfgwarn.Beta("Central management is enabled")
	cm.logger.Info("Starting central management service")
}

// Stop the config manager
func (cm *ConfigManager) Stop() {
	if !cm.Enabled() {
		return
	}
	cm.logger.Info("Stopping central management service")
}

// CheckRawConfig check settings are correct to start the beat
func (cm *ConfigManager) CheckRawConfig(cfg *common.Config) error {
	return nil
}
