// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/application/info"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/dir"
	reporting "github.com/elastic/beats/x-pack/agent/pkg/reporter"
	logreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/log"
)

type emitterFunc func(*config.Config) error

// ConfigHandler is capable of handling config and perform actions at it.
type ConfigHandler interface {
	HandleConfig(configrequest.Request) error
}

type discoverFunc func() ([]string, error)

// ErrNoConfiguration is returned when no configuration are found.
var ErrNoConfiguration = errors.New("no configuration found", errors.TypeConfig)

// Local represents a standalone agents, that will read his configuration directly from disk.
// Some part of the configuration can be reloaded.
type Local struct {
	bgContext   context.Context
	cancelCtxFn context.CancelFunc
	log         *logger.Logger
	source      source
	agentInfo   *info.AgentInfo
}

type source interface {
	Start() error
	Stop() error
}

// newLocal return a agent managed by local configuration.
func newLocal(
	ctx context.Context,
	log *logger.Logger,
	pathConfigFile string,
	config *config.Config,
) (*Local, error) {
	var err error
	if log == nil {
		log, err = logger.New()
		if err != nil {
			return nil, err
		}
	}
	agentInfo, err := info.NewAgentInfo()
	if err != nil {
		return nil, err
	}

	c := localConfigDefault()
	if err := config.Unpack(c); err != nil {
		return nil, errors.New(err, "initialize local mode")
	}

	logR := logreporter.NewReporter(log, c.Management.Reporting)

	localApplication := &Local{
		log:       log,
		agentInfo: agentInfo,
	}

	localApplication.bgContext, localApplication.cancelCtxFn = context.WithCancel(ctx)

	reporter := reporting.NewReporter(localApplication.bgContext, log, localApplication.agentInfo, logR)

	router, err := newRouter(log, streamFactory(localApplication.bgContext, config, nil, reporter))
	if err != nil {
		return nil, errors.New(err, "fail to initialize pipeline router")
	}

	discover := discoverer(pathConfigFile, c.Management.Path)
	emit := emitter(log, router, injectMonitoring)

	var cfgSource source
	if !c.Management.Reload.Enabled {
		log.Debug("Reloading of configuration is off")
		cfgSource = newOnce(log, discover, emit)
	} else {
		log.Debugf("Reloading of configuration is on, frequency is set to %s", c.Management.Reload.Period)
		cfgSource = newPeriodic(log, c.Management.Reload.Period, discover, emit)
	}

	localApplication.source = cfgSource

	return localApplication, nil
}

// Start starts a local agent.
func (l *Local) Start() error {
	l.log.Info("Agent is starting")
	defer l.log.Info("Agent is stopped")

	if err := l.source.Start(); err != nil {
		return err
	}

	return nil
}

// Stop stops a local agent.
func (l *Local) Stop() error {
	l.cancelCtxFn()
	return l.source.Stop()
}

// AgentInfo retrieves agent information.
func (l *Local) AgentInfo() *info.AgentInfo {
	return l.agentInfo
}

func discoverer(patterns ...string) discoverFunc {
	var p []string
	for _, newP := range patterns {
		if len(newP) == 0 {
			continue
		}

		p = append(p, newP)
	}

	if len(p) == 0 {
		return func() ([]string, error) {
			return []string{}, ErrNoConfiguration
		}
	}

	return func() ([]string, error) {
		return dir.DiscoverFiles(p...)
	}
}
