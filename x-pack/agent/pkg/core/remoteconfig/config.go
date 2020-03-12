// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remoteconfig

import (
	"context"
)

// Client for remote calls
type Client interface{}

// ConfiguratorClient is the client connecting agent and a process
type ConfiguratorClient interface {
	Config(ctx context.Context, config string) error
	Close() error
}

// ConnectionCreator describes a creator of connections.
// ConnectionCreator should be used in client vault to generate new connections.
type ConnectionCreator interface {
	NewConnection(address ConnectionProvider) (Client, error)
}

// ConnectionProvider is a basic provider everybody needs to implement
// in order to provide a valid connection.
// Minimal set of properties is: address
type ConnectionProvider interface {
	Address() string
}
