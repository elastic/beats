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

package net

import (
	"fmt"
	"runtime/debug"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

type manager struct {
	inputType string
	configure func(*conf.C) (Input, error)
}

type config struct {
	NumPipelineWorkers int    `config:"number_of_workers" validate:"positive,nonzero"`
	Host               string `config:"host"`
}

// New creates a v2.InputManager for net inputs
// TODO: improve it
func New(fn func(*conf.C) (Input, error)) v2.InputManager {
	return &manager{configure: fn}
}

// Init is required to fulfil the input.InputManager interface. Noop
func (*manager) Init(grp unison.Group) error { return nil }

// Create builds a new Input instance from the given configuration, or returns
// an error if the configuration is invalid.
func (m *manager) Create(cfg *conf.C) (v2.Input, error) {
	wrapperCfg := config{NumPipelineWorkers: 1}
	if err := cfg.Unpack(&wrapperCfg); err != nil {
		return nil, err
	}

	inp, err := m.configure(cfg)
	if err != nil {
		return nil, err
	}

	w := wrapper{
		inp:                inp,
		NumPipelineWorkers: wrapperCfg.NumPipelineWorkers,
		host:               wrapperCfg.Host,
		evtChan:            make(chan beat.Event),
	}

	return w, nil
}

type Input interface {
	Name() string
	Test(v2.TestContext) error
	InitMetrics(string, *logp.Logger) Metrics
	Run(v2.Context, chan<- beat.Event, Metrics) error
}

type Metrics interface {
	EventPublished(start time.Time)
	EventReceived(len int, timestamp time.Time)
}

type wrapper struct {
	inp                Input
	NumPipelineWorkers int
	evtChan            chan beat.Event
	host               string // used for metrics
}

// Name reports the input name.
func (w wrapper) Name() string { return w.inp.Name() }

// Test checks the configuration and runs additional checks if the Input can
// actually collect data for the given configuration (e.g. check if host/port or files are
// accessible).
func (w wrapper) Test(ctx v2.TestContext) error { return w.inp.Test(ctx) }

// Run starts the data collection. Run must return an error only if the
// error is fatal making it impossible for the input to recover.
func (w wrapper) Run(ctx v2.Context, pipeline beat.PipelineConnector) (err error) {
	logger := ctx.Logger.With("host", w.host)
	ctx.Logger = logger

	defer func() {
		if v := recover(); v != nil {
			if e, ok := v.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%s input panic with: %+v\n%s", w.inp.Name(), v, debug.Stack())
			}
			logger.Errorw("%s input panic", w.inp.Name(), err)
		}
	}()

	logger.Infof("starting %s input", w.inp.Name())
	defer logger.Infof("%s input stopped", w.inp.Name())

	ctx.UpdateStatus(status.Starting, "")
	ctx.UpdateStatus(status.Configuring, "")

	m := w.inp.InitMetrics(ctx.ID, ctx.Logger)
	if err := w.initWorkers(ctx, pipeline, m); err != nil {
		logger.Errorf("cannot initialise pipeline workers: %s", err)
		return fmt.Errorf("cannot initialise pipeline workers: %w", err)
	}

	return w.inp.Run(ctx, w.evtChan, m)
}

func (w wrapper) initWorkers(ctx v2.Context, pipeline beat.Pipeline, metrics Metrics) error {
	clients := []beat.Client{}
	for id := range w.NumPipelineWorkers {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			PublishMode: beat.DefaultGuarantees,
		})
		if err != nil {
			return fmt.Errorf("[worker %0d] cannot connect to publishing pipeline: %w", id, err)
		}

		clients = append(clients, client)
		go w.publishLoop(ctx, id, client, metrics)
	}

	return nil
}

// publishLoop reads events from w.evtChan and publishes them to the client.
// If ctx is cancelled publishLoop closes the client and returns
func (w wrapper) publishLoop(ctx v2.Context, id int, client beat.Client, metrics Metrics) {
	logger := ctx.Logger
	logger.Debugf("[Worker %d] starting publish loop", id)
	defer logger.Debugf("[Worker %d] finished publish loop", id)
	defer func() {
		if err := client.Close(); err != nil {
			logger.Errorf("[Worker %d] cannot close pipeline client: %s", id, err)
		}
	}()

	for {
		select {
		case <-ctx.Cancelation.Done():
			logger.Debugf("[Worker %d] Context cancelled, closing publish Loop", id)
			return
		case evt := <-w.evtChan:
			start := time.Now()
			client.Publish(evt)
			metrics.EventPublished(start)
		}
	}
}
