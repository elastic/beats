// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
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
	desc         *app.Descriptor
	srv          *server.Server
	srvState     *server.ApplicationState
	limiter      *tokenbucket.Bucket
	startContext context.Context
	tag          app.Taggable
	state        state.State
	reporter     state.Reporter

	uid int
	gid int

	monitor        monitoring.Monitor
	statusReporter status.Reporter

	processConfig *process.Config

	logger *logger.Logger

	appLock          sync.Mutex
	restartCanceller context.CancelFunc
	restartConfig    map[string]interface{}
}

// ArgsDecorator decorates arguments before calling an application
type ArgsDecorator func([]string) []string

// NewApplication creates a new instance of an applications. It will not automatically start
// the application.
func NewApplication(
	ctx context.Context,
	id, appName, pipelineID, logLevel string,
	desc *app.Descriptor,
	srv *server.Server,
	cfg *configuration.SettingsConfig,
	logger *logger.Logger,
	reporter state.Reporter,
	monitor monitoring.Monitor,
	statusController status.Controller) (*Application, error) {

	s := desc.ProcessSpec()
	uid, gid, err := s.UserGroup()
	if err != nil {
		return nil, err
	}

	b, _ := tokenbucket.NewTokenBucket(ctx, 3, 3, 1*time.Second)
	return &Application{
		bgContext:     ctx,
		id:            id,
		name:          appName,
		pipelineID:    pipelineID,
		logLevel:      logLevel,
		desc:          desc,
		srv:           srv,
		processConfig: cfg.ProcessConfig,
		logger:        logger,
		limiter:       b,
		state: state.State{
			Status: state.Stopped,
		},
		reporter:       reporter,
		monitor:        monitor,
		uid:            uid,
		gid:            gid,
		statusReporter: statusController.RegisterApp(id, appName),
	}, nil
}

// Monitor returns monitoring handler of this app.
func (a *Application) Monitor() monitoring.Monitor {
	return a.monitor
}

// Spec returns the program spec of this app.
func (a *Application) Spec() program.Spec {
	return a.desc.Spec()
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
	return a.state.Status != state.Stopped && a.state.Status != state.Crashed && a.state.Status != state.Failed
}

// Stop stops the current application.
func (a *Application) Stop() {
	a.appLock.Lock()
	status := a.state.Status
	srvState := a.srvState
	a.appLock.Unlock()

	if status == state.Stopped {
		return
	}

	stopSig := os.Interrupt
	if srvState != nil {
		if err := srvState.Stop(a.processConfig.StopTimeout); err != nil {
			// kill the process if stop through GRPC doesn't work
			stopSig = os.Kill
		}
	}

	a.appLock.Lock()
	defer a.appLock.Unlock()

	a.srvState = nil
	if a.state.ProcessInfo != nil {
		if err := a.state.ProcessInfo.Process.Signal(stopSig); err == nil {
			// no error on signal, so wait for it to stop
			_, _ = a.state.ProcessInfo.Process.Wait()
		}
		a.state.ProcessInfo = nil

		// cleanup drops
		a.cleanUp()
	}
	a.setState(state.Stopped, "Stopped", nil)
}

// Shutdown stops the application (aka. subprocess).
func (a *Application) Shutdown() {
	a.logger.Infof("Signaling application to stop because of shutdown: %s", a.id)
	a.Stop()
}

// SetState sets the status of the application.
func (a *Application) SetState(s state.Status, msg string, payload map[string]interface{}) {
	a.appLock.Lock()
	defer a.appLock.Unlock()
	a.setState(s, msg, payload)
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
		a.setState(state.Crashed, msg, nil)

		// it was a crash
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

func (a *Application) setState(s state.Status, msg string, payload map[string]interface{}) {
	if a.state.Status != s || a.state.Message != msg || !reflect.DeepEqual(a.state.Payload, payload) {
		a.state.Status = s
		a.state.Message = msg
		a.state.Payload = payload
		if a.reporter != nil {
			go a.reporter.OnStateChange(a.id, a.name, a.state)
		}
		a.statusReporter.Update(s, msg, payload)
	}
}

func (a *Application) cleanUp() {
	a.monitor.Cleanup(a.desc.Spec(), a.pipelineID)
}
