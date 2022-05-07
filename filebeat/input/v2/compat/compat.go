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
	"fmt"
	"sync"

	"github.com/mitchellh/hashstructure"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/ctxtool"
)

// factory implements the cfgfile.RunnerFactory interface and wraps the
// v2.Loader to create cfgfile.Runner instances based on available v2 inputs.
type factory struct {
	log    *logp.Logger
	info   beat.Info
	loader *v2.Loader
}

// runner wraps a v2.Input, starting a go-routine
// On start the runner spawns a go-routine that will call (v2.Input).Run with
// the `sig` setup for shutdown signaling.
// On stop the runner triggers the shutdown signal and waits until the input
// has returned.
type runner struct {
	id        string
	log       *logp.Logger
	agent     *beat.Info
	wg        sync.WaitGroup
	sig       ctxtool.CancelContext
	input     v2.Input
	connector beat.PipelineConnector
}

// RunnerFactory creates a cfgfile.RunnerFactory from an input Loader that is
// compatible with config file based input reloading, autodiscovery, and filebeat modules.
// The RunnerFactory is can be used to integrate v2 inputs into existing Beats.
func RunnerFactory(
	log *logp.Logger,
	info beat.Info,
	loader *v2.Loader,
) cfgfile.RunnerFactory {
	return &factory{log: log, info: info, loader: loader}
}

func (f *factory) CheckConfig(cfg *conf.C) error {
	_, err := f.loader.Configure(cfg)
	if err != nil {
		return err
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
		id:        id,
		log:       f.log.Named(input.Name()).With("id", id),
		agent:     &f.info,
		sig:       ctxtool.WithCancelContext(context.Background()),
		input:     input,
		connector: p,
	}, nil
}

func (r *runner) String() string { return r.input.Name() }

func (r *runner) Start() {
	log := r.log
	name := r.input.Name()

	go func() {
		log.Infof("Input %v starting", name)
		err := r.input.Run(
			v2.Context{
				ID:          r.id,
				Agent:       *r.agent,
				Logger:      log,
				Cancelation: r.sig,
			},
			r.connector,
		)
		if err != nil {
			log.Errorf("Input '%v' failed with: %+v", name, err)
		} else {
			log.Infof("Input '%v' stopped", name)
		}
	}()
}

func (r *runner) Stop() {
	r.sig.Cancel()
	r.wg.Wait()
	r.log.Infof("Input '%v' stopped", r.input.Name())
}

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
	config.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute id from configuration: %w", err)
	}

	return fmt.Sprintf("%16X", id), nil
}
