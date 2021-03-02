// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awaitable

import (
	"context"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

type embeddedInstaller interface {
	Install(ctx context.Context, spec program.Spec, version, installDir string) error
}

type embeddedChecker interface {
	Check(ctx context.Context, spec program.Spec, version, installDir string) error
}

// Installer installs into temporary destination and moves to correct one after
// successful finish.
type Installer struct {
	installer embeddedInstaller
	checker   embeddedChecker
	wg        sync.WaitGroup
}

// NewInstaller creates a new AtomicInstaller
func NewInstaller(i embeddedInstaller, ch embeddedChecker) (*Installer, error) {
	return &Installer{
		installer: i,
		checker:   ch,
	}, nil
}

// Wait allows caller to wait for install to be finished
func (i *Installer) Wait() {
	i.wg.Wait()
}

// Install performs installation of program in a specific version.
func (i *Installer) Install(ctx context.Context, spec program.Spec, version, installDir string) error {
	i.wg.Add(1)
	defer i.wg.Done()

	return i.installer.Install(ctx, spec, version, installDir)
}

// Check performs installation checks
func (i *Installer) Check(ctx context.Context, spec program.Spec, version, installDir string) error {
	i.wg.Add(1)
	defer i.wg.Done()

	return i.checker.Check(ctx, spec, version, installDir)
}
