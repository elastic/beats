// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
)

// ApplicationStatusHandler expects that only Application is registered in the server and updates the
// current state of the application from the OnStatusChange callback from inside the server.
//
// In the case that an application is reported as failed by the server it will then restart the application, unless
// it expects that the application should be stopping.
type ApplicationStatusHandler struct{}

// OnStatusChange is the handler called by the GRPC server code.
//
// It updates the status of the application and handles restarting the application is needed.
func (*ApplicationStatusHandler) OnStatusChange(s *server.ApplicationState, status proto.StateObserved_Status, msg string) {
	app, ok := s.App().(*Application)
	if !ok {
		panic(errors.New("only *Application can be registered when using the ApplicationStatusHandler", errors.TypeUnexpected))
	}

	app.appLock.Lock()

	// If the application is stopped, do not update the state. Stopped is a final state
	// and should not be overridden.
	if app.state.Status == state.Stopped {
		app.appLock.Unlock()
		return
	}

	app.setStateFromProto(status, msg)
	if status == proto.StateObserved_FAILED {
		// ignore when expected state is stopping
		if s.Expected() == proto.StateExpected_STOPPING {
			app.appLock.Unlock()
			return
		}

		// it was a crash, cleanup anything required
		go app.cleanUp()

		// kill the process
		if app.state.ProcessInfo != nil {
			_ = app.state.ProcessInfo.Process.Kill()
			app.state.ProcessInfo = nil
		}
		ctx := app.startContext
		tag := app.tag

		// it was marshalled to pass into the state, so unmarshall will always succeed
		var cfg map[string]interface{}
		_ = yaml.Unmarshal([]byte(s.Config()), &cfg)

		err := app.start(ctx, tag, cfg)
		if err != nil {
			app.setState(state.Crashed, fmt.Sprintf("failed to restart: %s", err))
		}
	}
	app.appLock.Unlock()
}
