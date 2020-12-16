// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/stateresolver"
	downloader "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

type operatorStream struct {
	configHandler ConfigHandler
	log           *logger.Logger
	monitor       monitoring.Monitor
}

func (b *operatorStream) Close() error {
	return b.configHandler.Close()
}

func (b *operatorStream) Execute(cfg configrequest.Request) error {
	return b.configHandler.HandleConfig(cfg)
}

func (b *operatorStream) Shutdown() {
	b.configHandler.Shutdown()
}

func streamFactory(ctx context.Context, agentInfo *info.AgentInfo, cfg *configuration.SettingsConfig, srv *server.Server, r state.Reporter, m monitoring.Monitor, statusController status.Controller) func(*logger.Logger, routingKey) (stream, error) {
	return func(log *logger.Logger, id routingKey) (stream, error) {
		// new operator per stream to isolate processes without using tags
		operator, err := newOperator(ctx, log, agentInfo, id, cfg, srv, r, m, statusController)
		if err != nil {
			return nil, err
		}

		return &operatorStream{
			log:           log,
			configHandler: operator,
		}, nil
	}
}

func newOperator(ctx context.Context, log *logger.Logger, agentInfo *info.AgentInfo, id routingKey, config *configuration.SettingsConfig, srv *server.Server, r state.Reporter, m monitoring.Monitor, statusController status.Controller) (*operation.Operator, error) {
	fetcher := downloader.NewDownloader(log, config.DownloadConfig)
	allowEmptyPgp, pgp := release.PGP()
	verifier, err := downloader.NewVerifier(log, config.DownloadConfig, allowEmptyPgp, pgp)
	if err != nil {
		return nil, errors.New(err, "initiating verifier")
	}

	installer, err := install.NewInstaller(config.DownloadConfig)
	if err != nil {
		return nil, errors.New(err, "initiating installer")
	}

	uninstaller, err := uninstall.NewUninstaller()
	if err != nil {
		return nil, errors.New(err, "initiating uninstaller")
	}

	stateResolver, err := stateresolver.NewStateResolver(log)
	if err != nil {
		return nil, err
	}

	return operation.NewOperator(
		ctx,
		log,
		agentInfo,
		id,
		config,
		fetcher,
		verifier,
		installer,
		uninstaller,
		stateResolver,
		srv,
		r,
		m,
		statusController,
	)
}
