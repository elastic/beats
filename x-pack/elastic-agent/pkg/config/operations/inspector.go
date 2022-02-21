// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operations

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

var (
	// ErrNoFleetConfig is returned when no configuration was retrieved from fleet just yet.
	ErrNoFleetConfig = fmt.Errorf("no fleet config retrieved yet")
)

// LoadFullAgentConfig load agent config based on provided paths and defined capabilities.
// In case fleet is used, config from policy action is returned.
func LoadFullAgentConfig(cfgPath string, failOnFleetMissing bool) (*config.Config, error) {
	rawConfig, err := loadConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if configuration.IsStandalone(cfg.Fleet) {
		return rawConfig, nil
	}

	fleetConfig, err := loadFleetConfig(rawConfig)
	if err != nil {
		return nil, err
	} else if fleetConfig == nil {
		if failOnFleetMissing {
			return nil, ErrNoFleetConfig
		}

		// resolving fleet config but not fleet config retrieved yet, returning last applied config
		return rawConfig, nil
	}

	return config.NewConfigFrom(fleetConfig)
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

	log, err := logger.New("sdh-logger", false)
	if err != nil {
		panic(fmt.Errorf("could not create a sdh-debug logger: %w", err))
	}

	mapstr, err := rawConfig.ToMapStr()
	if err != nil {
		log.Errorf("could not parse rawConfig to ToMapStr: %v", err)
	}
	log.With("rawConfig", fmt.Sprintf("%#v", rawConfig)).
		Infof("[loadConfig] rawConfig")
	log.With("mapstr", fmt.Sprintf("%#v", mapstr)).
		Infof("[loadConfig] rawConfig map[str]")

	mapstr, err = config.ToMapStr()
	if err != nil {
		log.Errorf("could not parse config to ToMapStr failed: %v", err)
	}
	log.With("config", fmt.Sprintf("%#v", config)).
		Infof("[loadConfig] config")
	log.With("config", fmt.Sprintf("%#v", mapstr)).
		Infof("[loadConfig] config map[str]")

	// merge local configuration and configuration persisted from fleet.
	if err = rawConfig.Merge(config); err != nil {
		return nil, fmt.Errorf("failed merging configs: %w", err)
	}

	mapstr, err = rawConfig.ToMapStr()
	if err != nil {
		log.Errorf("could not parse merged rawConfig to ToMapStr: %v", err)
	}
	log.With("merged.rawConfig", fmt.Sprintf("%#v", rawConfig)).
		Infof("[loadConfig] merged rawConfig")
	log.With("merged.rawConfig", fmt.Sprintf("%#v", mapstr)).
		Infof("[loadConfig] merged rawConfig map[str]")

	if err := info.InjectAgentConfig(rawConfig); err != nil {
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

		return cfgChange.Policy, nil
	}
	return nil, nil
}

func newErrorLogger() (*logger.Logger, error) {
	return logger.NewWithLogpLevel("", logp.ErrorLevel, false)
}
