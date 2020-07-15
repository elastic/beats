// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

// OnStatusChange is the handler called by the GRPC server code.
//
// It updates the status of the application and handles restarting the application if needed.
func (a *Application) OnStatusChange(s *server.ApplicationState, status proto.StateObserved_Status, msg string, payload map[string]interface{}) {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	// If the application is stopped, do not update the state. Stopped is a final state
	// and should not be overridden.
	if a.state.Status == state.Stopped {
		return
	}

	a.setStateFromProto(status, msg, payload)
	if status == proto.StateObserved_FAILED {
		// ignore when expected state is stopping
		if s.Expected() == proto.StateExpected_STOPPING {
			return
		}

		// it was a crash, cleanup anything required
		go a.cleanUp()

		// kill the process
		if a.state.ProcessInfo != nil {
			_ = a.state.ProcessInfo.Process.Kill()
			a.state.ProcessInfo = nil
		}
		ctx := a.startContext
		tag := a.tag

		// it was marshalled to pass into the state, so unmarshall will always succeed
		var cfg map[string]interface{}
		_ = yaml.Unmarshal([]byte(s.Config()), &cfg)

		err := a.start(ctx, tag, cfg)
		if err != nil {
			a.setState(state.Crashed, fmt.Sprintf("failed to restart: %s", err), nil)
		}
	}
}
