// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/operation"
	operatorCfg "github.com/elastic/beats/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/stateresolver"
	downloader "github.com/elastic/beats/x-pack/agent/pkg/artifact/download/localremote"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact/install"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

// EventProcessor is an processor of application event
type reporter interface {
	OnStarting(ctx context.Context, app string)
	OnRunning(ctx context.Context, app string)
	OnFailing(ctx context.Context, app string, err error)
	OnStopping(ctx context.Context, app string)
	OnStopped(ctx context.Context, app string)
	OnFatal(ctx context.Context, app string, err error)
}

type sender interface {
	Send(
		ctx context.Context,
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)
}

type operatorStream struct {
	configHandler ConfigHandler
	log           *logger.Logger
}

func (b *operatorStream) Close() error {
	return b.configHandler.HandleConfig(&configRequest{})
}

func (b *operatorStream) Execute(cfg *configRequest) error {
	return b.configHandler.HandleConfig(cfg)
}

func streamFactory(ctx context.Context, cfg *config.Config, client sender, r reporter) func(*logger.Logger, routingKey) (stream, error) {
	return func(log *logger.Logger, id routingKey) (stream, error) {
		// new operator per stream to isolate processes without using tags
		operator, err := newOperator(ctx, log, id, cfg, r)
		if err != nil {
			return nil, err
		}

		return &operatorStream{
			log:           log,
			configHandler: operator,
		}, nil
	}
}

func newOperator(ctx context.Context, log *logger.Logger, id routingKey, config *config.Config, r reporter) (*operation.Operator, error) {
	operatorConfig := &operatorCfg.Config{}
	if err := config.Unpack(&operatorConfig); err != nil {
		return nil, err
	}

	fetcher := downloader.NewDownloader(operatorConfig.DownloadConfig)
	installer, err := install.NewInstaller(operatorConfig.DownloadConfig)
	if err != nil {
		return nil, errors.New(err, "initiating installer")
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
		installer,
		stateResolver,
		r,
	)
}
