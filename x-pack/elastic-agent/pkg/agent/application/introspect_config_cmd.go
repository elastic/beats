// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// IntrospectConfigCmd is an introspect subcommand that shows configurations of the agent.
type IntrospectConfigCmd struct {
	cfgPath string
}

// NewIntrospectConfigCmd creates a new introspect command.
func NewIntrospectConfigCmd(configPath string,
) (*IntrospectConfigCmd, error) {
	return &IntrospectConfigCmd{
		cfgPath: configPath,
	}, nil
}

// Execute introspects agent configuration.
func (c *IntrospectConfigCmd) Execute() error {
	return c.introspectConfig()
}

func (c *IntrospectConfigCmd) introspectConfig() error {
	cfg, err := loadConfig(c.cfgPath)
	if err != nil {
		return err
	}

	isLocal, err := isLocalMode(cfg)
	if err != nil {
		return err
	}

	if isLocal {
		return printConfig(cfg)
	}

	fleetConfig, err := loadFleetConfig(cfg)
	if err != nil {
		return err
	} else if fleetConfig == nil {
		return fmt.Errorf("no fleet config retrieved yet")
	}

	return printMapStringConfig(fleetConfig)
}

func loadConfig(configPath string) (*config.Config, error) {
	rawConfig, err := config.LoadYAML(configPath)
	if err != nil {
		return nil, err
	}

	if err := InjectAgentConfig(rawConfig); err != nil {
		return nil, err
	}

	return rawConfig, nil
}

func loadFleetConfig(cfg *config.Config) (map[string]interface{}, error) {
	log, err := newErrorLogger()
	if err != nil {
		return nil, err
	}

	as, err := newActionStore(log, storage.NewDiskStore(info.AgentActionStoreFile()))
	if err != nil {
		return nil, err
	}

	for _, c := range as.Actions() {
		cfgChange, ok := c.(*fleetapi.ActionConfigChange)
		if !ok {
			continue
		}

		fmt.Println("Action ID:", cfgChange.ID())
		return cfgChange.Config, nil
	}
	return nil, nil
}

func isLocalMode(rawConfig *config.Config) (bool, error) {
	c := localDefaultConfig()
	if err := rawConfig.Unpack(&c); err != nil {
		return false, errors.New(err, "initiating application")
	}

	managementConfig := struct {
		Mode string `config:"mode" yaml:"mode"`
	}{}

	if err := c.Management.Unpack(&managementConfig); err != nil {
		return false, errors.New(err, "initiating application")
	}
	return managementConfig.Mode == "local", nil
}

func printMapStringConfig(mapStr map[string]interface{}) error {
	data, err := yaml.Marshal(mapStr)
	if err != nil {
		return errors.New(err, "could not marshal to YAML")
	}

	fmt.Println(string(data))
	return nil
}

func printConfig(cfg *config.Config) error {
	mapStr, err := cfg.ToMapStr()
	if err != nil {
		return err
	}

	return printMapStringConfig(mapStr)
}
