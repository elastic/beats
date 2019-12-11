package application

import (
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
	scheduler := scheduler.NewPeriodic(settings)
	return newFleetGatewayWithScheduler(
		log,
		settings,
		agentID,
		client,
		dispatcher,
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
		settings:   settings,
		dispatcher: d,
		client:     client,
		agentID:    agentID, //TODO(ph): this need to be a struct.
		scheduler:  scheduler,
	}, nil
}

func (f *fleetGateway) worker() {
	for {
		select {
		case time.After(f.settings.Period):
			resp, err := f.execute()
			if err != nil {
				// todo log?
			}
			err := f.dispatcher.Dispatch(resp.Actions...)
			// TODO err
		}
	}
}

func (f *fleetGateway) execute() (*fleetapi.CheckinResponse, error) {
	cmd := fleetapi.NewCheckinCmd(f.agentID, f.client)

	// TODO: batch events.
	req := fleetapi.CheckinRequest{}
	resp, err := cmd.Execute(cmd)
	if err != nil {
		return err
	}

	return resp, nil
}

func (f *fleetGateway) Report(event interface{}) error {
	// TODO, make sure we accumulate

}

func (f *fleetGateway) Start() error {
}

func (f *fleetGateway) Stop() error {
	// TODO lets try to flush events before shutting down.
}

// TODO:
// Questions(ph) Block or not on the stop.
// - [ ] refactor the application.go
// - [ ] Use a Scheduler interface to make synchronous testing working.
