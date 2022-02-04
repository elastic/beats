// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// Application is the application interface implemented by the different running mode.
type Application interface {
	Start() error
	Stop() error
	AgentInfo() *info.AgentInfo
	Routes() *sorted.Set
}

type reexecManager interface {
	ReExec(callback reexec.ShutdownCallbackFn, argOverrides ...string)
}

type upgraderControl interface {
	SetUpgrader(upgrader *upgrade.Upgrader)
}

// New creates a new Agent and bootstrap the required subsystem.
func New(log *logger.Logger, reexec reexecManager, statusCtrl status.Controller, uc upgraderControl, agentInfo *info.AgentInfo) (Application, error) {
	// Load configuration from disk to understand in which mode of operation
	// we must start the elastic-agent, the mode of operation cannot be changed without restarting the
	// elastic-agent.
	pathConfigFile := paths.ConfigFile()
	rawConfig, err := config.LoadFile(pathConfigFile)
	if err != nil {
		return nil, err
	}

	if err := info.InjectAgentConfig(rawConfig); err != nil {
		return nil, err
	}

	return createApplication(log, pathConfigFile, rawConfig, reexec, statusCtrl, uc, agentInfo)
}

func createApplication(
	log *logger.Logger,
	pathConfigFile string,
	rawConfig *config.Config,
	reexec reexecManager,
	statusCtrl status.Controller,
	uc upgraderControl,
	agentInfo *info.AgentInfo,
) (Application, error) {
	log.Info("Detecting execution mode")
	ctx := context.Background()
	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if configuration.IsStandalone(cfg.Fleet) {
		log.Info("Agent is managed locally")
		return newLocal(ctx, log, paths.ConfigFile(), rawConfig, reexec, statusCtrl, uc, agentInfo)
	}

	// not in standalone; both modes require reading the fleet.yml configuration file
	var store storage.Store
	store, cfg, err = mergeFleetConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if configuration.IsFleetServerBootstrap(cfg.Fleet) {
		log.Info("Agent is in Fleet Server bootstrap mode")
		return newFleetServerBootstrap(ctx, log, pathConfigFile, rawConfig, statusCtrl, agentInfo)
	}

	log.Info("Agent is managed by Fleet")
	return newManaged(ctx, log, store, cfg, rawConfig, reexec, statusCtrl, agentInfo)
}

func mergeFleetConfig(rawConfig *config.Config) (storage.Store, *configuration.Configuration, error) {
	path := paths.AgentConfigFile()
	store := storage.NewDiskStore(path)
	reader, err := store.Load()
	if err != nil {
		return store, nil, errors.New(err, "could not initialize config store",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}
	config, err := config.NewConfigFrom(reader)
	if err != nil {
		return store, nil, errors.New(err,
			fmt.Sprintf("fail to read configuration %s for the elastic-agent", path),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	// merge local configuration and configuration persisted from fleet.
	err = rawConfig.Merge(config)
	if err != nil {
		return store, nil, errors.New(err,
			fmt.Sprintf("fail to merge configuration with %s for the elastic-agent", path),
			errors.TypeConfig,
			errors.M(errors.MetaKeyPath, path))
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return store, nil, errors.New(err,
			fmt.Sprintf("fail to unpack configuration from %s", path),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	if err := cfg.Fleet.Valid(); err != nil {
		return store, nil, errors.New(err,
			"fleet configuration is invalid",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	return store, cfg, nil
}
