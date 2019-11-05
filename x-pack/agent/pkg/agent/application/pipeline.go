// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/operation"
	operatorCfg "github.com/elastic/beats/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/stateresolver"
	downloader "github.com/elastic/beats/x-pack/agent/pkg/artifact/download/http"
	"github.com/elastic/beats/x-pack/agent/pkg/artifact/install"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

// EventProcessor is an processor of application event
type reporter interface {
	OnStarting(app string)
	OnRunning(app string)
	OnFailing(app string, err error)
	OnStopping(app string)
	OnStopped(app string)
	OnFatal(app string, err error)
}

type sender interface {
	Send(
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)
}

type operatorPipeline struct {
	configHandler ConfigHandler
	log           *logger.Logger
}

func (b *operatorPipeline) Close() error {
	return nil
}

func (b *operatorPipeline) Execute(cfg *configRequest) error {
	return b.configHandler.HandleConfig(cfg)
}

func pipelineFactory(cfg *config.Config, client sender, r reporter) func(*logger.Logger, routingKey) (pipeline, error) {
	return func(log *logger.Logger, id routingKey) (pipeline, error) {
		// new operator per pipeline to isolate processes without using tags
		operator, err := newOperator(log, id, cfg, r)
		if err != nil {
			return nil, err
		}

		return &operatorPipeline{
			log:           log,
			configHandler: operator,
		}, nil
	}
}

func newOperator(log *logger.Logger, id routingKey, config *config.Config, r reporter) (*operation.Operator, error) {
	operatorConfig := &operatorCfg.Config{}
	if err := config.Unpack(&operatorConfig); err != nil {
		return nil, err
	}

	fetcher := downloader.NewDownloader(operatorConfig.DownloadConfig)
	installer, err := install.NewInstaller(operatorConfig.DownloadConfig)
	if err != nil {
		return nil, errors.Wrap(err, "initiating installer")
	}

	stateResolver, err := stateresolver.NewStateResolver(log)
	if err != nil {
		return nil, err
	}

	return operation.NewOperator(
		log,
		id,
		config,
		fetcher,
		installer,
		stateResolver,
		r,
	)
}
