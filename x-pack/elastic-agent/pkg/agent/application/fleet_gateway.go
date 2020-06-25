// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/scheduler"
)

// Default Configuration for the Fleet Gateway.
var defaultGatewaySettings = &fleetGatewaySettings{
	Duration: 1 * time.Second,        // time between successful calls
	Jitter:   500 * time.Millisecond, // used as a jitter for duration
	Backoff: backoffSettings{ // time after a failed call
		Init: 5 * time.Second,
		Max:  60 * time.Second,
	},
}

type fleetGatewaySettings struct {
	Duration time.Duration   `config:"checkin_frequency"`
	Jitter   time.Duration   `config:"jitter"`
	Backoff  backoffSettings `config:"backoff"`
}

type backoffSettings struct {
	Init time.Duration `config:"init"`
	Max  time.Duration `config:"max"`
}

type dispatcher interface {
	Dispatch(acker fleetAcker, actions ...action) error
}

type agentInfo interface {
	AgentID() string
}

type fleetReporter interface {
	Events() ([]fleetapi.SerializableEvent, func())
}

type fleetAcker interface {
	Ack(ctx context.Context, action fleetapi.Action) error
	Commit(ctx context.Context) error
}

// fleetGateway is a gateway between the Agent and the Fleet API, it's take cares of all the
// bidirectional communication requirements. The gateway aggregates events and will periodically
// call the API to send the events and will receive actions to be executed locally.
// The only supported action for now is a "ActionPolicyChange".
type fleetGateway struct {
	bgContext  context.Context
	log        *logger.Logger
	dispatcher dispatcher
	client     clienter
	scheduler  scheduler.Scheduler
	backoff    backoff.Backoff
	settings   *fleetGatewaySettings
	agentInfo  agentInfo
	reporter   fleetReporter
	done       chan struct{}
	wg         sync.WaitGroup
	acker      fleetAcker
}

func newFleetGateway(
	ctx context.Context,
	log *logger.Logger,
	rawConfig *config.Config,
	agentInfo agentInfo,
	client clienter,
	d dispatcher,
	r fleetReporter,
	acker fleetAcker,
) (*fleetGateway, error) {

	settings := defaultGatewaySettings
	if err := rawConfig.Unpack(settings); err != nil {
		return nil, errors.New(err, "fail to read gateway configuration")
	}

	scheduler := scheduler.NewPeriodicJitter(settings.Duration, settings.Jitter)
	return newFleetGatewayWithScheduler(
		ctx,
		log,
		settings,
		agentInfo,
		client,
		d,
		scheduler,
		r,
		acker,
	)
}

func newFleetGatewayWithScheduler(
	ctx context.Context,
	log *logger.Logger,
	settings *fleetGatewaySettings,
	agentInfo agentInfo,
	client clienter,
	d dispatcher,
	scheduler scheduler.Scheduler,
	r fleetReporter,
	acker fleetAcker,
) (*fleetGateway, error) {

	// Backoff implementation doesn't support the using context as the shutdown mechanism.
	// So we keep a done channel that will be closed when the current context is shutdown.
	done := make(chan struct{})

	return &fleetGateway{
		bgContext:  ctx,
		log:        log,
		dispatcher: d,
		client:     client,
		settings:   settings,
		agentInfo:  agentInfo,
		scheduler:  scheduler,
		backoff: backoff.NewEqualJitterBackoff(
			done,
			settings.Backoff.Init,
			settings.Backoff.Max,
		),
		done:     done,
		reporter: r,
		acker:    acker,
	}, nil
}

func (f *fleetGateway) worker() {
	for {
		select {
		case <-f.scheduler.WaitTick():
			f.log.Debug("FleetGateway calling Checkin API")

			// Execute the checkin call and for any errors returned by the fleet API
			// the function will retry to communicate with fleet with an exponential delay and some
			// jitter to help better distribute the load from a fleet of agents.
			resp, err := f.doExecute()
			if err != nil {
				f.log.Error(err)
				continue
			}

			actions := make([]action, len(resp.Actions))
			for idx, a := range resp.Actions {
				actions[idx] = a
			}

			if err := f.dispatcher.Dispatch(f.acker, actions...); err != nil {
				f.log.Errorf("failed to dispatch actions, error: %s", err)
			}

			f.log.Debugf("FleetGateway is sleeping, next update in %s", f.settings.Duration)
		case <-f.bgContext.Done():
			f.stop()
			return
		}
	}
}

func (f *fleetGateway) doExecute() (*fleetapi.CheckinResponse, error) {
	f.backoff.Reset()
	for f.bgContext.Err() == nil {
		// TODO: wrap with timeout context
		resp, err := f.execute(f.bgContext)
		if err != nil {
			f.log.Errorf("Could not communicate with Checking API will retry, error: %s", err)
			if !f.backoff.Wait() {
				return nil, errors.New(
					"execute retry loop was stopped",
					errors.TypeNetwork,
					errors.M(errors.MetaKeyURI, f.client.URI()),
				)
			}
			continue
		}
		return resp, nil
	}

	return nil, f.bgContext.Err()
}

func (f *fleetGateway) execute(ctx context.Context) (*fleetapi.CheckinResponse, error) {
	// get events
	ee, ack := f.reporter.Events()

	ecsMeta, err := metadata()
	if err != nil {
		f.log.Error(errors.New("failed to load metadata", err))
	}

	// checkin
	cmd := fleetapi.NewCheckinCmd(f.agentInfo, f.client)
	req := &fleetapi.CheckinRequest{
		Events:   ee,
		Metadata: ecsMeta,
	}

	resp, err := cmd.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	// ack events so they are dropped from queue
	ack()
	return resp, nil
}

func (f *fleetGateway) Start() {
	f.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer f.log.Info("Fleet gateway is stopped")
		defer wg.Done()

		f.worker()
	}(&f.wg)
}

func (f *fleetGateway) stop() {
	f.log.Info("Fleet gateway is stopping")
	defer f.scheduler.Stop()
	close(f.done)
	f.wg.Wait()
}
