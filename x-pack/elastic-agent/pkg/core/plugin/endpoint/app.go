// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package endpoint

import (
	"context"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

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
	// ErrAppNotInstalled is returned when configuration is performed on not installed application.
	ErrAppNotInstalled = errors.New("application is not installed", errors.TypeApplication)
)

// Application encapsulates the Endpoint application that is ran by the system service manager.
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

	monitor monitoring.Monitor

	processConfig  *process.Config
	downloadConfig *artifact.Config
	retryConfig    *retry.Config

	logger *logger.Logger

	appLock sync.Mutex
}

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

// SetState sets the status of the application.
func (a *Application) SetState(status state.Status, msg string) {
	a.appLock.Lock()
	defer a.appLock.Unlock()
	a.setState(status, msg)
}

// Start starts the application with a specified config.
func (a *Application) Start(ctx context.Context, t app.Taggable, cfg map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
		}
	}()

	a.appLock.Lock()
	defer a.appLock.Unlock()

	cfgStr, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// already started
	if a.srvState != nil {
		a.setState(state.Starting, "Starting")
		a.srvState.SetStatus(proto.StateObserved_STARTING, a.state.Message)
		a.srvState.UpdateConfig(string(cfgStr))
	} else {
		a.srvState, err = a.srv.Register(a, string(cfgStr))
		if err != nil {
			return err
		}
	}
	return nil
}

// Configure configures the application with the passed configuration.
func (a *Application) Configure(_ context.Context, config map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
		}
	}()

	a.appLock.Lock()
	defer a.appLock.Unlock()

	if a.state.Status == state.Stopped {
		return errors.New(ErrAppNotInstalled)
	}
	if a.srvState == nil {
		return errors.New(ErrAppNotInstalled)
	}

	cfgStr, err := yaml.Marshal(config)
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}
	err = a.srvState.UpdateConfig(string(cfgStr))
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}
	return nil
}

// Stop stops the current application.
func (a *Application) Stop() {
	a.appLock.Lock()
	defer a.appLock.Unlock()

	if a.srvState == nil {
		return
	}

	if err := a.srvState.Stop(a.processConfig.StopTimeout); err != nil {
		a.setState(state.Failed, errors.New(err, "Failed to stopped").Error())
	} else {
		a.setState(state.Stopped, "Stopped")
	}
}

// OnStatusChange is the handler called by the GRPC server code.
//
// It updates the status of the application and handles restarting the application is needed.
func (a *Application) OnStatusChange(s *server.ApplicationState, status proto.StateObserved_Status, msg string) {
	a.appLock.Lock()

	// If the application is stopped, do not update the state. Stopped is a final state
	// and should not be overridden.
	if a.state.Status == state.Stopped {
		a.appLock.Unlock()
		return
	}

	a.setStateFromProto(status, msg)
	a.appLock.Unlock()
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
