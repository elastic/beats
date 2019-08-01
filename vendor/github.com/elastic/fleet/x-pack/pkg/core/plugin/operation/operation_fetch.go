// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"os"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
	"github.com/elastic/fleet/x-pack/pkg/artifact/download"
	"github.com/pkg/errors"
	"github.com/urso/ecslog"
)

// operationFetch fetches artifact from preconfigured source
// skips if artifact is already downloaded
type operationFetch struct {
	logger         *ecslog.Logger
	program        Program
	operatorConfig *Config
	downloader     download.Downloader
}

func newOperationFetch(
	logger *ecslog.Logger,
	program Program,
	operatorConfig *Config,
	downloader download.Downloader) *operationFetch {

	return &operationFetch{
		logger:         logger,
		program:        program,
		operatorConfig: operatorConfig,
		downloader:     downloader,
	}
}

// Name is human readable name identifying an operation
func (o *operationFetch) Name() string {
	return "operation-fetch"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationFetch) Check() (bool, error) {
	downloadConfig := o.operatorConfig.DownloadConfig
	fullPath, err := artifact.GetArtifactPath(o.program.BinaryName(), o.program.Version(), downloadConfig.OS(), downloadConfig.Arch(), downloadConfig.TargetDirectory)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if os.IsNotExist(err) {
		return true, nil
	}

	o.logger.Infof("%s.%s already exists in %s. Skipping operation %s", o.program.BinaryName(), o.program.Version(), fullPath, o.Name())
	return false, err
}

// Run runs the operation
func (o *operationFetch) Run() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, o.Name())
		}
	}()

	fullPath, err := o.downloader.Download(o.program.BinaryName(), o.program.Version())
	if err == nil {
		o.logger.Infof("operation '%s' downloaded %s.%s into %s", o.Name(), o.program.BinaryName(), o.program.Version(), fullPath)
	}

	return err
}
