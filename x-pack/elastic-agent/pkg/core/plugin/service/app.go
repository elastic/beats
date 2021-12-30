// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/tokenbucket"
)

var (
	// ErrAppNotInstalled is returned when configuration is performed on not installed application.
	ErrAppNotInstalled = errors.New("application is not installed", errors.TypeApplication)
)

// Application encapsulates an application that is ran as a service by the system service manager.
type Application struct {
	bgContext  context.Context
	id         string
	name       string
	pipelineID string
	logLevel   string
	desc       *app.Descriptor
	srv        *server.Server
	srvState   *server.ApplicationState
	limiter    *tokenbucket.Bucket
	state      state.State
	reporter   state.Reporter

	uid int
	gid int

	monitor        monitoring.Monitor
	statusReporter status.Reporter

	processConfig *process.Config

	logger *logger.Logger

	credsPort     int
	credsWG       sync.WaitGroup
	credsListener net.Listener

	appLock sync.Mutex
}

// NewApplication creates a new instance of an applications.
func NewApplication(
	ctx context.Context,
	id, appName, pipelineID, logLevel string,
	credsPort int,
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
		credsPort:      credsPort,
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
	return a.srvState != nil
}

// SetState sets the status of the application.
func (a *Application) SetState(s state.Status, msg string, payload map[string]interface{}) {
	a.appLock.Lock()
	defer a.appLock.Unlock()
	a.setState(s, msg, payload)
}

// Start starts the application with a specified config.
func (a *Application) Start(ctx context.Context, _ app.Taggable, cfg map[string]interface{}) (err error) {
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
		a.setState(state.Starting, "Starting", nil)
		a.srvState.SetStatus(proto.StateObserved_STARTING, a.state.Message, a.state.Payload)
		a.srvState.UpdateConfig(a.srvState.Config())
	} else {
		a.setState(state.Starting, "Starting", nil)
		a.srvState, err = a.srv.Register(a, string(cfgStr))
		if err != nil {
			return err
		}

		// Set input types from the spec
		a.srvState.SetInputTypes(a.desc.Spec().ActionInputTypes)
	}

	defer func() {
		if err != nil {
			if a.srvState != nil {
				a.srvState.Destroy()
				a.srvState = nil
			}
		}
	}()

	if err := a.monitor.Prepare(a.desc.Spec(), a.pipelineID, a.uid, a.gid); err != nil {
		return err
	}

	if a.limiter != nil {
		a.limiter.Add()
	}

	// start the credentials listener for the service
	if err := a.startCredsListener(); err != nil {
		return err
	}

	// allow the service manager to ensure that the application is started, currently this does not start/stop
	// the actual service in the system service manager

	return nil
}

// Configure configures the application with the passed configuration.
func (a *Application) Configure(ctx context.Context, config map[string]interface{}) (err error) {
	defer func() {
		if err != nil {
			// inject App metadata
			err = errors.New(err, errors.M(errors.MetaKeyAppName, a.name), errors.M(errors.MetaKeyAppName, a.id))
			a.statusReporter.Update(state.Degraded, err.Error(), nil)
		}
	}()

	a.appLock.Lock()
	defer a.appLock.Unlock()

	if a.srvState == nil {
		return errors.New(ErrAppNotInstalled)
	}

	cfgStr, err := yaml.Marshal(config)
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}

	isRestartNeeded := plugin.IsRestartNeeded(a.logger, a.Spec(), a.srvState, config)

	err = a.srvState.UpdateConfig(string(cfgStr))
	if err != nil {
		return errors.New(err, errors.TypeApplication)
	}

	if isRestartNeeded {
		a.logger.Infof("initiating restart of '%s' due to config change", a.Name())
		a.appLock.Unlock()
		a.Stop()
		err = a.Start(ctx, a.desc, config)
		// lock back so it wont panic on deferred unlock
		a.appLock.Lock()
	}

	return err
}

// Stop stops the current application.
func (a *Application) Stop() {
	a.appLock.Lock()
	srvState := a.srvState
	a.appLock.Unlock()

	if srvState == nil {
		return
	}

	if err := srvState.Stop(a.processConfig.StopTimeout); err != nil {
		a.appLock.Lock()
		a.setState(
			state.Failed,
			fmt.Errorf("failed to stop after %s: %w", a.processConfig.StopTimeout, err).Error(),
			nil)
	} else {
		a.appLock.Lock()
		a.setState(state.Stopped, "Stopped", nil)
	}
	a.srvState = nil

	a.cleanUp()
	a.stopCredsListener()
	a.appLock.Unlock()
}

// Shutdown disconnects the service, but doesn't signal it to stop.
func (a *Application) Shutdown() {
	a.appLock.Lock()
	defer a.appLock.Unlock()
	a.logger.Infof("Signaling service to stop because of shutdown: %s", a.id)

	if a.srvState == nil {
		return
	}

	// destroy the application in the server, this skips sending
	// the expected stopping state to the service
	a.setState(state.Stopped, "Stopped", nil)
	a.srvState.Destroy()
	a.srvState = nil

	a.cleanUp()
	a.stopCredsListener()
}

// OnStatusChange is the handler called by the GRPC server code.
//
// It updates the status of the application and handles restarting the application is needed.
func (a *Application) OnStatusChange(
	s *server.ApplicationState,
	status proto.StateObserved_Status,
	msg string, payload map[string]interface{}) {

	a.appLock.Lock()
	defer a.appLock.Unlock()

	// If the application is stopped, do not update the state. Stopped is a final state
	// and should not be overridden.
	if a.state.Status == state.Stopped {
		return
	}

	a.setState(state.FromProto(status), msg, payload)
}

func (a *Application) setState(s state.Status, msg string, payload map[string]interface{}) {
	if a.state.Status != s ||
		a.state.Message != msg ||
		!reflect.DeepEqual(a.state.Payload, payload) {

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

func (a *Application) startCredsListener() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", a.credsPort))
	if err != nil {
		return errors.New(err, "failed to start connection credentials listener")
	}
	a.credsListener = lis
	a.credsWG.Add(1)
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				break
			}
			a.appLock.Lock()
			srvState := a.srvState
			a.appLock.Unlock()
			if srvState == nil {
				// application stopped
				_ = conn.Close()
				continue
			}
			if err := srvState.WriteConnInfo(conn); err != nil {
				_ = conn.Close()
				if err != io.EOF {
					a.logger.Errorf("failed to write connection credentials: %s", err)
				}
				continue
			}
			_ = conn.Close()
		}
		a.credsWG.Done()
	}()

	return nil
}

func (a *Application) stopCredsListener() {
	a.credsListener.Close()
	a.credsWG.Wait()
	a.credsListener = nil
}
