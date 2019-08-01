// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"net"
	"time"

	"github.com/elastic/fleet/pkg/core/backoff"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/clientvault"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/retry"
	"github.com/elastic/fleet/x-pack/pkg/core/remoteconfig"

	"github.com/pkg/errors"
	"github.com/urso/ecslog"
	"gopkg.in/yaml.v2"
)

var (
	// ErrClientNotFound is an error when client is not found
	ErrClientNotFound = errors.New("client not found, check if process is running")
	// ErrClientNotConfigurable happens when stored client does not implement Config func
	ErrClientNotConfigurable = errors.New("client does not provide configuration")
)

const (
	// DefaultTimeout is the default timeout for network calls
	DefaultTimeout = 60 * time.Second
)

type backoffClient interface {
	Backoff() backoff.Backoff
}

// Configures running process by sending a configuration to its
// grpc endpoint
type operationConfig struct {
	logger         *ecslog.Logger
	program        Program
	operatorConfig *Config

	cv *clientvault.ClientVault
}

func newOperationConfig(logger *ecslog.Logger, p Program, operatorConfig *Config, cv *clientvault.ClientVault) *operationConfig {
	return &operationConfig{
		logger:         logger,
		program:        p,
		operatorConfig: operatorConfig,
		cv:             cv,
	}
}

// Name is human readable name identifying an operation
func (o *operationConfig) Name() string {
	return "operation-config"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationConfig) Check() (bool, error) {
	spec, err := o.program.Spec(o.operatorConfig.DownloadConfig)
	if err != nil {
		o.logger.Errorf("failed to load program.spec for %s.%s: %v", o.program.BinaryName(), o.program.Version(), err)
		return false, nil
	}

	isConfigurable := isGrpcConfigurable(spec.Configurable)
	if !isConfigurable {
		o.logger.Infof("'%s.%s' is not configurable: %s", o.program.BinaryName(), o.program.Version(), spec.Configurable)
	}

	return isConfigurable, nil
}

// Run runs the operation
func (o *operationConfig) Run() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, o.Name())
		}
	}()

	// TODO: use libbeat MapStr
	rawYaml, err := yaml.Marshal(o.program.Config())
	if err != nil {
		return err
	}


	// if we do not have client process is not running
	c, err := o.cv.GetClient(o.program.ID())
	if err != nil {
		return errors.Wrap(err, ErrClientNotFound.Error())
	}


	configClient, ok := c.(remoteconfig.ConfiguratorClient)
	if !ok {
		return ErrClientNotConfigurable
	}


	retryFn := func() error {
		ctx, cancelFn := context.WithTimeout(context.Background(), DefaultTimeout)
		defer cancelFn()

		err := configClient.Config(ctx, string(rawYaml))
		if err != nil {

			return err
		}


		if netErr, ok := err.(net.Error); ok && (netErr.Timeout() || netErr.Temporary()) {
			// not fatal, we will retry

			return netErr
		}


		// if not transient mark as fatal
		return retry.ErrorMakeFatal(err)
	}

	// retry config in case process is not warmed up
	if backoffClient, ok := configClient.(backoffClient); ok {
		return retry.DoWithBackoff(o.operatorConfig.RetryConfig, backoffClient.Backoff(), retryFn)
	}

	return retry.Do(o.operatorConfig.RetryConfig, retryFn)
}
