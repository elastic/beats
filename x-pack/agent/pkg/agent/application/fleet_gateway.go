// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	"github.com/elastic/beats/x-pack/agent/pkg/scheduler"
)

type dispatcher interface {
	Dispatch(...action) error
}

type fleetGateway struct {
	log        *logger.Logger
	dispatcher dispatcher
	client     clienter
	scheduler  scheduler.Scheduler
	agentID    string
	done       chan struct{}
}

type fleetGatewaySettings struct {
	Duration time.Duration
}

func newFleetGateway(
	log *logger.Logger,
	settings *fleetGatewaySettings,
	agentID string,
	client clienter,
	d dispatcher,
) (*fleetGateway, error) {
	scheduler := scheduler.NewPeriodic(settings.Duration)
	return newFleetGatewayWithScheduler(
		log,
		settings,
		agentID,
		client,
		d,
		scheduler,
	)
}

func newFleetGatewayWithScheduler(
	log *logger.Logger,
	settings *fleetGatewaySettings,
	agentID string,
	client clienter,
	d dispatcher,
	scheduler scheduler.Scheduler,
) (*fleetGateway, error) {
	return &fleetGateway{
		log:        log,
		dispatcher: d,
		client:     client,
		agentID:    agentID, //TODO(ph): this need to be a struct.
		scheduler:  scheduler,
		done:       make(chan struct{}),
	}, nil
}

func (f *fleetGateway) worker() {
	for {
		select {
		case <-f.scheduler.WaitTick():
			resp, err := f.execute()
			if err != nil {
				fmt.Println(err)
				// record
				continue
			}

			actions := make([]action, len(resp.Actions))
			for idx, a := range actions {
				actions[idx] = a
			}

			if err := f.dispatcher.Dispatch(actions...); err != nil {
				// record
			}
		case <-f.done:
			return
		}
	}
}

func (f *fleetGateway) execute() (*fleetapi.CheckinResponse, error) {
	cmd := fleetapi.NewCheckinCmd(f.agentID, f.client)

	req := &fleetapi.CheckinRequest{}
	resp, err := cmd.Execute(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (f *fleetGateway) Start() {
	go f.worker()
}

func (f *fleetGateway) Stop() {
	close(f.done)
}
