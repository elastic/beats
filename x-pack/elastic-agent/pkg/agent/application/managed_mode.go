// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/elastic/go-sysinfo"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filters"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/emitter"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/router"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/stream"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/capabilities"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/acker/fleet"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/acker/lazy"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
	reporting "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter"
	fleetreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/fleet"
	logreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/log"
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
	bgContext   context.Context
	cancelCtxFn context.CancelFunc
	log         *logger.Logger
	Config      configuration.FleetAgentConfig
	api         apiClient
	agentInfo   *info.AgentInfo
	gateway     FleetGateway
	router      pipeline.Dispatcher
	srv         *server.Server
	stateStore  stateStore
	upgrader    *upgrade.Upgrader
}

func newManaged(
	ctx context.Context,
	log *logger.Logger,
	storeSaver storage.Store,
	cfg *configuration.Configuration,
	rawConfig *config.Config,
	reexec reexecManager,
	statusCtrl status.Controller,
	agentInfo *info.AgentInfo,
) (*Managed, error) {
	caps, err := capabilities.Load(paths.AgentCapabilitiesPath(), log, statusCtrl)
	if err != nil {
		return nil, err
	}

	client, err := client.NewAuthWithConfig(log, cfg.Fleet.AccessAPIKey, cfg.Fleet.Kibana)
	if err != nil {
		return nil, errors.New(err,
			"fail to create API client",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, cfg.Fleet.Kibana.Host))
	}

	sysInfo, err := sysinfo.Host()
	if err != nil {
		return nil, errors.New(err,
			"fail to get system information",
			errors.TypeUnexpected)
	}

	managedApplication := &Managed{
		log:       log,
		agentInfo: agentInfo,
	}

	managedApplication.bgContext, managedApplication.cancelCtxFn = context.WithCancel(ctx)
	managedApplication.srv, err = server.NewFromConfig(log, cfg.Settings.GRPC, &operation.ApplicationStatusHandler{})
	if err != nil {
		return nil, errors.New(err, "initialize GRPC listener", errors.TypeNetwork)
	}
	// must start before `Start` is called as Fleet will already try to start applications
	// before `Start` is even called.
	err = managedApplication.srv.Start()
	if err != nil {
		return nil, errors.New(err, "starting GRPC listener", errors.TypeNetwork)
	}

	logR := logreporter.NewReporter(log)
	fleetR, err := fleetreporter.NewReporter(agentInfo, log, cfg.Fleet.Reporting)
	if err != nil {
		return nil, errors.New(err, "fail to create reporters")
	}

	combinedReporter := reporting.NewReporter(managedApplication.bgContext, log, agentInfo, logR, fleetR)
	monitor, err := monitoring.NewMonitor(cfg.Settings)
	if err != nil {
		return nil, errors.New(err, "failed to initialize monitoring")
	}

	router, err := router.New(log, stream.Factory(managedApplication.bgContext, agentInfo, cfg.Settings, managedApplication.srv, combinedReporter, monitor, statusCtrl))
	if err != nil {
		return nil, errors.New(err, "fail to initialize pipeline router")
	}
	managedApplication.router = router

	composableCtrl, err := composable.New(log, rawConfig)
	if err != nil {
		return nil, errors.New(err, "failed to initialize composable controller")
	}

	emit, err := emitter.New(
		managedApplication.bgContext,
		log,
		agentInfo,
		composableCtrl,
		router,
		&pipeline.ConfigModifiers{
			Decorators: []pipeline.DecoratorFunc{injectMonitoring},
			Filters:    []pipeline.FilterFunc{filters.StreamChecker, injectFleet(rawConfig, sysInfo.Info(), agentInfo)},
		},
		caps,
		monitor,
	)
	if err != nil {
		return nil, err
	}
	acker, err := fleet.NewAcker(log, agentInfo, client)
	if err != nil {
		return nil, err
	}

	batchedAcker := lazy.NewAcker(acker, log)

	// Create the state store that will persist the last good policy change on disk.
	stateStore, err := store.NewStateStoreWithMigration(log, paths.AgentActionStoreFile(), paths.AgentStateStoreFile())
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("fail to read action store '%s'", paths.AgentActionStoreFile()))
	}
	managedApplication.stateStore = stateStore
	actionAcker := store.NewStateStoreActionAcker(batchedAcker, stateStore)

	actionDispatcher, err := newActionDispatcher(managedApplication.bgContext, log, &handlerDefault{log: log})
	if err != nil {
		return nil, err
	}

	managedApplication.upgrader = upgrade.NewUpgrader(
		agentInfo,
		cfg.Settings.DownloadConfig,
		log,
		[]context.CancelFunc{managedApplication.cancelCtxFn},
		reexec,
		acker,
		combinedReporter,
		caps)

	policyChanger := &handlerPolicyChange{
		log:       log,
		emitter:   emit,
		agentInfo: agentInfo,
		config:    cfg,
		store:     storeSaver,
	}
	if cfg.Fleet.Server == nil {
		// setters only set when not running a local Fleet Server
		policyChanger.setters = []clientSetter{acker}
	}
	actionDispatcher.MustRegister(
		&fleetapi.ActionPolicyChange{},
		policyChanger,
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionPolicyReassign{},
		&handlerPolicyReassign{log: log},
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionUnenroll{},
		&handlerUnenroll{
			log:        log,
			emitter:    emit,
			dispatcher: router,
			closers:    []context.CancelFunc{managedApplication.cancelCtxFn},
			stateStore: stateStore,
		},
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionUpgrade{},
		&handlerUpgrade{
			upgrader: managedApplication.upgrader,
			log:      log,
		},
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionSettings{},
		&handlerSettings{
			log:       log,
			reexec:    reexec,
			agentInfo: agentInfo,
		},
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionApp{},
		&handlerAppAction{
			srv: managedApplication.srv,
			log: log,
		},
	)

	actionDispatcher.MustRegister(
		&fleetapi.ActionUnknown{},
		&handlerUnknown{log: log},
	)

	actions := stateStore.Actions()

	if len(actions) > 0 && !managedApplication.wasUnenrolled() {
		// TODO(ph) We will need an improvement on fleet, if there is an error while dispatching a
		// persisted action on disk we should be able to ask Fleet to get the latest configuration.
		// But at the moment this is not possible because the policy change was acked.
		if err := store.ReplayActions(log, actionDispatcher, actionAcker, actions...); err != nil {
			log.Errorf("could not recover state, error %+v, skipping...", err)
		}
	}

	gateway, err := newFleetGateway(
		managedApplication.bgContext,
		log,
		agentInfo,
		client,
		actionDispatcher,
		fleetR,
		actionAcker,
		statusCtrl,
		stateStore,
	)
	if err != nil {
		return nil, err
	}
	gateway, err = wrapLocalFleetServer(managedApplication.bgContext, log, cfg.Fleet, rawConfig, gateway, emit)
	if err != nil {
		return nil, err
	}
	// add the gateway to setters, so the gateway can be updated
	// when the hosts for Kibana are updated by the policy.
	policyChanger.setters = append(policyChanger.setters, gateway)

	managedApplication.gateway = gateway
	return managedApplication, nil
}

// Start starts a managed elastic-agent.
func (m *Managed) Start() error {
	m.log.Info("Agent is starting")
	if m.wasUnenrolled() {
		m.log.Warnf("agent was previously unenrolled. To reactivate please reconfigure or enroll again.")
		return nil
	}

	// reload ID because of win7 sync issue
	if err := m.agentInfo.ReloadID(); err != nil {
		return err
	}

	err := m.upgrader.Ack(m.bgContext)
	if err != nil {
		m.log.Warnf("failed to ack update %v", err)
	}

	err = m.gateway.Start()
	if err != nil {
		return err
	}
	return nil
}

// Stop stops a managed elastic-agent.
func (m *Managed) Stop() error {
	defer m.log.Info("Agent is stopped")
	m.cancelCtxFn()
	m.router.Shutdown()
	m.srv.Stop()
	return nil
}

// AgentInfo retrieves elastic-agent information.
func (m *Managed) AgentInfo() *info.AgentInfo {
	return m.agentInfo
}

func (m *Managed) wasUnenrolled() bool {
	actions := m.stateStore.Actions()
	for _, a := range actions {
		if a.Type() == "UNENROLL" {
			return true
		}
	}

	return false
}
