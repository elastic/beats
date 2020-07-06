// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation"
	operatorCfg "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/stateresolver"
	downloader "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

type operatorStream struct {
	configHandler ConfigHandler
	log           *logger.Logger
	monitor       monitoring.Monitor
}

func (b *operatorStream) Close() error {
	return b.configHandler.HandleConfig(&configRequest{})
}

func (b *operatorStream) Execute(cfg *configRequest) error {
	return b.configHandler.HandleConfig(cfg)
}

func streamFactory(ctx context.Context, cfg *config.Config, srv *server.Server, r state.Reporter, m monitoring.Monitor) func(*logger.Logger, routingKey) (stream, error) {
	return func(log *logger.Logger, id routingKey) (stream, error) {
		// new operator per stream to isolate processes without using tags
		operator, err := newOperator(ctx, log, id, cfg, srv, r, m)
		if err != nil {
			return nil, err
		}

		return &operatorStream{
			log:           log,
			configHandler: operator,
		}, nil
	}
}

func newOperator(ctx context.Context, log *logger.Logger, id routingKey, config *config.Config, srv *server.Server, r state.Reporter, m monitoring.Monitor) (*operation.Operator, error) {
	operatorConfig := operatorCfg.DefaultConfig()
	if err := config.Unpack(&operatorConfig); err != nil {
		return nil, err
	}

	fetcher := downloader.NewDownloader(log, operatorConfig.DownloadConfig)
	verifier, err := downloader.NewVerifier(log, operatorConfig.DownloadConfig)
	if err != nil {
		return nil, errors.New(err, "initiating verifier")
	}

	installer, err := install.NewInstaller(operatorConfig.DownloadConfig)
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
	)
}
