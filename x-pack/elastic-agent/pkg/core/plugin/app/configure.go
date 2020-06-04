// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"context"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
)

const (
	// DefaultTimeout is the default timeout for network calls
	DefaultTimeout = 60 * time.Second
)

type backoffClient interface {
	Backoff() backoff.Backoff
}

// Configure configures the application with the passed configuration.
func (a *Application) Configure(ctx context.Context, config map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
		}
	}()

	if a.state.Status() == state.Stopped {
		return errors.New(ErrAppNotRunning)
	}

	cfgStr, err := yaml.Marshal(config)
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}
	err = a.state.UpdateConfig(string(cfgStr))
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}
	return nil
}
