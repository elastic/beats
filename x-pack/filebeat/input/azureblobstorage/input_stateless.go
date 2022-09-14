// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/types"
)

type statelessInput struct {
	config     config
	serviceURL string
}

func (statelessInput) Name() string {
	return "azureblobstorage-stateless"
}

func newStatelessInput(config config) *statelessInput {
	return &statelessInput{config: config}
}

func (in *statelessInput) Test(v2.TestContext) error {
	return nil
}

type statelessPublisher struct {
	wrapped stateless.Publisher
}

func (pub statelessPublisher) Publish(event beat.Event, _ interface{}) error {
	pub.wrapped.Publish(event)
	return nil
}

// Run starts the input and blocks until it ends the execution.
// It will return on context cancellation, any other error will be retried.
func (in *statelessInput) Run(inputCtx v2.Context, publisher stateless.Publisher) error {

	pub := statelessPublisher{wrapped: publisher}
	var source cursor.Source
	for _, c := range in.config.Containers {
		container := tryOverrideOrDefault(in.config, c)
		source = &types.Source{
			AccountName:   in.config.AccountName,
			ContainerName: c.Name,
			MaxWorkers:    *container.MaxWorkers,
			Poll:          *container.Poll,
			PollInterval:  *container.PollInterval,
		}

		st := state.NewState()
		currentSource := source.(*types.Source)
		log := inputCtx.Logger.With("account_name", currentSource.AccountName).With("container", currentSource.ContainerName)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-inputCtx.Cancelation.Done()
			cancel()
		}()

		serviceClient, credential, err := fetchServiceClientAndCreds(in.config, in.serviceURL, log)
		if err != nil {
			return err
		}
		containerClient, err := fetchContainerClient(serviceClient, currentSource.ContainerName, log)
		if err != nil {
			return err
		}

		scheduler := NewAzureInputScheduler(pub, containerClient, credential, currentSource, &in.config, st, in.serviceURL, log)
		err = scheduler.Schedule(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
