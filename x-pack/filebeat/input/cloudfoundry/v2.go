// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package cloudfoundry

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry"
	"github.com/elastic/go-concert/ctxtool"
)

// inputV2 defines a Cloudfoundry input that uses the consumer V2 API
type inputV2 struct {
	config cloudfoundry.Config
}

func configureV2(config cloudfoundry.Config) (*inputV2, error) {
	return &inputV2{config: config}, nil
}

func (i *inputV2) Name() string { return "cloudfoundry-v2" }

func (i *inputV2) Test(ctx v2.TestContext) error {
	hub := cloudfoundry.NewHub(&i.config, "filebeat", ctx.Logger)
	_, err := hub.Client()
	return err
}

func (i *inputV2) Run(ctx v2.Context, publisher stateless.Publisher) error {
	log := ctx.Logger
	hub := cloudfoundry.NewHub(&i.config, "filebeat", log)

	callbacks := cloudfoundry.RlpListenerCallbacks{
		HttpAccess: func(evt *cloudfoundry.EventHttpAccess) {
			publisher.Publish(createEvent(evt))
		},
		Log: func(evt *cloudfoundry.EventLog) {
			publisher.Publish(createEvent(evt))
		},
		Error: func(evt *cloudfoundry.EventError) {
			publisher.Publish(createEvent(evt))
		},
	}

	listener, err := hub.RlpListener(callbacks)
	if err != nil {
		return err
	}

	listener.Start(ctxtool.FromCanceller(ctx.Cancelation))
	listener.Wait()
	return nil
}
