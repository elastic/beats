// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	reporting "github.com/elastic/beats/x-pack/agent/pkg/reporter"
	fleetreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/fleet"
	logreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/log"
)

// Application is the application interface implemented by the different running mode.
type Application interface {
	Start() error
	Stop() error
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
		return nil, errors.Wrap(err, "initiating application")
	}

	mgmt := defaultManagementConfig()
	err = c.Management.Unpack(mgmt)
	if err != nil {
		return nil, errors.Wrap(err, "initiating application")
	}

	client, err := getKibanaClient(log, mgmt)
	if err != nil {
		return nil, errors.Wrap(err, "initiating application")
	}

	reporter, err := getReporter(c.Management, log, getAgentID(), client)
	if err != nil {
		return nil, err
	}

	router, err := newRouter(log, pipelineFactory(config, client, reporter))
	if err != nil {
		return nil, errors.Wrap(err, "initiating application")
	}

	switch mgmt.Mode {
	case localMode:
		log.Info("Agent is managed locally")
		return newLocal(log, pathConfigFile, c.Management, router)
	case fleetMode:
		log.Info("Agent is managed by Fleet")
		return newManaged(log, c.Management, router, client)
	default:
		return nil, ErrInvalidMgmtMode
	}
}

func getKibanaClient(log *logger.Logger, c *ManagementConfig) (_ sender, err error) {
	if c.Mode == localMode {
		// fleet not configured client not needed
		return nil, nil
	}

	defer func() {
		if err != nil {
			err = errors.Wrap(err, "fail to create the API client")
		}
	}()

	if c.Fleet == nil {
		return nil, errors.New("fleet mode enabled but management.fleet not specified")
	}

	kibanaConfig := c.Fleet.Kibana
	if kibanaConfig == nil {
		return nil, errors.New("fleet mode enabled but management.fleet.kibana not specified")
	}

	if c.Fleet.AccessAPIKey != "" {
		rawConfig, err := config.NewConfigFrom(kibanaConfig)
		if err != nil {
			return nil, err
		}

		return fleetapi.NewAuthWithConfig(log, rawConfig, c.Fleet.AccessAPIKey)
	}

	return fleetapi.NewWithConfig(log, kibanaConfig)
}

func getReporter(cfg *config.Config, log *logger.Logger, id string, client sender) (reporter, error) {
	c := defaultManagementConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}

	backends := make([]reporting.Backend, 0, 2)
	backends = append(backends, logreporter.NewReporter(log, c.Reporting.LogReporting))

	if c.Mode == fleetMode && c.Reporting.FleetManagement != nil && c.Reporting.FleetManagement.Enabled {
		agentID := getAgentID()

		fr, err := fleetreporter.NewReporter(agentID, log, c.Reporting.FleetManagement, client)
		if err != nil {
			return nil, err
		}

		backends = append(backends, fr)
	}

	return reporting.NewReporter(log, id, backends...), nil
}

func getAgentID() string {
	// TODO: implement correct way of acquiring agent ID
	return "default"
}
