// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/tokenbucket"
)

var (
	// ErrAppNotRunning is returned when configuration is performed on not running application.
	ErrAppNotRunning = errors.New("application is not running", errors.TypeApplication)
	// ErrClientNotFound signals that client is not present in the vault.
	ErrClientNotFound = errors.New("client not present", errors.TypeApplication)
	// ErrClientNotConfigurable happens when stored client does not implement Config func
	ErrClientNotConfigurable = errors.New("client does not provide configuration", errors.TypeApplication)
)

// ReportFailureFunc is a callback func used to report async failures due to crashes.
type ReportFailureFunc func(context.Context, string, error)

// Application encapsulates a concrete application ran by elastic-agent e.g Beat.
type Application struct {
	bgContext       context.Context
	id              string
	name            string
	pipelineID      string
	logLevel        string
	spec            Specifier
	srv             *server.Server
	srvState        *server.ApplicationState
	limiter         *tokenbucket.Bucket
	failureReporter ReportFailureFunc
	startContext    context.Context
	tag             Taggable
	state           state.State

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
	spec Specifier,
	srv *server.Server,
	cfg *config.Config,
	logger *logger.Logger,
	failureReporter ReportFailureFunc,
	monitor monitoring.Monitor) (*Application, error) {

	s := spec.Spec()
	uid, gid, err := getUserGroup(s)
	if err != nil {
		return nil, err
	}

	b, _ := tokenbucket.NewTokenBucket(ctx, 3, 3, 1*time.Second)
	return &Application{
		bgContext:       ctx,
		id:              id,
		name:            appName,
		pipelineID:      pipelineID,
		logLevel:        logLevel,
		spec:            spec,
		srv:             srv,
		processConfig:   cfg.ProcessConfig,
		downloadConfig:  cfg.DownloadConfig,
		retryConfig:     cfg.RetryConfig,
		logger:          logger,
		limiter:         b,
		failureReporter: failureReporter,
		monitor:         monitor,
		uid:             uid,
		gid:             gid,
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
		a.monitor.Cleanup(a.name, a.pipelineID)
	}
	a.state.Status = state.Stopped
	a.state.Message = "Stopped"
}

func (a *Application) watch(ctx context.Context, p Taggable, proc *process.Info, cfg map[string]interface{}) {
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
		a.state.ProcessInfo = nil
		srvState := a.srvState

		if srvState == nil || srvState.Expected() == proto.StateExpected_STOPPING {
			a.appLock.Unlock()
			return
		}

		msg := fmt.Sprintf("Exited with code: %d", procState.ExitCode())
		a.state.Status = state.Crashed
		a.state.Message = msg
		a.appLock.Unlock()

		// it was a crash, report it async not to block
		// process management with networking issues
		go a.reportCrash(ctx)
		a.Start(ctx, p, cfg)
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

func (a *Application) reportCrash(ctx context.Context) {
	a.monitor.Cleanup(a.name, a.pipelineID)

	// TODO: reporting crash
	if a.failureReporter != nil {
		crashError := errors.New(
			fmt.Sprintf("application '%s' crashed", a.id),
			errors.TypeApplicationCrash,
			errors.M(errors.MetaKeyAppName, a.name),
			errors.M(errors.MetaKeyAppName, a.id))
		a.failureReporter(ctx, a.name, crashError)
	}
}
