package application

import (
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

type dispatcher interface {
	Dispatch(...action) error
}

type fleetGateway struct {
	log        *logger.Logger
	dispatcher dispatcher
	client     clienter
}

type fleetGatewaySettings struct {
	Period time.Duration
}

func newFleetGateway(
	log *logger.Logger,
	settings *fleetGatewaySettings,
	agentID string,
	client clienter,
	d dispatcher,
) (*fleetGateway, error) {
	return &fleetGateway{
		log:        log,
		settings:   settings,
		dispatcher: d,
		client:     client,
		agentID:    agentID,
	}
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

func (f *fleetGateway) Start() error {
}

func (f *fleetGateway) Stop() error {
}

// TODO:
// Questions(ph) Block or not on the stop.
// - refactor the application.go
