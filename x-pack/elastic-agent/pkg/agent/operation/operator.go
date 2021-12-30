// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/stateresolver"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/noop"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/service"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/status"
)

const (
	isMonitoringMetricsFlag = 1 << 0
	isMonitoringLogsFlag    = 1 << 1
)

type waiter interface {
	Wait()
}

// Operator runs Start/Stop/Update operations
// it is responsible for detecting reconnect to existing processes
// based on backed up configuration
// Enables running sidecars for processes.
// TODO: implement retry strategies
type Operator struct {
	bgContext     context.Context
	pipelineID    string
	logger        *logger.Logger
	agentInfo     *info.AgentInfo
	config        *configuration.SettingsConfig
	handlers      map[string]handleFunc
	stateResolver *stateresolver.StateResolver
	srv           *server.Server
	reporter      state.Reporter
	monitor       monitoring.Monitor
	isMonitoring  int

	apps     map[string]Application
	appsLock sync.Mutex

	downloader       download.Downloader
	verifier         download.Verifier
	installer        install.InstallerChecker
	uninstaller      uninstall.Uninstaller
	statusController status.Controller
	statusReporter   status.Reporter
}

// NewOperator creates a new operator, this operator holds
// a collection of running processes, back it up
// Based on backed up collection it prepares clients, watchers... on init
func NewOperator(
	ctx context.Context,
	logger *logger.Logger,
	agentInfo *info.AgentInfo,
	pipelineID string,
	config *configuration.SettingsConfig,
	fetcher download.Downloader,
	verifier download.Verifier,
	installer install.InstallerChecker,
	uninstaller uninstall.Uninstaller,
	stateResolver *stateresolver.StateResolver,
	srv *server.Server,
	reporter state.Reporter,
	monitor monitoring.Monitor,
	statusController status.Controller) (*Operator, error) {
	if config.DownloadConfig == nil {
		return nil, fmt.Errorf("artifacts configuration not provided")
	}

	operator := &Operator{
		bgContext:        ctx,
		config:           config,
		pipelineID:       pipelineID,
		logger:           logger,
		agentInfo:        agentInfo,
		downloader:       fetcher,
		verifier:         verifier,
		installer:        installer,
		uninstaller:      uninstaller,
		stateResolver:    stateResolver,
		srv:              srv,
		apps:             make(map[string]Application),
		reporter:         reporter,
		monitor:          monitor,
		statusController: statusController,
		statusReporter:   statusController.RegisterComponent("operator-" + pipelineID),
	}

	operator.initHandlerMap()

	os.MkdirAll(config.DownloadConfig.TargetDirectory, 0755)
	os.MkdirAll(config.DownloadConfig.InstallPath, 0755)

	return operator, nil
}

// State describes the current state of the system.
// Reports all known applications and theirs states. Whether they are running
// or not, and if they are information about process is also present.
func (o *Operator) State() map[string]state.State {
	result := make(map[string]state.State)

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for k, v := range o.apps {
		result[k] = v.State()
	}

	return result
}

// Specs returns all program specifications
func (o *Operator) Specs() map[string]program.Spec {
	r := make(map[string]program.Spec)

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, app := range o.apps {
		// use app.Name() instead of the (map) key so we can easy find the "_monitoring" processes
		r[app.Name()] = app.Spec()
	}

	return r
}

// Close stops all programs handled by operator and clears state
func (o *Operator) Close() error {
	o.monitor.Close()
	o.statusReporter.Unregister()

	return o.HandleConfig(configrequest.New("", time.Now(), nil))
}

// HandleConfig handles configuration for a pipeline and performs actions to achieve this configuration.
func (o *Operator) HandleConfig(cfg configrequest.Request) (err error) {
	defer func() {
		err = filterContextCancelled(err)
	}()

	_, stateID, steps, ack, err := o.stateResolver.Resolve(cfg)
	if err != nil {
		if err == filterContextCancelled(err) {
			// error is not filtered and should be reported
			o.statusReporter.Update(state.Failed, err.Error(), nil)
			err = errors.New(err, errors.TypeConfig, fmt.Sprintf("operator: failed to resolve configuration %s, error: %v", cfg, err))
		}

		return err
	}
	o.statusController.UpdateStateID(stateID)

	for _, step := range steps {
		if !strings.EqualFold(step.ProgramSpec.Cmd, monitoringName) {
			if _, isSupported := program.SupportedMap[strings.ToLower(step.ProgramSpec.Cmd)]; !isSupported {
				// mark failed, new config cannot be run
				msg := fmt.Sprintf("program '%s' is not supported", step.ProgramSpec.Cmd)
				o.statusReporter.Update(state.Failed, msg, nil)
				return errors.New(msg,
					errors.TypeApplication,
					errors.M(errors.MetaKeyAppName, step.ProgramSpec.Cmd))
			}
		}

		handler, found := o.handlers[step.ID]
		if !found {
			msg := fmt.Sprintf("operator: received unexpected event '%s'", step.ID)
			o.statusReporter.Update(state.Failed, msg, nil)
			return errors.New(msg, errors.TypeConfig)
		}

		if err := handler(step); err != nil {
			msg := fmt.Sprintf("operator: failed to execute step %s, error: %v", step.ID, err)
			o.statusReporter.Update(state.Failed, msg, nil)
			return errors.New(err, errors.TypeConfig, msg)
		}
	}

	// Ack the resolver should state for next call.
	o.statusReporter.Update(state.Healthy, "", nil)
	ack()

	return nil
}

