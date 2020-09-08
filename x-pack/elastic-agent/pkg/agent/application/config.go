// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io/ioutil"

	"github.com/elastic/go-ucfg"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
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

// LoadConfigFromFile loads the Agent configuration from a file.
//
// This must be used to load the Agent configuration, so that variables defined in the inputs are not
// parsed by go-ucfg. Variables from the inputs should be parsed by the transpiler.
func LoadConfigFromFile(path string) (*config.Config, error) {
	in, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(in, &m); err != nil {
		return nil, err
	}
	return LoadConfig(m)
}

// LoadConfig loads the Agent configuration from a map.
//
// This must be used to load the Agent configuration, so that variables defined in the inputs are not
// parsed by go-ucfg. Variables from the inputs should be parsed by the transpiler.
func LoadConfig(m map[string]interface{}) (*config.Config, error) {
	inputs, ok := m["inputs"]
	if ok {
		// remove the inputs
		delete(m, "inputs")
	}
	cfg, err := config.NewConfigFrom(m)
	if err != nil {
		return nil, err
	}
	if ok {
		inputsOnly := map[string]interface{}{
			"inputs": inputs,
		}
		// convert to config without variable substitution
		inputsCfg, err := config.NewConfigFrom(inputsOnly, ucfg.PathSep("."), ucfg.ResolveNOOP)
		if err != nil {
			return nil, err
		}
		err = cfg.Merge(inputsCfg, ucfg.PathSep("."), ucfg.ResolveNOOP)
		if err != nil {
			return nil, err
		}
	}
	return cfg, err
}
