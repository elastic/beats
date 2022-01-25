// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

type once struct {
	log      *logger.Logger
	discover discoverFunc
	loader   *config.Loader
	emitter  pipeline.EmitterFunc
}

func newOnce(log *logger.Logger, discover discoverFunc, loader *config.Loader, emitter pipeline.EmitterFunc) *once {
	return &once{log: log, discover: discover, loader: loader, emitter: emitter}
}

func (o *once) Start() error {
	files, err := o.discover()
	if err != nil {
		return errors.New(err, "could not discover configuration files", errors.TypeConfig)
	}

	if len(files) == 0 {
		return ErrNoConfiguration
	}

	return readfiles(files, o.loader, o.emitter)
}

func (o *once) Stop() error {
	return nil
}

func readfiles(files []string, loader *config.Loader, emitter pipeline.EmitterFunc) error {
	c, err := loader.Load(files)
	if err != nil {
		return errors.New(err, "could not load or merge configuration", errors.TypeConfig)
	}

	return emitter(c)
}