// Shutdown handles shutting down the running apps for Agent shutdown.
func (o *Operator) Shutdown() {
	//  wait for installer and downloader
	if awaitable, ok := o.installer.(waiter); ok {
		o.logger.Infof("waiting for installer of pipeline '%s' to finish", o.pipelineID)
		awaitable.Wait()
		o.logger.Debugf("pipeline installer '%s' done", o.pipelineID)
	}

	wg := sync.WaitGroup{}
	started := time.Now()
	for _, a := range o.apps {
		// shutdown apps concurrently.
		// TODO(Anderson): it's fine, right?
		wg.Add(1)
		go func(a Application) {
			a.Shutdown()
			wg.Done()
		}(a)
	}
	wg.Wait()
	o.logger.Debugf("took %s to shutdown %d apps",
		time.Now().Sub(started), len(o.apps))
}

// Start starts a new process based on a configuration
// specific configuration of new process is passed
func (o *Operator) start(p Descriptor, cfg map[string]interface{}) (err error) {
	flow := []operation{
		newRetryableOperations(
			o.logger,
			o.config.RetryConfig,
			newOperationFetch(o.logger, p, o.config, o.downloader),
			newOperationVerify(p, o.config, o.verifier),
		),
		newOperationInstall(o.logger, p, o.config, o.installer),
		newOperationStart(o.logger, p, o.config, cfg),
		newOperationConfig(o.logger, o.config, cfg),
	}
	return o.runFlow(p, flow)
}

// Stop stops the running process, if process is already stopped it does not return an error
func (o *Operator) stop(p Descriptor) (err error) {
	flow := []operation{
		newOperationStop(o.logger, o.config),
		newOperationUninstall(o.logger, p, o.uninstaller),
	}

	return o.runFlow(p, flow)
}

// PushConfig tries to push config to a running process
func (o *Operator) pushConfig(p Descriptor, cfg map[string]interface{}) error {
	flow := []operation{
		newOperationConfig(o.logger, o.config, cfg),
	}

	return o.runFlow(p, flow)
}

func (o *Operator) runFlow(p Descriptor, operations []operation) error {
	if len(operations) == 0 {
		o.logger.Infof("operator received event with no operations for program '%s'", p.ID())
		return nil
	}

	app, err := o.getApp(p)
	if err != nil {
		return err
	}

	for _, op := range operations {
		if err := o.bgContext.Err(); err != nil {
			return err
		}

		shouldRun, err := op.Check(o.bgContext, app)
		if err != nil {
			return err
		}

		if !shouldRun {
			o.logger.Infof("operation '%s' skipped for %s.%s", op.Name(), p.BinaryName(), p.Version())
			continue
		}

		o.logger.Debugf("running operation '%s' for %s.%s", op.Name(), p.BinaryName(), p.Version())
		if err := op.Run(o.bgContext, app); err != nil {
			return err
		}
	}

	// when application is stopped remove from the operator
	if app.State().Status == state.Stopped {
		o.deleteApp(p)
	}

	return nil
}

func (o *Operator) getApp(p Descriptor) (Application, error) {
	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	id := p.ID()

	o.logger.Debugf("operator is looking for %s in app collection: %v", p.ID(), o.apps)
	if a, ok := o.apps[id]; ok {
		return a, nil
	}

	desc, ok := p.(*app.Descriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor is not an app.Descriptor")
	}

	// TODO: (michal) join args into more compact options version
	var a Application
	var err error

	monitor := o.monitor
	appName := p.BinaryName()
	if app.IsSidecar(p) {
		// make watchers unmonitorable
		monitor = noop.NewMonitor()
		appName += "_monitoring"
	}

	if p.ServicePort() == 0 {
		// Applications without service ports defined are ran as through the process application type.
		a, err = process.NewApplication(
			o.bgContext,
			p.ID(),
			appName,
			o.pipelineID,
			o.config.LoggingConfig.Level.String(),
			desc,
			o.srv,
			o.config,
			o.logger,
			o.reporter,
			monitor,
			o.statusController)
	} else {
		// Service port is defined application is ran with service application type, with it fetching
		// the connection credentials through the defined service port.
		a, err = service.NewApplication(
			o.bgContext,
			p.ID(),
			appName,
			o.pipelineID,
			o.config.LoggingConfig.Level.String(),
			p.ServicePort(),
			desc,
			o.srv,
			o.config,
			o.logger,
			o.reporter,
			monitor,
			o.statusController)
	}

	if err != nil {
		return nil, err
	}

	o.apps[id] = a
	return a, nil
}

func (o *Operator) deleteApp(p Descriptor) {
	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	id := p.ID()

	o.logger.Debugf("operator is removing %s from app collection: %v", p.ID(), o.apps)
	delete(o.apps, id)
}

func filterContextCancelled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
