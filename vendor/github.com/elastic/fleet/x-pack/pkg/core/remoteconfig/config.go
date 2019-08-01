// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remoteconfig

import (
	"context"

	"github.com/elastic/fleet/x-pack/pkg/core/plugin/clientvault"
)

// ConfiguratorClient is the client connecting agent and a process
type ConfiguratorClient interface {
	Config(ctx context.Context, config string) error
	Close() error
}

// ConnectionCreator describes a creator of connections.
// ConnectionCreator should be used in client vault to generate new connections.
type ConnectionCreator interface {
	NewConnection(address clientvault.ConnectionProvider) (clientvault.Client, error)
}
