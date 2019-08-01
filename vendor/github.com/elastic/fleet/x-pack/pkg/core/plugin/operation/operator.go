// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/urso/ecslog"

	"github.com/elastic/fleet/x-pack/pkg/artifact/download"
	"github.com/elastic/fleet/x-pack/pkg/artifact/install"
	"github.com/elastic/fleet/x-pack/pkg/bus"
	"github.com/elastic/fleet/x-pack/pkg/bus/events"
	"github.com/elastic/fleet/x-pack/pkg/bus/topic"
	"github.com/elastic/fleet/x-pack/pkg/config"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/clientvault"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process/watcher"
)

// Operator runs Start/Stop/Update operations
// it is responsible for detecting reconnect to existing processes
// based on backed up configuration
// Enables running sidecars for processes.
// TODO: implement retry strategies
type Operator struct {
	logger         *ecslog.Logger
	clientVault    *clientvault.ClientVault
	processWatcher *watcher.Watcher
	config         *Config
	reattachInfo   *reattachCollection
	topics         map[topic.Topic]struct{}
	handlers       map[string]handleFunc
	handlersList   []string

	downloader download.Downloader
	installer  install.Installer
}

// NewOperator creates a new operator, this operator holds
// a collection of running processes, back it up
// Based on backed up collection it prepares clients, watchers... on init
func NewOperator(
	logger *ecslog.Logger,
	cv *clientvault.ClientVault,
	w *watcher.Watcher,
	config *config.Config,
	eb bus.Bus,
	stopOnStart bool,
	fetcher download.Downloader,
	installer install.Installer,
	inputTopics ...topic.Topic) (*Operator, error) {

	operatorConfig := &Config{}
	if err := config.Unpack(&operatorConfig); err != nil {
		return nil, err
	}

	rc := newReattachCollection(operatorConfig)
	if stopOnStart {
		stopAllPreviousRuns(logger, rc)
	}

	if operatorConfig.DownloadConfig == nil {
		return nil, fmt.Errorf("artifacts configuration not provided")
	}

	operator := &Operator{
		clientVault:    cv,
		processWatcher: w,
		config:         operatorConfig,
		logger:         logger,
		reattachInfo:   rc,
		downloader:     fetcher,
		installer:      installer,
	}

	operator.initHandlerMap()

	os.MkdirAll(operatorConfig.DownloadConfig.TargetDirectory, 0755)
	os.MkdirAll(operatorConfig.DownloadConfig.InstallPath, 0755)

	subscribedTo, err := operator.subscribe(eb, inputTopics...)
	if err != nil {
		return nil, errors.Wrap(err, "operator subscribing to events")
	}

	operator.topics = subscribedTo
	return operator, nil
}

// Start starts a new process based on a configuration
// specific configuration of new process is passed
func (o *Operator) Start(p Program) error {
	flow := []operation{
		newOperationFetch(o.logger, p, o.config, o.downloader),
		newOperationVerify(),
		newOperationInstall(o.logger, p, o.config, o.installer),
		newOperationStart(o.logger, p, o.reattachInfo, o.config, o.clientVault, o.processWatcher),
		newOperationConfig(o.logger, p, o.config, o.clientVault),
	}
	return o.runFlow(p, flow)
}

// Stop stops the running process, if process is already stopped it does not return an error
func (o *Operator) Stop(p Program) error {
	flow := []operation{
		newOperationStop(o.logger, p, o.reattachInfo, o.config, o.processWatcher, o.clientVault),
	}

	return o.runFlow(p, flow)
}

// PushConfig tries to push config to a running process
func (o *Operator) PushConfig(p Program) error {
	var flow []operation
	spec, err := p.Spec(o.config.DownloadConfig)
	if err != nil {
		return err
	}

	switch spec.Configurable {
	case ConfigurableFile:
		flow = []operation{
			// updates a configuration file and restarts a process
			newOperationStop(o.logger, p, o.reattachInfo, o.config, o.processWatcher, o.clientVault),
			newOperationStart(o.logger, p, o.reattachInfo, o.config, o.clientVault, o.processWatcher),
		}
	case ConfigurableGrpc:
		flow = []operation{
			newOperationConfig(o.logger, p, o.config, o.clientVault),
		}
	}

	return o.runFlow(p, flow)
}

// Sidecar defines a sidecar for a process.
// E.g: Metricbeat for filebeat 7.0.
func (o *Operator) Sidecar(sidecarType, p Program) error {
	// TODO: allow sidecar monitoring
	return nil
}

// StopSidecar tops existing sidecar for a process.
// E.g: Metricbeat for filebeat 7.0.
func (o *Operator) StopSidecar(sidecarType, p Program) error {
	// TODO: allow sidecar monitoring
	return nil
}

func (o *Operator) runFlow(p Program, operations []operation) error {
	if len(operations) == 0 {
		o.logger.Infof("operator received event with no operations for program '%s'", p.ID())
		return nil
	}

	for _, op := range operations {
		shouldRun, err := op.Check()
		if err != nil {
			return err
		}

		if !shouldRun {
			o.logger.Infof("operation '%s' skipped for %s.%s", op.Name(), p.BinaryName(), p.Version())
			continue
		}

		if err := op.Run(); err != nil {
			return err
		}
	}

	return nil
}

func stopAllPreviousRuns(logger *ecslog.Logger, rc *reattachCollection) error {
	programs, err := rc.items()
	if err != nil {
		return err
	}

	var result error

	for _, info := range programs {
		if err := process.Stop(logger, info.PID); err != nil {
			result = multierror.Append(result, err)
		}
		rc.removeProcess(info.PID)
	}

	return result
}

func (o *Operator) subscribe(eb bus.Bus, inputTopics ...topic.Topic) (map[topic.Topic]struct{}, error) {
	if len(inputTopics) == 0 {
		err := eb.Subscribe(topic.StateChanges, o.stateChangeHandler)
		if err != nil {
			return nil, err
		}
		o.logger.Debugf("Operator subscribed for topic: '%v'", topic.StateChanges)

		return map[topic.Topic]struct{}{topic.StateChanges: struct{}{}}, nil
	}

	subscriptionMap := make(map[topic.Topic]struct{})
	for _, t := range inputTopics {
		err := eb.Subscribe(t, o.stateChangeHandler)
		if err != nil {
			return nil, err
		}

		o.logger.Debugf("Operator subscribed for topic: '%v'", t)
		subscriptionMap[t] = struct{}{}
	}

	return subscriptionMap, nil
}
func (o *Operator) stateChangeHandler(t topic.Topic, e bus.Event) {
	o.logger.Debugf("Operator: received event %v", t)

	// TODO: reuse event from stateresolver
	if !o.isSubscribed(t) {
		o.logger.Infof("Operator: not subscribed to a topic %v", t)
		return
	}

	stateChangeEvent, ok := e.(*events.StateChangeEvent)
	if !ok {
		o.logger.Errorf("Operator: received event which is not 'StateChangeEvent'")
		return
	}

	for _, step := range stateChangeEvent.Steps {
		handler, found := o.handlers[step.ID]
		if !found {
			o.logger.Errorf("Operator: received unexpected event '%s'. Available handlers are for %v", step.ID, o.handlersList)
			return
		}

		if err := handler(step); err != nil {
			o.logger.Errorf("Operator: failed to execute step %s, error: %v", step.ID, err)
			return
		}
	}
}

func (o *Operator) isSubscribed(t topic.Topic) bool {
	_, ok := o.topics[t]
	return ok
}
