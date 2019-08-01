// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logger

import (
	"github.com/urso/ecslog"
	"github.com/urso/ecslog/backend"
	"github.com/urso/ecslog/backend/appender"
	"github.com/urso/ecslog/backend/layout"

	"github.com/elastic/fleet/x-pack/pkg/config"
)

// Logger alias ecslog.Logger with Logger.
type Logger = ecslog.Logger

// New returns a configured ECS Logger
func New() (*Logger, error) {
	backend, err := createJSONBackend()
	if err != nil {
		return nil, err
	}
	return ecslog.New(backend), nil
}

func createJSONBackend() (backend.Backend, error) {
	return appender.Console(backend.Trace, layout.Text(true))
}

//NewFromConfig takes the user configuration and generate the right logger.
// TODO: Finish implementation, need support on the library that we use.
func NewFromConfig(_ *config.Config) (*Logger, error) {
	return New()
}
