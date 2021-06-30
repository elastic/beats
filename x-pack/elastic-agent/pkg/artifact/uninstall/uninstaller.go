// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package uninstall

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/uninstall/hooks"
)

// Uninstaller is an interface allowing un-installation of an artifact
type Uninstaller interface {
	// Uninstall uninstalls an artifact.
	Uninstall(ctx context.Context, spec program.Spec, version, installDir string) error
}

// NewUninstaller returns a correct uninstaller.
func NewUninstaller() (Uninstaller, error) {
	return hooks.NewUninstaller(&nilUninstaller{})
}

type nilUninstaller struct{}

func (*nilUninstaller) Uninstall(_ context.Context, _ program.Spec, _, _ string) error {
	return nil
}
