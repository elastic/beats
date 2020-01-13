// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/application/info"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/storage"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	reporting "github.com/elastic/beats/x-pack/agent/pkg/reporter"
	fleetreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/fleet"
	logreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/log"
)

var gatewaySettings = &fleetGatewaySettings{
	Duration: 2 * time.Second,
	Jitter:   1 * time.Second,
}

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
	log       *logger.Logger
	Config    FleetAgentConfig
	api       apiClient
	agentInfo *info.AgentInfo
	gateway   *fleetGateway
}

func newManaged(
	log *logger.Logger,
	rawConfig *config.Config,
) (*Managed, error) {

	agentInfo, err := info.NewAgentInfo()
	if err != nil {
		return nil, err
	}

	path := fleetAgentConfigPath()

	// TODO(ph): Define the encryption password.
	store := storage.NewEncryptedDiskStore(path, []byte(""))
	reader, err := store.Load()
	if err != nil {
		return nil, errors.New(err, "could not initialize config store",
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	config, err := config.NewConfigFrom(reader)
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("fail to read configuration %s for the agent", path),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	rawConfig.Merge(config)

	cfg := defaultFleetAgentConfig()
	if err := config.Unpack(cfg); err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("fail to unpack configuration from %s", path),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, path))
	}

	client, err := fleetapi.NewAuthWithConfig(log, cfg.API.AccessAPIKey, cfg.API.Kibana)
	if err != nil {
		return nil, errors.New(err,
			"fail to create API client",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, cfg.API.Kibana.Host))
	}

	managedApplication := &Managed{
		log:       log,
		agentInfo: agentInfo,
	}

	logR := logreporter.NewReporter(log, cfg.Reporting.Log)
	fleetR, err := fleetreporter.NewReporter(agentInfo, log, cfg.Reporting.Fleet)
	if err != nil {
		return nil, errors.New(err, "fail to create reporters")
	}

	combinedReporter := reporting.NewReporter(log, agentInfo, logR, fleetR)

	router, err := newRouter(log, streamFactory(rawConfig, client, combinedReporter))
	if err != nil {
		return nil, errors.New(err, "fail to initialize pipeline router")
	}

	emit := emitter(log, router)
	acker, err := newActionAcker(log, agentInfo, client)
	if err != nil {
		return nil, err
	}

	batchedAcker := newLazyAcker(acker)

	actionDispatcher, err := newActionDispatcher(log, &handlerDefault{log: log})
	if err != nil {
		return nil, err
	}

	actionDispatcher.MustRegister(
		&fleetapi.ActionPolicyChange{},
		&handlerPolicyChange{log: log, emitter: emit},
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionUnknown{},
		&handlerUnknown{log: log},
	)

	gateway, err := newFleetGateway(
		log,
		gatewaySettings,
		agentInfo,
		client,
		actionDispatcher,
		fleetR,
		batchedAcker,
	)
	if err != nil {
		return nil, err
	}

	managedApplication.gateway = gateway
	return managedApplication, nil
}

// Start starts a managed agent.
func (m *Managed) Start() error {
	m.log.Info("Agent is starting")
	m.gateway.Start()
	return nil
}

// Stop stops a managed agent.
func (m *Managed) Stop() error {
	defer m.log.Info("Agent is stopped")
	m.gateway.Stop()
	return nil
}

// AgentInfo retrieves agent information.
func (m *Managed) AgentInfo() *info.AgentInfo {
	return m.agentInfo
}
