// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

// Configure configures the application with the passed configuration.
func (a *Application) Configure(ctx context.Context, config map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
			a.statusReporter.Update(state.Degraded, err.Error(), nil)
		}
	}()

	a.appLock.Lock()
	defer a.appLock.Unlock()

	if a.state.Status == state.Stopped {
		return errors.New(ErrAppNotRunning)
	}
	if a.srvState == nil {
		return errors.New(ErrAppNotRunning)
	}

	a.logger.Infof("Application %s, config: %#v", a.Name(), config)
	cfgStr, err := yaml.Marshal(config)
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}
	a.logger.With("config.yaml", string(cfgStr)).Infof("sending config to %s", a.Name())
	a.logger.Infof("%s sending config: %s", a.Name(), string(cfgStr))

	isRestartNeeded := plugin.IsRestartNeeded(a.logger, a.Spec(), a.srvState, config)

	err = a.srvState.UpdateConfig(string(cfgStr))
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}

	if isRestartNeeded {
		a.logger.Infof("initiating restart of '%s' due to config change", a.Name())
		a.appLock.Unlock()
		a.Stop()
		err = a.Start(ctx, a.desc, config)
		// lock back so it won't panic on deferred unlock
		a.appLock.Lock()
	}

	return err
}
