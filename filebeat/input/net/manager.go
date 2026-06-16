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
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/unison"
)

// Input is the interface for net inputs
// go:generate moq -out inputmock_test.go . Input
type Input interface {
	// Returns the input name
	Name() string

	// Tests the input, if possible it should ensure the input can
	// start a server at the configured address and port
	Test(v2.TestContext) error

	// InitMetrics initialises the metrics for the input.
	// The 'id' argument is the input ID to be used in the metrics
	InitMetrics(id string, reg *monitoring.Registry, logger *logp.Logger) Metrics

	// Runs the input. Events sent to the channel will be
	// published by any of the pipeline workers.
	// Run must call EventReceived on Metrics once the
	// event is received passing the event size and the
	// current time.
	Run(v2.Context, chan<- DataMetadata, Metrics) error
}

// Metrics is an interface to abstract the metrics
// from input/netmetrics
type Metrics interface {
	// EventPublished updates all metrics related to published events.
	EventPublished(start time.Time)
	// EventReceived update all metrics related to receiving events.
	EventReceived(len int, timestamp time.Time)
}

type manager struct {
	configure func(*conf.C) (Input, error)
}

type config struct {
	NumWorkers int    `config:"number_of_workers" validate:"positive,nonzero"`
	Host       string `config:"host"`
}

// DataMetadata contains the data read from the network connection
// and its metadata
type DataMetadata struct {
	Timestamp time.Time
	Data      []byte
	Metadata  inputsource.NetworkMetadata
}

type wrapper struct {
	inp                Input
	numPipelineWorkers int
	evtChan            chan DataMetadata
	host               string // used for the logger
}

// NewManager creates a v2.InputManager for net inputs.
// The returned manager and the input wrapper it uses are responsible for:
//   - Handling the pipeline workers (including parsing 'number_of_workers')
//   - Adding the 'host' field to the logger
//   - Updating the input status (Configuring, Starting, Failed).
//     The input must update the status to 'Running'
//   - Handling context cancellation errors
//   - Recovering from panic
func NewManager(fn func(*conf.C) (Input, error)) v2.InputManager {
	return &manager{configure: fn}
}

// Init Noop, it is required to fulfil the v2.InputManager interface.
func (*manager) Init(grp unison.Group) error { return nil }

// Create creates a new Input instance from the given configuration
// by calling the manager's configure callback, or returns
// an error if the configuration is invalid.
func (m *manager) Create(cfg *conf.C) (v2.Input, error) {
	wrapperCfg := config{NumWorkers: 1} // Default config
	if err := cfg.Unpack(&wrapperCfg); err != nil {
		return nil, err
	}

	inp, err := m.configure(cfg)
	if err != nil {
		return nil, err
	}

	w := wrapper{
		inp:                inp,
		numPipelineWorkers: wrapperCfg.NumWorkers,
		host:               wrapperCfg.Host,
		// 5 is a magic number, we just need to ensure there is some buffer
		// in the channel to reduce contention
		evtChan: make(chan DataMetadata, wrapperCfg.NumWorkers*runtime.NumCPU()+1),
	}

	return w, nil
}

// Name proxies the call to the input
func (w wrapper) Name() string { return w.inp.Name() }

// Test proxies the call to the input instance
func (w wrapper) Test(ctx v2.TestContext) error { return w.inp.Test(ctx) }

// Run initialise the metrics, starts the worker, updates the status
// to 'Configuring', then 'Starting', finally it calls the input's Run method.
// Run recovers from panic and logs the reason for the panic.
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

	m := w.inp.InitMetrics(ctx.ID, ctx.MetricsRegistry, ctx.Logger)
	if err := w.initWorkers(ctx, pipeline, m); err != nil {
		logger.Errorf("cannot initialise pipeline workers: %s", err)
		return fmt.Errorf("cannot initialise pipeline workers: %w", err)
	}

	err = w.inp.Run(ctx, w.evtChan, m)
	if errors.Is(err, context.Canceled) {
		ctx.UpdateStatus(status.Stopped, "")
		return nil
	}

	if err != nil {
		ctx.UpdateStatus(status.Failed, "Input exited unexpectedly: "+err.Error())
		return err
	}

	ctx.UpdateStatus(status.Stopped, "")
	return nil
}

func (w wrapper) initWorkers(ctx v2.Context, pipeline beat.Pipeline, metrics Metrics) error {
	for id := range w.numPipelineWorkers {
		client, err := pipeline.ConnectWith(beat.ClientConfig{
			PublishMode: beat.DefaultGuarantees,
		})
		if err != nil {
			return fmt.Errorf("[worker %d] cannot connect to publishing pipeline: %w", id, err)
		}

		go w.publishLoop(ctx, id, client, metrics)
	}

	return nil
}

// publishLoop reads events from w.evtChan and publishes them to the client.
// If ctx is cancelled publishLoop returns. The client is always closed by
// publishLoop.
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
		case d := <-w.evtChan:
			start := time.Now()
			evt := beat.Event{
				Timestamp: d.Timestamp,
				Fields: mapstr.M{
					"message": string(d.Data),
				},
			}
			if d.Metadata.RemoteAddr != nil {
				evt.Fields["log"] = mapstr.M{
					"source": mapstr.M{
						"address": d.Metadata.RemoteAddr.String(),
					},
				}
			}

			client.Publish(evt)
			metrics.EventPublished(start)
		}
	}
}
