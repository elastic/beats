// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/fleet/x-pack/pkg/core/plugin/clientvault"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process"
	"github.com/elastic/fleet/x-pack/pkg/core/plugin/process/watcher"
	"github.com/pkg/errors"
	"github.com/urso/ecslog"
)

// operationStop stops the running process
// skips if process is already skipped
type operationStop struct {
	program        Program
	logger         *ecslog.Logger
	operatorConfig *Config

	rc *reattachCollection
	w  *watcher.Watcher
	cv *clientvault.ClientVault
}

func newOperationStop(
	logger *ecslog.Logger,
	p Program,
	rc *reattachCollection,
	operatorConfig *Config,
	w *watcher.Watcher,
	cv *clientvault.ClientVault) *operationStop {
	return &operationStop{
		logger:         logger,
		program:        p,
		rc:             rc,
		w:              w,
		cv:             cv,
		operatorConfig: operatorConfig,
	}
}

// Name is human readable name identifying an operation
func (o *operationStop) Name() string {
	return "operation-stop"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationStop) Check() (bool, error) {
	return true, nil
}

// Run runs the operation
func (o *operationStop) Run() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, o.Name())
		}
	}()

	if o.program == nil {
		return fmt.Errorf("operation '%s' does not contain runnable program", o.Name())
	}
	info, err := o.getReattachInfo()
	if err != nil {
		return err
	}

	// info not found, nothing to stop
	if info == nil {
		return nil
	}

	// Do not watch over the process any more
	o.w.UnWatch(info.PID)

	// Remove client from the vault
	o.cv.UpdateClient(o.program.ID(), nil)

	// Kill the process
	process.Stop(o.logger, info.PID)

	// Remove process from collection, manager will not try to connect or manager it afterwards
	o.rc.removeProcess(info.PID)

	// remove generated configuration if present
	filename := fmt.Sprintf(configFileTempl, o.program.ID())
	filePath, err := filepath.Abs(filepath.Join(o.operatorConfig.DownloadConfig.InstallPath, filename))
	os.Remove(filePath)

	return nil
}

// Run runs the operation
func (o *operationStop) getReattachInfo() (*ReattachInfo, error) {
	if o.rc == nil {
		return nil, nil
	}

	rcItems, err := o.rc.items()
	if err != nil {
		return nil, err
	}

	id := o.program.ID()
	for _, item := range rcItems {
		p := NewProgramWithContext(item.ExecutionContext, nil)
		if p.ID() == id {
			return item, nil
		}
	}

	return nil, nil
}
