// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/x-pack/agent/pkg/agent/application/info"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
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
	// we must start the agent, the mode of operation cannot be changed without restarting the
	// agent.
	config, err := config.LoadYAML(pathConfigFile)
	if err != nil {
		return nil, err
	}

	if err := InjectAgentConfig(config); err != nil {
		return nil, err
	}

	return createApplication(log, pathConfigFile, config)
}

func createApplication(
	log *logger.Logger,
	pathConfigFile string,
	config *config.Config,
) (Application, error) {

	log.Info("Detecting execution mode")
	c := localDefaultConfig()
	err := config.Unpack(c)
	if err != nil {
		return nil, errors.New(err, "initiating application")
	}

	mgmt := defaultManagementConfig()
	err = c.Management.Unpack(mgmt)
	if err != nil {
		return nil, errors.New(err, "initiating application")
	}

	switch mgmt.Mode {
	case localMode:
		log.Info("Agent is managed locally")
		return newLocal(log, pathConfigFile, config)
	case fleetMode:
		log.Info("Agent is managed by Fleet")
		return newManaged(log, config)
	default:
		return nil, ErrInvalidMgmtMode
	}
}
