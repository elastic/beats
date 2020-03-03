// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"context"
	"net"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/retry"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/state"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/remoteconfig"
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

	spec := a.spec.Spec()
	if spec.Configurable != ConfigurableGrpc {
		return nil
	}

	if a.state.Status == state.Stopped {
		return errors.New(ErrAppNotRunning)
	}

	retryFn := func() error {
		a.appLock.Lock()
		defer a.appLock.Unlock()

		// TODO: check versions(logical clock) in between retries in case newer version sneaked in

		ctx, cancelFn := context.WithTimeout(ctx, DefaultTimeout)
		defer cancelFn()

		if a.grpcClient == nil {
			return errors.New(ErrClientNotFound)
		}

		rawYaml, err := yaml.Marshal(config)
		if err != nil {
			return errors.New(err, errors.TypeApplication)
		}

		configClient, ok := a.grpcClient.(remoteconfig.ConfiguratorClient)
		if !ok {
			return errors.New(ErrClientNotConfigurable, errors.TypeApplication)
		}

		err = configClient.Config(ctx, string(rawYaml))

		if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || netErr.Temporary()) {
			// not fatal, we will retry
			return errors.New(netErr, errors.TypeApplication)
		}

		// if not transient mark as fatal
		return retry.ErrorMakeFatal(err)
	}

	return retry.Do(a.retryConfig, retryFn)
}
