// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"go.elastic.co/apm"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
	"github.com/elastic/go-sysinfo"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filters"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/emitter/modifiers"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/router"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/stream"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	monitoringCfg "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	reporting "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter"
	logreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/log"
)

// FleetServerBootstrap application, does just enough to get a Fleet Server up and running so enrollment
// can complete.
type FleetServerBootstrap struct {
	bgContext   context.Context
	cancelCtxFn context.CancelFunc
	log         *logger.Logger
	Config      configuration.FleetAgentConfig
	agentInfo   *info.AgentInfo
	router      pipeline.Router
	source      source
	srv         *server.Server
}

func newFleetServerBootstrap(
	ctx context.Context,
	log *logger.Logger,
	pathConfigFile string,
	rawConfig *config.Config,
	statusCtrl status.Controller,
	agentInfo *info.AgentInfo,
	tracer *apm.Tracer,
) (*FleetServerBootstrap, error) {
	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	if log == nil {
		log, err = logger.NewFromConfig("", cfg.Settings.LoggingConfig, false)
		if err != nil {
			return nil, err
		}
	}

	logR := logreporter.NewReporter(log)

	sysInfo, err := sysinfo.Host()
	if err != nil {
		return nil, errors.New(err,
			"fail to get system information",
			errors.TypeUnexpected)
	}

	bootstrapApp := &FleetServerBootstrap{
		log:       log,
		agentInfo: agentInfo,
	}

	bootstrapApp.bgContext, bootstrapApp.cancelCtxFn = context.WithCancel(ctx)
	bootstrapApp.srv, err = server.NewFromConfig(log, cfg.Settings.GRPC, &operation.ApplicationStatusHandler{}, tracer)
	if err != nil {
		return nil, errors.New(err, "initialize GRPC listener")
	}

	reporter := reporting.NewReporter(bootstrapApp.bgContext, log, bootstrapApp.agentInfo, logR)

	if cfg.Settings.MonitoringConfig != nil {
		cfg.Settings.MonitoringConfig.Enabled = false
	} else {
		cfg.Settings.MonitoringConfig = &monitoringCfg.MonitoringConfig{Enabled: false}
	}
	monitor, err := monitoring.NewMonitor(cfg.Settings)
	if err != nil {
		return nil, errors.New(err, "failed to initialize monitoring")
	}

	router, err := router.New(log, stream.Factory(bootstrapApp.bgContext, agentInfo, cfg.Settings, bootstrapApp.srv, reporter, monitor, statusCtrl))
	if err != nil {
		return nil, errors.New(err, "fail to initialize pipeline router")
	}
	bootstrapApp.router = router

	emit, err := bootstrapEmitter(
		bootstrapApp.bgContext,
		log,
		agentInfo,
		router,
		&pipeline.ConfigModifiers{
			Filters: []pipeline.FilterFunc{filters.StreamChecker, modifiers.InjectFleet(rawConfig, sysInfo.Info(), agentInfo)},
		},
	)
	if err != nil {
		return nil, err
	}

	discover := discoverer(pathConfigFile, cfg.Settings.Path)
	bootstrapApp.source = newOnce(log, discover, emit)
	return bootstrapApp, nil
}

// Routes returns a list of routes handled by server.
func (b *FleetServerBootstrap) Routes() *sorted.Set {
	return b.router.Routes()
}

// Start starts a managed elastic-agent.
func (b *FleetServerBootstrap) Start() error {
	b.log.Info("Agent is starting")
	defer b.log.Info("Agent is stopped")

	if err := b.srv.Start(); err != nil {
		return err
	}
	if err := b.source.Start(); err != nil {
		return err
	}

	return nil
}

// Stop stops a local agent.
func (b *FleetServerBootstrap) Stop() error {
	err := b.source.Stop()
	b.cancelCtxFn()
	b.router.Shutdown()
	b.srv.Stop()
	return err
}

// AgentInfo retrieves elastic-agent information.
func (b *FleetServerBootstrap) AgentInfo() *info.AgentInfo {
	return b.agentInfo
}

func bootstrapEmitter(ctx context.Context, log *logger.Logger, agentInfo transpiler.AgentInfo, router pipeline.Router, modifiers *pipeline.ConfigModifiers) (pipeline.EmitterFunc, error) {
	ch := make(chan *config.Config)

	go func() {
		for {
			var c *config.Config
			select {
			case <-ctx.Done():
				return
			case c = <-ch:
			}

			err := emit(ctx, log, agentInfo, router, modifiers, c)
			if err != nil {
				log.Error(err)
			}
		}
	}()

	return func(ctx context.Context, c *config.Config) error {
		span, _ := apm.StartSpan(ctx, "emit", "app.internal")
		defer span.End()
		ch <- c
		return nil
	}, nil
}

func emit(ctx context.Context, log *logger.Logger, agentInfo transpiler.AgentInfo, router pipeline.Router, modifiers *pipeline.ConfigModifiers, c *config.Config) error {
	if err := info.InjectAgentConfig(c); err != nil {
		return err
	}

	// perform and verify ast translation
	m, err := c.ToMapStr()
	if err != nil {
		return errors.New(err, "could not create the AST from the configuration", errors.TypeConfig)
	}
	ast, err := transpiler.NewAST(m)
	if err != nil {
		return errors.New(err, "could not create the AST from the configuration", errors.TypeConfig)
	}
	for _, filter := range modifiers.Filters {
		if err := filter(log, ast); err != nil {
			return errors.New(err, "failed to filter configuration", errors.TypeConfig)
		}
	}

	// overwrite the inputs to only have a single fleet-server input
	transpiler.Insert(ast, transpiler.NewList([]transpiler.Node{
		transpiler.NewDict([]transpiler.Node{
			transpiler.NewKey("type", transpiler.NewStrVal("fleet-server")),
		}),
	}), "inputs")

	spec, ok := program.SupportedMap["fleet-server"]
	if !ok {
		return errors.New("missing required fleet-server program specification")
	}
	ok, err = program.DetectProgram(spec, agentInfo, ast)
	if err != nil {
		return errors.New(err, "failed parsing the configuration")
	}
	if !ok {
		return errors.New("bootstrap configuration is incorrect causing fleet-server to not be started")
	}

	return router.Route(ctx, ast.HashStr(), map[pipeline.RoutingKey][]program.Program{
		pipeline.DefaultRK: {
			{
				Spec:   spec,
				Config: ast,
			},
		},
	})
}
