// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
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
	Check(ctx context.Context, application Application) (bool, error)
	// Run runs the operation
	Run(ctx context.Context, application Application) error
}

// Application is an application capable of being started, stopped and configured.
type Application interface {
	Name() string
	Started() bool
	Start(ctx context.Context, p app.Taggable, cfg map[string]interface{}) error
	Stop()
	Shutdown()
	Configure(ctx context.Context, config map[string]interface{}) error
	Monitor() monitoring.Monitor
	State() state.State
	SetState(status state.Status, msg string, payload map[string]interface{})
	OnStatusChange(s *server.ApplicationState, status proto.StateObserved_Status, msg string, payload map[string]interface{})
}

// Descriptor defines a program which needs to be run.
// Is passed around operator operations.
type Descriptor interface {
	ServicePort() int
	BinaryName() string
	ArtifactName() string
	Version() string
	ID() string
	Directory() string
	Tags() map[app.Tag]string
}

// ApplicationStatusHandler expects that only Application is registered in the server and updates the
// current state of the application from the OnStatusChange callback from inside the server.
//
// In the case that an application is reported as failed by the server it will then restart the application, unless
// it expects that the application should be stopping.
type ApplicationStatusHandler struct{}

// OnStatusChange is the handler called by the GRPC server code.
//
// It updates the status of the application and handles restarting the application is needed.
func (*ApplicationStatusHandler) OnStatusChange(s *server.ApplicationState, status proto.StateObserved_Status, msg string, payload map[string]interface{}) {
	app, ok := s.App().(Application)
	if !ok {
		panic(errors.New("only Application can be registered when using the ApplicationStatusHandler", errors.TypeUnexpected))
	}
	app.OnStatusChange(s, status, msg, payload)
}
