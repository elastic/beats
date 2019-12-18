// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/configrequest"
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
var ErrNoConfiguration = errors.New("no configuration found")

// Local represents a standalone agents, that will read his configuration directly from disk.
// Some part of the configuration can be reloaded.
type Local struct {
	log    *logger.Logger
	source source
}

type source interface {
	Start() error
	Stop() error
}

// newLocal return a agent managed by local configuration.
func newLocal(
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

	agentID := getAgentID()

	c := localConfigDefault()
	if err := config.Unpack(c); err != nil {
		return nil, errors.Wrap(err, "initialize local mode")
	}

	logR := logreporter.NewReporter(log, c.Reporting)

	reporter := reporting.NewReporter(log, agentID, logR)

	router, err := newRouter(log, streamFactory(config, nil, reporter))
	if err != nil {
		return nil, errors.Wrap(err, "fail to initialize pipeline router")
	}

	discover := discoverer(pathConfigFile, c.Path)
	emit := emitter(log, router, injectMonitoring)

	var cfgSource source
	if !c.Reload.Enabled {
		log.Debug("Reloading of configuration is off")
		cfgSource = newOnce(log, discover, emit)
	} else {
		log.Debugf("Reloading of configuration is on, frequency is set to %s", c.Reload.Period)
		cfgSource = newPeriodic(log, c.Reload.Period, discover, emit)
	}

	return &Local{
		log:    log,
		source: cfgSource,
	}, nil
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
	return l.source.Stop()
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
