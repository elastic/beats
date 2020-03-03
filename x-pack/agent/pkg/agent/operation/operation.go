// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/state"
)

// operation is an operation definition
// each operation needs to implement this interface in order
// to ease up rollbacks
type operation interface {
	// Name is human readable name which identifies an operation
	Name() string
	// Check  checks whether operation needs to be run
	// In case prerequisites (such as invalid cert or tweaked binary) are not met, it returns error
	// examples:
	// - Start does not need to run if process is running
	// - Fetch does not need to run if package is already present
	Check() (bool, error)
	// Run runs the operation
	Run(ctx context.Context, application Application) error
}

// Application is an application capable of being started, stopped and configured.
type Application interface {
	Name() string
	Start(ctx context.Context, cfg map[string]interface{}) error
	Stop()
	Configure(ctx context.Context, config map[string]interface{}) error
	State() state.State
	Monitor() monitoring.Monitor
}

// Descriptor defines a program which needs to be run.
// Is passed around operator operations.
type Descriptor interface {
	BinaryName() string
	Version() string
	ID() string
	Directory() string
	IsGrpcConfigurable() bool
}
