// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/storage"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"

	reporting "github.com/elastic/beats/x-pack/agent/pkg/reporter"
	fleetreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/fleet"
	logreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/log"
)

type apiClient interface {
	Send(
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)
}

// Managed application, when the application is run in managed mode, most of the configuration are
// coming from the Fleet App.
type Managed struct {
	log    *logger.Logger
	Config FleetAgentConfig
	api    apiClient
}

func newManaged(
	log *logger.Logger,
	rawConfig *config.Config,
) (*Managed, error) {

	agentID := getAgentID()

	path := fleetAgentConfigPath()

	// TODO(ph): Define the encryption password.
	store := storage.NewEncryptedDiskStore(path, []byte(""))
	reader, err := store.Load()
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize config store")
	}

	config, err := config.NewConfigFrom(reader)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to read configuration %s for the agent", path)
	}

	cfg := defaultFleetAgentConfig()
	if err := config.Unpack(cfg); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack configuration from %s", path)
	}

	client, err := fleetapi.NewAuthWithConfig(log, cfg.API.AccessAPIKey, cfg.API.Kibana)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create API client")
	}

	reporter, err := createFleetReporters(log, cfg, agentID, client)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create reporters")
	}

	// TODO(michal, ph) Link router with configuration
	_, err = newRouter(log, streamFactory(config, client, reporter))
	if err != nil {
		return nil, errors.Wrap(err, "fail to initialize pipeline router")
	}

	return &Managed{
		log: log,
		api: client,
	}, nil
}

// Start starts a managed agent.
func (m *Managed) Start() error {
	m.log.Info("Agent is starting")
	defer m.log.Info("Agent is stopped")
	return nil
}

// Stop stops a managed agent.
func (m *Managed) Stop() error {
	return nil
}

func createFleetReporters(
	log *logger.Logger,
	cfg *FleetAgentConfig,
	agentID string,
	client apiClient,
) (reporter, error) {

	logR := logreporter.NewReporter(log, cfg.Reporting.Log)

	fleetR, err := fleetreporter.NewReporter(agentID, log, cfg.Reporting.Fleet, client)
	if err != nil {
		return nil, err
	}

	return reporting.NewReporter(log, agentID, logR, fleetR), nil
}
