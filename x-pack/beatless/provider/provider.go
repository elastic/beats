// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/x-pack/beatless/core"
)

// Create a new pipeline client based on the function configuration.
type clientFactory func(*common.Config) (core.Client, error)

// Function is temporary
type Function interface {
	Run(context.Context, core.Client) error
	Name() string
}

// Provider providers the layer between beatless and cloud specific settings, its is responsable to
// return the function that need to be executed.
type Provider interface {
	CreateFunctions(clientFactory) ([]core.Runner, error)
	Name() string
}

// Runnable is the unit of work managed by the coordinator, anything related to the life of a function
// is encapsulated into the runnable.
type Runnable struct {
	config     *common.Config
	function   Function
	makeClient clientFactory
}

// Run call the the function's Run method, the method is a specific goroutine, it will block until
// beats shutdown or an error happen.
func (r *Runnable) Run(ctx context.Context) error {
	client, err := r.makeClient(r.config)
	if err != nil {
		return errors.Wrap(err, "could not create a client for the function")
	}
	defer client.Close()
	return r.function.Run(ctx, client)
}
