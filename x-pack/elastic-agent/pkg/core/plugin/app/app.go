// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/remoteconfig"
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
	state           state.State
	grpcClient      remoteconfig.Client
	clientFactory   remoteconfig.ConnectionCreator
	limiter         *tokenbucket.Bucket
	failureReporter ReportFailureFunc

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
	factory remoteconfig.ConnectionCreator,
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
		clientFactory:   factory,
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

// Name returns application name
func (a *Application) Name() string {
	return a.name
}

// Stop stops the current application.
func (a *Application) Stop() {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	if a.state.Status == state.Running && a.state.ProcessInfo != nil {
		if closeClient, ok := a.grpcClient.(closer); ok {
			closeClient.Close()
		}

		process.Stop(a.logger, a.state.ProcessInfo.PID)

		a.state.Status = state.Stopped
		a.state.ProcessInfo = nil
		a.grpcClient = nil

		// remove generated configuration if present
		filename := fmt.Sprintf(configFileTempl, a.id)
		filePath, err := filepath.Abs(filepath.Join(a.downloadConfig.InstallPath, filename))
		if err == nil {
			// ignoring error: not critical
			os.Remove(filePath)
		}

		// cleanup drops
		a.monitor.Cleanup(a.name, a.pipelineID)
	}
}

// State returns the state of the application [Running, Stopped].
func (a *Application) State() state.State {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	return a.state
}

func (a *Application) watch(ctx context.Context, p Taggable, proc *os.Process, cfg map[string]interface{}) {
	go func() {
		var procState *os.ProcessState

		select {
		case ps := <-a.waitProc(proc):
			procState = ps
		case <-a.bgContext.Done():
			a.Stop()
			return
		}

		a.appLock.Lock()
		s := a.state.Status
		a.state.Status = state.Stopped
		a.state.ProcessInfo = nil
		a.appLock.Unlock()

		if procState.Success() {
			return
		}

		if s == state.Running {
			// it was a crash, report it async not to block
			// process management with networking issues
			go a.reportCrash(ctx)
			a.Start(ctx, p, cfg)
		}
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
