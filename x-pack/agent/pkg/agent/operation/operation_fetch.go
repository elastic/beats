// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"os"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/operation/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

// operationFetch fetches artifact from preconfigured source
// skips if artifact is already downloaded
type operationFetch struct {
	logger         *logger.Logger
	program        Descriptor
	operatorConfig *config.Config
	downloader     download.Downloader
	eventProcessor callbackHooks
}

func newOperationFetch(
	logger *logger.Logger,
	program Descriptor,
	operatorConfig *config.Config,
	downloader download.Downloader,
	eventProcessor callbackHooks) *operationFetch {

	return &operationFetch{
		logger:         logger,
		program:        program,
		operatorConfig: operatorConfig,
		downloader:     downloader,
		eventProcessor: eventProcessor,
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
func (o *operationFetch) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			err = errors.New(err,
				o.Name(),
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, application.Name()))
			o.eventProcessor.OnFailing(ctx, application.Name(), err)
		}
	}()

	fullPath, err := o.downloader.Download(ctx, o.program.BinaryName(), o.program.Version())
	if err == nil {
		o.logger.Infof("operation '%s' downloaded %s.%s into %s", o.Name(), o.program.BinaryName(), o.program.Version(), fullPath)
	}

	return err
}
