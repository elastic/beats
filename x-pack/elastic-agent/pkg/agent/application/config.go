// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
)

type localConfig struct {
	Fleet    *configuration.FleetAgentConfig `config:"fleet"`
	Settings *configuration.SettingsConfig   `config:"agent" yaml:"agent"`
}

func createFleetConfigFromEnroll(accessAPIKey string, kbn *kibana.Config) (*configuration.FleetAgentConfig, error) {
	cfg := configuration.DefaultFleetAgentConfig()
	cfg.Enabled = true
	cfg.AccessAPIKey = accessAPIKey
	cfg.Kibana = kbn

	if err := cfg.Valid(); err != nil {
		return nil, errors.New(err, "invalid enrollment options", errors.TypeConfig)
	}
	return cfg, nil
}
