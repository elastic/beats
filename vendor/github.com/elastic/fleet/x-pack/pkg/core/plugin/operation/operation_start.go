// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/urso/ecslog"
	"gopkg.in/yaml.v2"

	"github.com/elastic/fleet/x-pack/pkg/core/plugin/authority"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/clientvault"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process/watcher"
	"github.com/elastic/fleet/x-pack/pkg/core/remoteconfig/grpc"
	"github.com/elastic/fleet/x-pack/pkg/tokenbucket"
)

const (
	configurationFlag     = "-c"
	configFileTempl       = "%s.yml" // providing beat id
	configFilePermissions = 0644     // writable only by owner
)

var (
	// ErrUpdateClientFailed happens when call to a client vault returns an error.
	ErrUpdateClientFailed = errors.New("updating clientvault failed")
	// ErrStoringReattachInfoFailed happens when call to reattach collection fails
	// might be related to filesystem. Check logs for more information.
	ErrStoringReattachInfoFailed = errors.New("backing up reattach information failed")
)

// operationStart start installed process
// skips if process is already running
type operationStart struct {
	program        Program
	logger         *ecslog.Logger
	operatorConfig *Config
	rc             *reattachCollection
	cv             *clientvault.ClientVault
	w              *watcher.Watcher
	limiter        *tokenbucket.Bucket

	pi *process.Info
}

func newOperationStart(
	logger *ecslog.Logger,
	program Program,
	rc *reattachCollection,
	operatorConfig *Config,
	cv *clientvault.ClientVault,
	w *watcher.Watcher) *operationStart {
	// TODO: make configurable
	b, _ := tokenbucket.NewTokenBucket(3, 3, 1*time.Second)

	return &operationStart{
		logger:         logger,
		program:        program,
		rc:             rc,
		operatorConfig: operatorConfig,
		cv:             cv,
		w:              w,
		limiter:        b,
	}
}

// Name is human readable name identifying an operation
func (o *operationStart) Name() string {
	return "operation-start"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationStart) Check() (bool, error) {
	runningProcesses, err := o.rc.items()
	if err != nil {
		o.logger.Errorf("failed to load reattach collection for %s.%s: %v", o.program.BinaryName(), o.program.Version(), err)
		return true, nil
	}

	newID := o.program.ID()
	for _, p := range runningProcesses {
		prog := NewProgramWithContext(p.ExecutionContext, nil)
		if prog.ID() == newID {
			return false, nil
		}
	}

	return true, nil
}

// Run runs the operation
func (o *operationStart) Run() (err error) {
	defer func() {
		if err != nil {
			// kill the process if something failed
			o.rollback()
			err = errors.Wrap(err, o.Name())
		}
	}()

	spec, err := o.program.Spec(o.operatorConfig.DownloadConfig)
	if err != nil {
		return err
	}

	var processCreds *process.Creds
	ca, err := authority.NewCA()
	if err != nil {
		return errors.Wrap(err, "operation.Start")
	}

	if isGrpcConfigurable(spec.Configurable) {
		// processPK and Cert serves as a server credentials
		processPair, err := ca.GeneratePair()
		if err != nil {
			return errors.Wrap(err, "failed to generate credentials")
		}

		processCreds = &process.Creds{
			CaCert: ca.Crt(),
			PK:     processPair.Key,
			Cert:   processPair.Crt,
		}
	}

	uid, gid, err := getProcCredentials(spec)
	if err != nil {
		return err
	}

	if err := o.configureByFile(&spec); err != nil {
		return err
	}

	o.pi, err = process.Start(o.logger, spec.BinaryPath, o.operatorConfig.ProcessConfig, uid, gid, processCreds, spec.Args...)
	if err != nil {
		return err
	}

	// setup watcher
	exitChan, err := o.w.Watch(o.pi.Process)
	if err != nil {
		return err
	}

	go o.handleExit(exitChan)

	// generate client
	// TODO: generate real client
	if isGrpcConfigurable(spec.Configurable) {
		clientPair, err := ca.GeneratePair()
		if err != nil {
			return err
		}

		connectionProvider := grpc.NewConnectionProvider(o.pi.Address, ca.Crt(), clientPair.Key, clientPair.Crt)
		if err := o.cv.UpdateClient(o.program.ID(), connectionProvider); err != nil {
			o.logger.Errorf("failed to update client for %s.%s: %v", o.program.BinaryName(), o.program.Version(), err)
			return ErrUpdateClientFailed
		}
	}

	// generate reattach
	if err := o.rc.addProcess(o.program.ExecutionContext(), o.pi); err != nil {
		o.logger.Errorf("failed to add process to reattach collection for %s.%s: %v", o.program.BinaryName(), o.program.Version(), err)
		return ErrStoringReattachInfoFailed
	}

	return nil
}

// rollback rollbacks the effect of the operation
func (o *operationStart) rollback() error {
	if o.pi == nil {
		return nil
	}

	process.Stop(o.logger, o.pi.PID)
	o.w.UnWatch(o.pi.PID)
	o.cv.UpdateClient(o.program.ID(), nil)
	o.rc.removeProcess(o.pi.PID)

	return o.rc.removeProcess(o.pi.PID)
}

func (o *operationStart) handleExit(exitChan <-chan watcher.CloseReason) {
	reason := <-exitChan
	switch reason {
	case watcher.ProcessCrashed:
		o.cv.UpdateClient(o.program.ID(), nil)
		if o.pi != nil {
			o.rc.removeProcess(o.pi.PID)
		}

		// wait for rate limiter to allow restart
		o.limiter.Add()

		if v, _ := o.Check(); v {
			go o.Run()
		}
	case watcher.ProcessClosed:
		o.cv.UpdateClient(o.program.ID(), nil)
		if o.pi != nil {
			o.rc.removeProcess(o.pi.PID)
		}
		if o.limiter != nil {
			o.limiter.Close()
		}
	}
}

func (o *operationStart) configureByFile(spec *ProcessSpec) error {
	// check if configured by file
	if spec.Configurable != ConfigurableFile {
		return nil
	}

	// save yaml as filebeat_id.yml
	filename := fmt.Sprintf(configFileTempl, o.program.ID())
	filePath, err := filepath.Abs(filepath.Join(o.operatorConfig.DownloadConfig.InstallPath, filename))
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, configFilePermissions)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	if err := encoder.Encode(o.program.Config()); err != nil {
		return err
	}
	defer encoder.Close()

	// update args
	return o.updateSpecConfig(spec, filePath)
}

func (o *operationStart) updateSpecConfig(spec *ProcessSpec, configPath string) error {
	// check if config is already provided
	configIndex := -1
	for i, v := range spec.Args {
		if v == configurationFlag {
			configIndex = i
			break
		}
	}

	if configIndex != -1 {
		// -c provided
		if len(spec.Args) == configIndex+1 {
			// -c is last argument, appending
			spec.Args = append(spec.Args, configPath)
		}
		spec.Args[configIndex+1] = configPath
		return nil
	}

	spec.Args = append(spec.Args, configurationFlag, configPath)
	return nil
}

func isGrpcConfigurable(configurable string) bool {
	return configurable == ConfigurableGrpc
}
