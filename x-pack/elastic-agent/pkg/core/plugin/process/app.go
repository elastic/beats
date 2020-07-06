// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/tokenbucket"
)

var (
	// ErrAppNotRunning is returned when configuration is performed on not running application.
	ErrAppNotRunning = errors.New("application is not running", errors.TypeApplication)
)

// Application encapsulates a concrete application ran by elastic-agent e.g Beat.
type Application struct {
	bgContext    context.Context
	id           string
	name         string
	pipelineID   string
	logLevel     string
	spec         app.Specifier
	srv          *server.Server
	srvState     *server.ApplicationState
	limiter      *tokenbucket.Bucket
	startContext context.Context
	tag          app.Taggable
	state        state.State
	reporter     state.Reporter

	uid int
	gid int

	monitor monitoring.Monitor

	processConfig  *process.Config
	downloadConfig *artifact.Config
	retryConfig    *retry.Config

	logger *logger.Logger

	appLock sync.Mutex
}

// ArgsDecorator decorates arguments before calling an application
type ArgsDecorator func([]string) []string

// NewApplication creates a new instance of an applications. It will not automatically start
// the application.
func NewApplication(
	ctx context.Context,
	id, appName, pipelineID, logLevel string,
	spec app.Specifier,
	srv *server.Server,
	cfg *config.Config,
	logger *logger.Logger,
	reporter state.Reporter,
	monitor monitoring.Monitor) (*Application, error) {

	s := spec.Spec()
	uid, gid, err := s.UserGroup()
	if err != nil {
		return nil, err
	}

	b, _ := tokenbucket.NewTokenBucket(ctx, 3, 3, 1*time.Second)
	return &Application{
		bgContext:      ctx,
		id:             id,
		name:           appName,
		pipelineID:     pipelineID,
		logLevel:       logLevel,
		spec:           spec,
		srv:            srv,
		processConfig:  cfg.ProcessConfig,
		downloadConfig: cfg.DownloadConfig,
		retryConfig:    cfg.RetryConfig,
		logger:         logger,
		limiter:        b,
		reporter:       reporter,
		monitor:        monitor,
		uid:            uid,
		gid:            gid,
	}, nil
}

// Monitor returns monitoring handler of this app.
func (a *Application) Monitor() monitoring.Monitor {
	return a.monitor
}

// State returns the application state.
func (a *Application) State() state.State {
	a.appLock.Lock()
	defer a.appLock.Unlock()
	return a.state
}

// Name returns application name
func (a *Application) Name() string {
	return a.name
}

// Started returns true if the application is started.
func (a *Application) Started() bool {
	return a.state.Status != state.Stopped
}

// Stop stops the current application.
func (a *Application) Stop() {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	if a.state.Status == state.Stopped {
		return
	}

	stopSig := os.Interrupt
	if a.srvState != nil {
		if err := a.srvState.Stop(a.processConfig.StopTimeout); err != nil {
			// kill the process if stop through GRPC doesn't work
			stopSig = os.Kill
		}
		a.srvState = nil
	}
	if a.state.ProcessInfo != nil {
		if err := a.state.ProcessInfo.Process.Signal(stopSig); err == nil {
			// no error on signal, so wait for it to stop
			_, _ = a.state.ProcessInfo.Process.Wait()
		}
		a.state.ProcessInfo = nil

		// cleanup drops
		a.cleanUp()
	}
	a.setState(state.Stopped, "Stopped")
}

// SetState sets the status of the application.
func (a *Application) SetState(status state.Status, msg string) {
	a.appLock.Lock()
	defer a.appLock.Unlock()
	a.setState(status, msg)
}

func (a *Application) watch(ctx context.Context, p app.Taggable, proc *process.Info, cfg map[string]interface{}) {
	go func() {
		var procState *os.ProcessState

		select {
		case ps := <-a.waitProc(proc.Process):
			procState = ps
		case <-a.bgContext.Done():
			a.Stop()
			return
		}

		a.appLock.Lock()
		if a.state.ProcessInfo != proc {
			// already another process started, another watcher is watching instead
			a.appLock.Unlock()
			return
		}
		a.state.ProcessInfo = nil
		srvState := a.srvState

		if srvState == nil || srvState.Expected() == proto.StateExpected_STOPPING {
			a.appLock.Unlock()
			return
		}

		msg := fmt.Sprintf("exited with code: %d", procState.ExitCode())
		a.setState(state.Crashed, msg)

		// it was a crash, cleanup anything required
		go a.cleanUp()
		a.start(ctx, p, cfg)
		a.appLock.Unlock()
	}()
}

func (a *Application) waitProc(proc *os.Process) <-chan *os.ProcessState {
	resChan := make(chan *os.ProcessState)

	go func() {
		procState, err := proc.Wait()
		if err != nil {
			// process is not a child - some OSs requires process to be child
			a.externalProcess(proc)
		}

		resChan <- procState
	}()

	return resChan
}

func (a *Application) setStateFromProto(pstatus proto.StateObserved_Status, msg string) {
	var status state.Status
	switch pstatus {
	case proto.StateObserved_STARTING:
		status = state.Starting
	case proto.StateObserved_CONFIGURING:
		status = state.Configuring
	case proto.StateObserved_HEALTHY:
		status = state.Running
	case proto.StateObserved_DEGRADED:
		status = state.Degraded
	case proto.StateObserved_FAILED:
		status = state.Failed
	case proto.StateObserved_STOPPING:
		status = state.Stopping
	}
	a.setState(status, msg)
}

func (a *Application) setState(status state.Status, msg string) {
	if a.state.Status != status || a.state.Message != msg {
		a.state.Status = status
		a.state.Message = msg
		if a.reporter != nil {
			go a.reporter.OnStateChange(a.id, a.name, a.state)
		}
	}
}

func (a *Application) cleanUp() {
	a.monitor.Cleanup(a.name, a.pipelineID)
}
