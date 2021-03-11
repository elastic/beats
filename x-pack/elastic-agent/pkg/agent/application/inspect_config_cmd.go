// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/capabilities"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config/operations"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// InspectConfigCmd is an inspect subcommand that shows configurations of the agent.
type InspectConfigCmd struct {
	cfgPath string
}

// NewInspectConfigCmd creates a new inspect command.
func NewInspectConfigCmd(configPath string,
) (*InspectConfigCmd, error) {
	return &InspectConfigCmd{
		cfgPath: configPath,
	}, nil
}

// Execute inspects agent configuration.
func (c *InspectConfigCmd) Execute() error {
	return c.inspectConfig()
}

func (c *InspectConfigCmd) inspectConfig() error {
	fullCfg, err := operations.LoadFullAgentConfig(c.cfgPath)
	if err != nil {
		return err
	}

	return printConfig(fullCfg)
}

func loadConfig(configPath string) (*config.Config, error) {
	rawConfig, err := config.LoadFile(configPath)
	if err != nil {
		return nil, err
	}

	path := paths.AgentConfigFile()

	store := storage.NewDiskStore(path)
	reader, err := store.Load()
	if err != nil {
		return nil, errors.New(err, "could not initialize config store",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	config, err := config.NewConfigFrom(reader)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("fail to read configuration %s for the elastic-agent", path),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	// merge local configuration and configuration persisted from fleet.
	rawConfig.Merge(config)

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

	stateStore, err := store.NewStateStoreWithMigration(log, paths.AgentActionStoreFile(), paths.AgentStateStoreFile())
	if err != nil {
		return nil, err
	}

	for _, c := range stateStore.Actions() {
		cfgChange, ok := c.(*fleetapi.ActionPolicyChange)
		if !ok {
			continue
		}

		fmt.Println("Action ID:", cfgChange.ID())
		return cfgChange.Policy, nil
	}
	return nil, nil
}

func printMapStringConfig(mapStr map[string]interface{}) error {
	l, err := newErrorLogger()
	if err != nil {
		return err
	}
	caps, err := capabilities.Load(paths.AgentCapabilitiesPath(), l, status.NewController(l))
	if err != nil {
		return err
	}

	newCfg, err := caps.Apply(mapStr)
	if err != nil {
		return errors.New(err, "failed to apply capabilities")
	}
	newMap, ok := newCfg.(map[string]interface{})
	if !ok {
		return errors.New("config returned from capabilities has invalid type")
	}

	data, err := yaml.Marshal(newMap)
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
