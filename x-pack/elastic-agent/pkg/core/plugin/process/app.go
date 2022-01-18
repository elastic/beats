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
	procExitTimeout  = 10 * time.Second
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
	watchClosers map[int]context.CancelFunc

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
		watchClosers:   make(map[int]context.CancelFunc),
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

	if srvState != nil {
		// signal stop through GRPC, wait and kill is performed later in gracefulKill
		if err := srvState.Stop(a.processConfig.StopTimeout); err != nil {
			err := fmt.Errorf("failed to stop after %s: %w", a.processConfig.StopTimeout, err)
			a.setState(state.Failed, err.Error(), nil)

			a.logger.Error(err)
		}

	}

	a.appLock.Lock()
	defer a.appLock.Unlock()

	a.srvState = nil
	if a.state.ProcessInfo != nil {
		// stop and clean watcher
		a.stopWatcher(a.state.ProcessInfo)
		a.gracefulKill(a.state.ProcessInfo)

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
			return
		case <-ctx.Done():
			// closer called
			return
		}

		a.appLock.Lock()
		defer a.appLock.Unlock()
		if a.state.ProcessInfo != proc {
			// already another process started, another watcher is watching instead
			a.gracefulKill(proc)
			return
		}

		// stop the watcher
		a.stopWatcher(a.state.ProcessInfo)

		// was already stopped by Stop, do not restart
		if a.state.Status == state.Stopped {
			return
		}

		a.state.ProcessInfo = nil
		srvState := a.srvState

		if srvState == nil || srvState.Expected() == proto.StateExpected_STOPPING {
			return
		}

		msg := fmt.Sprintf("exited with code: %d", procState.ExitCode())
		a.setState(state.Restarting, msg, nil)

		// it was a crash
		a.start(ctx, p, cfg, true)
	}()
}

func (a *Application) stopWatcher(procInfo *process.Info) {
	if procInfo != nil {
		if closer, ok := a.watchClosers[procInfo.PID]; ok {
			closer()
			delete(a.watchClosers, procInfo.PID)
		}
	}
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
		if state.IsStateFiltered(msg, payload) {
			return
		}

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

func (a *Application) gracefulKill(proc *process.Info) {
	if proc == nil || proc.Process == nil {
		return
	}

	// send stop signal to request stop
	if err := proc.Stop(); err != nil {
		a.logger.Errorf("failed to stop %s: %v", a.Name(), err))
	}

	var wg sync.WaitGroup
	doneChan := make(chan struct{})
	wg.Add(1)
	go func() {
		wg.Done()

		if _, err := proc.Process.Wait(); err != nil {
			// process is not a child - some OSs requires process to be child
			a.externalProcess(proc.Process)
		}
		close(doneChan)
	}()

	// wait for awaiter
	wg.Wait()

	// kill in case it's still running after timeout
	t := time.NewTimer(procExitTimeout)
	defer t.Stop()
	select {
	case <-doneChan:
	case <-t.C:
		a.logger.Infof("gracefulKill timed out after %d, killing %s",
			procExitTimeout, a.Name())
		_ = proc.Process.Kill()
	}
}
