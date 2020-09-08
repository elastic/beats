// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/warn"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// Application is the application interface implemented by the different running mode.
type Application interface {
	Start() error
	Stop() error
	AgentInfo() *info.AgentInfo
}

// New creates a new Agent and bootstrap the required subsystem.
func New(log *logger.Logger, pathConfigFile string) (Application, error) {
	// Load configuration from disk to understand in which mode of operation
	// we must start the elastic-agent, the mode of operation cannot be changed without restarting the
	// elastic-agent.
	rawConfig, err := LoadConfigFromFile(pathConfigFile)
	if err != nil {
		return nil, err
	}

	if err := InjectAgentConfig(rawConfig); err != nil {
		return nil, err
	}

	return createApplication(log, pathConfigFile, rawConfig)
}

func createApplication(
	log *logger.Logger,
	pathConfigFile string,
	rawConfig *config.Config,
) (Application, error) {
	warn.LogNotGA(log)
	log.Info("Detecting execution mode")
	ctx := context.Background()

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if isStandalone(cfg.Fleet) {
		log.Info("Agent is managed locally")
		return newLocal(ctx, log, pathConfigFile, rawConfig)
	}

	log.Info("Agent is managed by Fleet")
	return newManaged(ctx, log, rawConfig)
}

// missing of fleet.enabled: true or fleet.{access_token,kibana} will place Elastic Agent into standalone mode.
func isStandalone(cfg *configuration.FleetAgentConfig) bool {
	return cfg == nil || !cfg.Enabled
}
