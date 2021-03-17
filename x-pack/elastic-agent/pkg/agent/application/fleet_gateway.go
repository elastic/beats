// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/scheduler"
)

const maxUnauthCounter int = 6

// Default Configuration for the Fleet Gateway.
var defaultGatewaySettings = &fleetGatewaySettings{
	Duration: 1 * time.Second,        // time between successful calls
	Jitter:   500 * time.Millisecond, // used as a jitter for duration
	Backoff: backoffSettings{ // time after a failed call
		Init: 60 * time.Second,
		Max:  10 * time.Minute,
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

type fleetAcker = store.FleetAcker

type dispatcher interface {
	Dispatch(acker fleetAcker, actions ...action) error
}

type agentInfo interface {
	AgentID() string
}

type fleetReporter interface {
	Events() ([]fleetapi.SerializableEvent, func())
}

// FleetGateway is a gateway between the Agent and the Fleet API, it's take cares of all the
// bidirectional communication requirements. The gateway aggregates events and will periodically
// call the API to send the events and will receive actions to be executed locally.
// The only supported action for now is a "ActionPolicyChange".
type FleetGateway interface {
	// Start starts the gateway.
	Start() error

	// Set the client for the gateway.
	SetClient(clienter)
}

type stateStore interface {
	Add(fleetapi.Action)
	AckToken() string
	SetAckToken(ackToken string)
	Save() error
	Actions() []fleetapi.Action
}

type fleetGateway struct {
	bgContext        context.Context
	log              *logger.Logger
	dispatcher       dispatcher
	client           clienter
	scheduler        scheduler.Scheduler
	backoff          backoff.Backoff
	settings         *fleetGatewaySettings
	agentInfo        agentInfo
	reporter         fleetReporter
	done             chan struct{}
	wg               sync.WaitGroup
	acker            fleetAcker
	unauthCounter    int
	statusController status.Controller
	statusReporter   status.Reporter
	stateStore       stateStore
}

func newFleetGateway(
	ctx context.Context,
	log *logger.Logger,
	agentInfo agentInfo,
	client clienter,
	d dispatcher,
	r fleetReporter,
	acker fleetAcker,
	statusController status.Controller,
	stateStore stateStore,
) (FleetGateway, error) {

	scheduler := scheduler.NewPeriodicJitter(defaultGatewaySettings.Duration, defaultGatewaySettings.Jitter)
	return newFleetGatewayWithScheduler(
		ctx,
		log,
		defaultGatewaySettings,
		agentInfo,
		client,
		d,
		scheduler,
		r,
		acker,
		statusController,
		stateStore,
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
	statusController status.Controller,
	stateStore stateStore,
) (FleetGateway, error) {

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
		done:             done,
		reporter:         r,
		acker:            acker,
		statusReporter:   statusController.RegisterComponent("gateway"),
		statusController: statusController,
		stateStore:       stateStore,
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
				f.statusReporter.Update(state.Failed, err.Error())
				continue
			}

			actions := make([]action, len(resp.Actions))
			for idx, a := range resp.Actions {
				actions[idx] = a
			}

			var errMsg string
			if err := f.dispatcher.Dispatch(f.acker, actions...); err != nil {
				errMsg = fmt.Sprintf("failed to dispatch actions, error: %s", err)
				f.log.Error(errMsg)
				f.statusReporter.Update(state.Failed, errMsg)
			}

			f.log.Debugf("FleetGateway is sleeping, next update in %s", f.settings.Duration)
			if errMsg != "" {
				f.statusReporter.Update(state.Failed, errMsg)
			} else {
				f.statusReporter.Update(state.Healthy, "")
			}

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

	// retrieve ack token from the store
	ackToken := f.stateStore.AckToken()
	if ackToken != "" {
		f.log.Debug("using previously saved ack token: %v", ackToken)
	}

	// checkin
	cmd := fleetapi.NewCheckinCmd(f.agentInfo, f.client)
	req := &fleetapi.CheckinRequest{
		AckToken: ackToken,
		Events:   ee,
		Metadata: ecsMeta,
		Status:   f.statusController.StatusString(),
	}

	resp, err := cmd.Execute(ctx, req)
	if isUnauth(err) {
		f.unauthCounter++

		if f.shouldUnroll() {
			f.log.Warnf("retrieved unauthorized for '%d' times. Unrolling.", f.unauthCounter)
			return &fleetapi.CheckinResponse{
				Actions: []fleetapi.Action{&fleetapi.ActionUnenroll{ActionID: "", ActionType: "UNENROLL", IsDetected: true}},
			}, nil
		}

		return nil, err
	}

	f.unauthCounter = 0
	if err != nil {
		return nil, err
	}

	// Save the latest ackToken
	if resp.AckToken != "" {
		f.stateStore.SetAckToken(resp.AckToken)
		serr := f.stateStore.Save()
		if serr != nil {
			f.log.Errorf("failed to save the ack token, err: %v", serr)
		}
	}

	// ack events so they are dropped from queue
	ack()
	return resp, nil
}

func (f *fleetGateway) shouldUnroll() bool {
	return f.unauthCounter >= maxUnauthCounter
}

func isUnauth(err error) bool {
	return errors.Is(err, fleetapi.ErrInvalidAPIKey)
}

func (f *fleetGateway) Start() error {
	f.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer f.log.Info("Fleet gateway is stopped")
		defer wg.Done()

		f.worker()
	}(&f.wg)
	return nil
}

func (f *fleetGateway) stop() {
	f.log.Info("Fleet gateway is stopping")
	defer f.scheduler.Stop()
	f.statusReporter.Unregister()
	close(f.done)
	f.wg.Wait()
}

func (f *fleetGateway) SetClient(client clienter) {
	f.client = client
}
