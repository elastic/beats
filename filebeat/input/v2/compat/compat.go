// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package compat provides helpers for integrating the input/v2 API with
// existing input based features like autodiscovery, config file reloading, or
// filebeat modules.
package compat

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gofrs/uuid/v5"
	"github.com/gohugoio/hashstructure"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/ctxtool"
)

// factory implements the cfgfile.RunnerFactory interface and wraps the
// v2.Loader to create cfgfile.Runner instances based on available v2 inputs.
type factory struct {
	log    *logp.Logger
	info   beat.Info
	loader *v2.Loader

	rootInputsRegistry *monitoring.Registry
}

// runner wraps a v2.Input, starting a go-routine
// On start the runner spawns a go-routine that will call (v2.Input).Run with
// the `sig` setup for shutdown signaling.
// On stop the runner triggers the shutdown signal and waits until the input
// has returned.
type runner struct {
	id                 string
	log                *logp.Logger
	agent              *beat.Info
	wg                 sync.WaitGroup
	sig                ctxtool.CancelContext
	input              v2.Input
	connector          beat.PipelineConnector
	statusReporter     status.StatusReporter
	rootInputsRegistry *monitoring.Registry
}

// RunnerFactory creates a cfgfile.RunnerFactory from an input Loader that is
// compatible with config file based input reloading, autodiscovery, and filebeat modules.
// The RunnerFactory is can be used to integrate v2 inputs into existing Beats.
func RunnerFactory(
	log *logp.Logger,
	info beat.Info,
	rootInputsRegistry *monitoring.Registry,
	loader *v2.Loader,
) cfgfile.RunnerFactory {
	return &factory{log: log, info: info, rootInputsRegistry: rootInputsRegistry, loader: loader}
}

func (f *factory) CheckConfig(cfg *conf.C) error {
	// just check the config, therefore to avoid potential side effects (ID duplication)
	// change the ID.
	checkCfg, err := f.generateCheckConfig(cfg)
	if err != nil {
		f.log.Warnw(fmt.Sprintf("input V2 factory.CheckConfig failed to clone config before checking it. Original config will be checked, it might trigger an input duplication warning: %v", err), "original_config", conf.DebugString(cfg, true))
		checkCfg = cfg
	}
	_, err = f.loader.Configure(checkCfg)
	if err != nil {
		return fmt.Errorf("runner factory could not check config: %w", err)
	}

	if err = f.loader.Delete(checkCfg); err != nil {
		return fmt.Errorf(
			"runner factory failed to delete an input after config check: %w",
			err)
	}

	return nil
}

func (f *factory) Create(
	p beat.PipelineConnector,
	config *conf.C,
) (cfgfile.Runner, error) {
	input, err := f.loader.Configure(config)
	if err != nil {
		return nil, err
	}

	id, err := configID(config)
	if err != nil {
		return nil, err
	}

	return &runner{
		id:                 id,
		log:                f.log.Named(input.Name()).With("id", id),
		agent:              &f.info,
		sig:                ctxtool.WithCancelContext(context.Background()),
		input:              input,
		connector:          p,
		rootInputsRegistry: f.rootInputsRegistry,
	}, nil
}

func (r *runner) SetStatusReporter(reported status.StatusReporter) {
	r.statusReporter = reported
}

func (r *runner) String() string { return r.input.Name() }

func (r *runner) Start() {
	r.wg.Add(1)
	log := r.log
	name := r.input.Name()

	go func() {
		defer r.wg.Done()
		log.Infof("Input '%s' starting", name)

		reg, pc, cancelMetrics := v2.PrepareInputMetrics(
			r.id,
			r.input.Name(),
			r.rootInputsRegistry,
			r.connector,
			r.log)
		defer cancelMetrics()

		ctx := v2.Context{
			ID:              r.id,
			IDWithoutName:   r.id,
			Name:            r.input.Name(),
			Agent:           *r.agent,
			Cancelation:     r.sig,
			MetricsRegistry: reg,
			Logger:          log,
		}
		ctx = ctx.WithStatusReporter(r.statusReporter)

		err := r.input.Run(ctx, pc)
		if err != nil && !errors.Is(err, context.Canceled) {
			errMsg := fmt.Sprintf("Input '%s' failed with: %+v", name, err)
			log.Error(errMsg)
			ctx.UpdateStatus(status.Failed, errMsg)
		} else {
			log.Infof("Input '%s' stopped (goroutine)", name)
		}
	}()
}

func (r *runner) Stop() {
	r.sig.Cancel()
	r.wg.Wait()
	r.log.Infof("Input '%s' stopped (runner)", r.input.Name())
	r.statusReporter = nil
}

// configID extracts or generates an ID for a configuration.
// If the "id" is present in config and is non-empty, it is returned.
// If the "id" is absent or empty, the function calculates a hash of the
// entire configuration and returns it as a hexadecimal string as the ID.
func configID(config *conf.C) (string, error) {
	tmp := struct {
		ID string `config:"id"`
	}{}
	if err := config.Unpack(&tmp); err != nil {
		return "", fmt.Errorf("error extracting ID: %w", err)
	}
	if tmp.ID != "" {
		return tmp.ID, nil
	}

	var h map[string]interface{}
	err := config.Unpack(&h)
	if err != nil {
		return "", fmt.Errorf("could not unpack config into %T: unpack failed: %w",
			h, err)
	}

	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute id from configuration: %w", err)
	}

	return fmt.Sprintf("%16X", id), nil
}

func (f *factory) generateCheckConfig(config *conf.C) (*conf.C, error) {
	// copy the config so it's safe to change it
	testCfg, err := conf.NewConfigFrom(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create new config: %w", err)
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("failed to generate check config id: %w", err)
	}

	finalID := uid.String()
	// if 'id' is present, use it as a prefix
	if testCfg.HasField("id") {
		inputID, err := testCfg.String("id", -1)
		if err != nil {
			return nil, fmt.Errorf("failed to get 'id': %w", err)
		}

		finalID = inputID + "-" + finalID
	}

	if err := testCfg.SetString("id", -1, finalID); err != nil {
		return nil, fmt.Errorf("failed to set 'id': %w", err)
	}

	return testCfg, nil
}
