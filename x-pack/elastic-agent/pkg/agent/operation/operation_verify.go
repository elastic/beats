// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

// operationVerify verifies downloaded artifact for correct signature
// skips if artifact is already installed
type operationVerify struct {
	program        Descriptor
	operatorConfig *configuration.SettingsConfig
	verifier       download.Verifier
}

func newOperationVerify(
	program Descriptor,
	operatorConfig *configuration.SettingsConfig,
	verifier download.Verifier) *operationVerify {
	return &operationVerify{
		program:        program,
		operatorConfig: operatorConfig,
		verifier:       verifier,
	}
}

// Name is human readable name identifying an operation
func (o *operationVerify) Name() string {
	return "operation-verify"
}

// Check checks whether verify needs to occur.
//
// Only if the artifacts exists does it need to be verified.
func (o *operationVerify) Check(_ context.Context, _ Application) (bool, error) {
	downloadConfig := o.operatorConfig.DownloadConfig
	fullPath, err := artifact.GetArtifactPath(o.program.Spec(), o.program.Version(), downloadConfig.OS(), downloadConfig.Arch(), downloadConfig.TargetDirectory)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return false, errors.New(errors.TypeApplication,
			fmt.Sprintf("%s.%s package does not exist in %s. Skipping operation %s", o.program.BinaryName(), o.program.Version(), fullPath, o.Name()))
	}

	return true, err
}

// Run runs the operation
func (o *operationVerify) Run(_ context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			application.SetState(state.Failed, err.Error(), nil)
		}
	}()

	if err := o.verifier.Verify(o.program.Spec(), o.program.Version()); err != nil {
		return errors.New(err,
			fmt.Sprintf("operation '%s' failed to verify %s.%s", o.Name(), o.program.BinaryName(), o.program.Version()),
			errors.TypeSecurity)
	}

	return nil
}
