// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
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

	a.setState(state.FromProto(status), msg, payload)
	if status == proto.StateObserved_FAILED {
		// ignore when expected state is stopping
		if s.Expected() == proto.StateExpected_STOPPING {
			return
		}

		// it was marshalled to pass into the state, so unmarshall will always succeed
		var cfg map[string]interface{}
		_ = yaml.Unmarshal([]byte(s.Config()), &cfg)

		// start the failed timer
		// pass process info to avoid killing new process spun up in a meantime
		a.startFailedTimer(cfg, a.state.ProcessInfo)
	} else {
		a.stopFailedTimer()
	}
}

// startFailedTimer starts a timer that will restart the application if it doesn't exit failed after a period of time.
//
// This does not grab the appLock, that must be managed by the caller.
func (a *Application) startFailedTimer(cfg map[string]interface{}, proc *process.Info) {
	if a.restartCanceller != nil {
		// already have running failed timer; just update config
		a.restartConfig = cfg
		return
	}

	ctx, cancel := context.WithCancel(a.startContext)
	a.restartCanceller = cancel
	a.restartConfig = cfg
	t := time.NewTimer(a.processConfig.FailureTimeout)
	go func() {
		defer func() {
			a.appLock.Lock()
			a.restartCanceller = nil
			a.restartConfig = nil
			a.appLock.Unlock()
		}()

		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.restart(proc)
		}
	}()
}

// stopFailedTimer stops the timer that would restart the application from reporting failure.
//
// This does not grab the appLock, that must be managed by the caller.
func (a *Application) stopFailedTimer() {
	if a.restartCanceller == nil {
		return
	}
	a.restartCanceller()
	a.restartCanceller = nil
}

// restart restarts the application
func (a *Application) restart(proc *process.Info) {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	// stop the watcher
	a.stopWatcher(proc)

	// kill the process
	if proc != nil && proc.Process != nil {
		_ = proc.Process.Kill()
	}

	if proc != a.state.ProcessInfo {
		// we're restarting different process than actually running
		// no need to start another one
		return
	}

	a.state.ProcessInfo = nil

	ctx := a.startContext
	tag := a.tag

	a.setState(state.Restarting, "", nil)
	err := a.start(ctx, tag, a.restartConfig, true)
	if err != nil {
		a.setState(state.Crashed, fmt.Sprintf("failed to restart: %s", err), nil)
	}
}
